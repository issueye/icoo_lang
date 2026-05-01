package compiler

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

func (c *Compiler) compileDecl(decl ast.Decl) {
	c.withNodeLine(decl.Span().Start.Line, func() {
		switch d := decl.(type) {
		case *ast.VarDecl:
			c.compileVarDecl(d)
		case *ast.FnDecl:
			c.compileFnDecl(d)
		case *ast.ImportDecl:
			c.compileImportDecl(d)
		case *ast.ExportDecl:
			c.compileExportDecl(d)
		case *ast.DecoratedDecl:
			c.compileDecoratedDecl(d)
		case *ast.ClassDecl:
			c.compileClassDecl(d)
		case *ast.TypeDecl:
			c.compileTypeDecl(d)
		case *ast.InterfaceDecl:
			c.compileInterfaceDecl(d)
		default:
			c.errorf("unsupported declaration")
		}
	})
}

func (c *Compiler) compileVarDecl(d *ast.VarDecl) {
	c.compileExpr(d.Value)

	if c.current.scopeDepth > 0 {
		c.addLocal(d.Name, d.Kind == ast.ConstVar)
		return
	}

	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Name})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileFnDecl(d *ast.FnDecl) {
	child := newFuncCompiler(c.current, d.Name)
	child.proto.Arity = len(d.Params)
	prev := c.current
	c.current = child
	c.beginScope()

	for _, param := range d.Params {
		c.addLocal(param.Name, false)
	}
	if d.Body != nil {
		c.compileBlockStmt(d.Body, false)
	}
	c.emitNull()
	c.emit(bytecode.OpReturn)
	child.proto.LocalCount = len(child.locals)
	c.current = prev

	if len(child.upvalues) > 0 {
		c.compileClosureWiring(child)
	} else {
		protoValue := &runtime.Closure{Proto: child.proto}
		constIdx := c.current.chunk.AddConstant(protoValue)
		c.emit(bytecode.OpClosure)
		c.emitShort(constIdx)
	}

	if c.current.scopeDepth > 0 {
		c.addLocal(d.Name, true)
		return
	}

	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Name})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileImportDecl(d *ast.ImportDecl) {
	pathIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Path})
	c.emit(bytecode.OpImportModule)
	c.emitShort(pathIdx)

	if c.current.scopeDepth > 0 {
		c.addLocal(d.Alias, true)
		return
	}

	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Alias})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileExportDecl(d *ast.ExportDecl) {
	if d.Decl == nil {
		c.errorf("export requires declaration")
		return
	}
	c.compileDecl(d.Decl)

	name, err := exportDeclName(d.Decl)
	if err != nil {
		c.errorf("%s", err.Error())
		return
	}

	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: name})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(nameIdx)
	c.emit(bytecode.OpExport)
	c.emitShort(nameIdx)
}

func exportDeclName(decl ast.Decl) (string, error) {
	switch d := decl.(type) {
	case *ast.VarDecl:
		return d.Name, nil
	case *ast.FnDecl:
		return d.Name, nil
	case *ast.DecoratedDecl:
		return exportDeclName(d.Decl)
	case *ast.ClassDecl:
		return d.Name, nil
	case *ast.TypeDecl:
		return d.Name, nil
	case *ast.InterfaceDecl:
		return d.Name, nil
	default:
		return "", fmt.Errorf("unsupported export declaration")
	}
}

func (c *Compiler) compileDecoratedDecl(d *ast.DecoratedDecl) {
	if d == nil || d.Decl == nil {
		c.errorf("decorated declaration requires target")
		return
	}

	switch d.Decl.(type) {
	case *ast.FnDecl, *ast.ClassDecl:
	default:
		c.errorf("decorators only support functions and classes")
		return
	}

	c.compileDecl(d.Decl)

	name, err := exportDeclName(d.Decl)
	if err != nil {
		c.errorf("%s", err.Error())
		return
	}

	ref, _ := c.resolve(name)
	for i := len(d.Decorators) - 1; i >= 0; i-- {
		c.compileExpr(d.Decorators[i])
		c.emitNamedRefGet(ref)
		c.emit(bytecode.OpCall)
		c.emitByte(1)
		c.emitNamedRefSet(ref)
		c.emit(bytecode.OpPop)
	}
}

func (c *Compiler) emitNamedRefGet(ref VarRef) {
	if ref.Kind == VarLocal {
		c.emit(bytecode.OpGetLocal)
		c.emitShort(uint16(ref.Index))
		return
	}
	if ref.Kind == VarUpvalue {
		c.emit(bytecode.OpGetUpvalue)
		c.emitShort(uint16(ref.Index))
		return
	}
	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: ref.Name})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) emitNamedRefSet(ref VarRef) {
	if ref.Kind == VarLocal {
		c.emit(bytecode.OpSetLocal)
		c.emitShort(uint16(ref.Index))
		return
	}
	if ref.Kind == VarUpvalue {
		c.emit(bytecode.OpSetUpvalue)
		c.emitShort(uint16(ref.Index))
		return
	}
	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: ref.Name})
	c.emit(bytecode.OpSetGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileTypeDecl(d *ast.TypeDecl) {
	if c.current.scopeDepth > 0 {
		c.addLocal(d.Name, true)
		return
	}
	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Name})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileInterfaceDecl(d *ast.InterfaceDecl) {
	methods := make([]runtime.InterfaceMethodSig, 0, len(d.Methods))
	for _, m := range d.Methods {
		methods = append(methods, runtime.InterfaceMethodSig{
			Name:       m.Name,
			ParamCount: len(m.ParamTypes),
		})
	}
	ifaceValue := &runtime.InterfaceValue{Name: d.Name, Methods: methods}
	constIdx := c.current.chunk.AddConstant(ifaceValue)
	c.emit(bytecode.OpConstant)
	c.emitShort(constIdx)

	if c.current.scopeDepth > 0 {
		c.addLocal(d.Name, true)
		return
	}
	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Name})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileClassDecl(d *ast.ClassDecl) {
	hasSuper := d.Super != nil

	var initMethod *ast.ClassMethod
	for i := range d.Methods {
		if d.Methods[i].Name == "init" {
			initMethod = &d.Methods[i]
			break
		}
	}

	buildClassIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "__buildClass"})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(buildClassIdx)

	c.emitConstant(runtime.StringValue{Value: d.Name})
	if d.Super != nil {
		c.compileExpr(d.Super)
	} else {
		c.emitNull()
	}

	if initMethod != nil {
		c.compileClassMethod(initMethod, hasSuper)
	} else {
		c.emitNull()
	}

	for _, method := range d.Methods {
		if method.Name == "init" {
			continue
		}
		c.emitConstant(runtime.StringValue{Value: method.Name})
		c.compileClassMethod(&method, hasSuper)
	}
	c.emit(bytecode.OpObject)
	c.emitShort(uint16(len(d.Methods) - boolToInt(initMethod != nil)))

	c.emit(bytecode.OpCall)
	c.emitByte(4)

	if c.current.scopeDepth > 0 {
		c.addLocal(d.Name, true)
		return
	}

	nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: d.Name})
	c.emit(bytecode.OpDefineGlobal)
	c.emitShort(nameIdx)
}

func (c *Compiler) compileClassMethod(method *ast.ClassMethod, hasSuper bool) {
	methodChild := newFuncCompiler(c.current, method.Name)
	methodChild.proto.Arity = len(method.Params)

	prev := c.current
	c.current = methodChild
	c.beginScope()

	c.addLocal("this", false)
	if hasSuper {
		c.addLocal("super", true)
	}
	for _, param := range method.Params {
		c.addLocal(param.Name, false)
	}
	if method.Body != nil {
		c.compileBlockStmt(method.Body, false)
	}
	c.emitNull()
	c.emit(bytecode.OpReturn)
	methodChild.proto.LocalCount = len(methodChild.locals)
	c.current = prev

	if len(method.Decorators) > 0 {
		for _, decorator := range method.Decorators {
			c.compileExpr(decorator)
		}

		proxyIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "__methodProxy"})
		c.emit(bytecode.OpGetGlobal)
		c.emitShort(proxyIdx)
	}

	if len(methodChild.upvalues) > 0 {
		c.compileClosureWiring(methodChild)
	} else {
		protoValue := &runtime.Closure{Proto: methodChild.proto}
		constIdx := c.current.chunk.AddConstant(protoValue)
		c.emit(bytecode.OpClosure)
		c.emitShort(constIdx)
	}

	if len(method.Decorators) > 0 {
		c.emitConstant(runtime.StringValue{Value: method.Name})
		if method.Name == "init" {
			c.emit(bytecode.OpTrue)
		} else {
			c.emit(bytecode.OpFalse)
		}
		c.emit(bytecode.OpCall)
		c.emitByte(3)
		for range method.Decorators {
			c.emit(bytecode.OpCall)
			c.emitByte(1)
		}
	}

	methodDefIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "__methodDef"})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(methodDefIdx)
	c.emit(bytecode.OpSwap)
	if len(method.Decorators) > 0 {
		c.emit(bytecode.OpFalse)
	} else {
		c.emit(bytecode.OpTrue)
	}
	if method.Name == "init" {
		c.emit(bytecode.OpTrue)
	} else {
		c.emit(bytecode.OpFalse)
	}
	c.emitConstant(runtime.StringValue{Value: method.Name})
	c.emit(bytecode.OpCall)
	c.emitByte(4)

}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
