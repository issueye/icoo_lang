package compiler

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

func (c *Compiler) compileDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.VarDecl:
		c.compileVarDecl(d)
	case *ast.FnDecl:
		c.compileFnDecl(d)
	case *ast.ImportDecl:
		c.compileImportDecl(d)
	case *ast.ExportDecl:
		c.compileExportDecl(d)
	default:
		c.errorf("unsupported declaration")
	}
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

	protoValue := &runtime.Closure{Proto: child.proto}
	constIdx := c.current.chunk.AddConstant(protoValue)
	c.emit(bytecode.OpClosure)
	c.emitShort(constIdx)

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
		c.errorf(err.Error())
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
	default:
		return "", fmt.Errorf("unsupported export declaration")
	}
}
