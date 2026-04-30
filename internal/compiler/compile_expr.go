package compiler

import (
	"strconv"
	"strings"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
	"icoo_lang/internal/token"
)

func (c *Compiler) compileExpr(expr ast.Expr) {
	c.withNodeLine(expr.Span().Start.Line, func() {
		switch e := expr.(type) {
		case *ast.IdentExpr:
			c.compileIdentExpr(e)
		case *ast.IntLiteral:
			v, err := strconv.ParseInt(e.Raw, 10, 64)
			if err != nil {
				c.errorf("invalid int literal: %s", e.Raw)
				c.emitNull()
				return
			}
			c.emitConstant(runtime.IntValue{Value: v})
		case *ast.FloatLiteral:
			v, err := strconv.ParseFloat(e.Raw, 64)
			if err != nil {
				c.errorf("invalid float literal: %s", e.Raw)
				c.emitNull()
				return
			}
			c.emitConstant(runtime.FloatValue{Value: v})
		case *ast.StringLiteral:
			raw := strings.Trim(e.Raw, "\"")
			c.emitConstant(runtime.StringValue{Value: raw})
		case *ast.BoolLiteral:
			if e.Value {
				c.emit(bytecode.OpTrue)
			} else {
				c.emit(bytecode.OpFalse)
			}
		case *ast.NullLiteral:
			c.emitNull()
		case *ast.UnaryExpr:
			c.compileExpr(e.Right)
			switch e.Op {
			case token.Minus:
				c.emit(bytecode.OpNegate)
			case token.Bang:
				c.emit(bytecode.OpNot)
			default:
				c.errorf("unsupported unary operator")
			}
		case *ast.BinaryExpr:
			if e.Op == token.AndAnd || e.Op == token.OrOr {
				c.compileLogicalExpr(e)
				return
			}
			c.compileExpr(e.Left)
			c.compileExpr(e.Right)
			c.compileBinaryOp(e.Op)
		case *ast.AssignExpr:
			c.compileAssignExpr(e)
		case *ast.CallExpr:
			c.compileExpr(e.Callee)
			for _, arg := range e.Args {
				c.compileExpr(arg)
			}
			c.emit(bytecode.OpCall)
			c.emitByte(byte(len(e.Args)))
		case *ast.MemberExpr:
			c.compileExpr(e.Object)
			nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: e.Name})
			c.emit(bytecode.OpGetProperty)
			c.emitShort(nameIdx)
		case *ast.IndexExpr:
			c.compileExpr(e.Object)
			c.compileExpr(e.Index)
			c.emit(bytecode.OpGetIndex)
		case *ast.ArrayLiteral:
			for _, item := range e.Items {
				c.compileExpr(item)
			}
			c.emit(bytecode.OpArray)
			c.emitShort(uint16(len(e.Items)))
		case *ast.ObjectLiteral:
			for _, field := range e.Fields {
				c.emitConstant(runtime.StringValue{Value: field.Name})
				c.compileExpr(field.Value)
			}
			c.emit(bytecode.OpObject)
			c.emitShort(uint16(len(e.Fields)))
		case *ast.FnExpr:
			c.compileFnExprExpr(e)
		case *ast.ThisExpr:
			ref, _ := c.resolve("this")
			if ref.Kind == VarLocal {
				c.emit(bytecode.OpGetLocal)
				c.emitShort(uint16(ref.Index))
			} else if ref.Kind == VarUpvalue {
				c.emit(bytecode.OpGetUpvalue)
				c.emitShort(uint16(ref.Index))
			} else {
				c.errorf("this used outside class context")
				c.emitNull()
			}
		case *ast.TryExpr:
			c.compileTryExpr(e)
		default:
			c.errorf("unsupported expression")
			c.emitNull()
		}
	})
}

func (c *Compiler) compileIdentExpr(e *ast.IdentExpr) {
	ref, _ := c.resolve(e.Name)
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

func (c *Compiler) compileAssignExpr(e *ast.AssignExpr) {
	c.compileExpr(e.Value)
	switch target := e.Target.(type) {
	case *ast.IdentExpr:
		ref, _ := c.resolve(target.Name)
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
	case *ast.MemberExpr:
		c.compileExpr(target.Object)
		nameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: target.Name})
		c.emit(bytecode.OpSetProperty)
		c.emitShort(nameIdx)
	case *ast.IndexExpr:
		c.compileExpr(target.Object)
		c.compileExpr(target.Index)
		c.emit(bytecode.OpSetIndex)
	default:
		c.errorf("invalid assignment target")
	}
}

func (c *Compiler) compileBinaryOp(op token.Type) {
	switch op {
	case token.Plus:
		c.emit(bytecode.OpAdd)
	case token.Minus:
		c.emit(bytecode.OpSub)
	case token.Star:
		c.emit(bytecode.OpMul)
	case token.Slash:
		c.emit(bytecode.OpDiv)
	case token.Percent:
		c.emit(bytecode.OpMod)
	case token.Eq:
		c.emit(bytecode.OpEqual)
	case token.Neq:
		c.emit(bytecode.OpNotEqual)
	case token.Gt:
		c.emit(bytecode.OpGreater)
	case token.Gte:
		c.emit(bytecode.OpGreaterEqual)
	case token.Lt:
		c.emit(bytecode.OpLess)
	case token.Lte:
		c.emit(bytecode.OpLessEqual)
	default:
		c.errorf("unsupported binary operator")
	}
}

func (c *Compiler) compileLogicalExpr(e *ast.BinaryExpr) {
	c.compileExpr(e.Left)

	if e.Op == token.AndAnd {
		c.emit(bytecode.OpDup)
		endJump := c.emitJump(bytecode.OpJumpIfFalse)
		c.emit(bytecode.OpPop)
		c.compileExpr(e.Right)
		c.patchJump(endJump)
	} else {
		c.emit(bytecode.OpDup)
		endJump := c.emitJump(bytecode.OpJumpIfTrue)
		c.emit(bytecode.OpPop)
		c.compileExpr(e.Right)
		c.patchJump(endJump)
	}
}

func (c *Compiler) compileTryExpr(e *ast.TryExpr) {
	c.compileExpr(e.Expr)

	c.beginScope()
	trySlot := c.addLocal(c.syntheticName("try"), false)

	// _tryCheck(val) → returns true if val is ErrorValue
	checkIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "_tryCheck"})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(checkIdx)
	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(trySlot))
	c.emit(bytecode.OpCall)
	c.emitByte(1)

	// If true (is error), return the error
	errorJump := c.emitJump(bytecode.OpJumpIfTrue)
	c.emit(bytecode.OpPop)
	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(trySlot))
	c.emit(bytecode.OpReturn)
	c.patchJump(errorJump)
	c.emit(bytecode.OpPop)

	c.current.locals = c.current.locals[:len(c.current.locals)-1]
	c.current.scopeDepth--
}

func (c *Compiler) compileFnExprExpr(e *ast.FnExpr) {
	child := newFuncCompiler(c.current, "")
	child.proto.Arity = len(e.Params)
	prev := c.current
	c.current = child

	for _, param := range e.Params {
		c.addLocal(param.Name, false)
	}
	if e.Body != nil {
		c.compileBlockStmt(e.Body, false)
	}
	c.emitNull()
	c.emit(bytecode.OpReturn)
	child.proto.LocalCount = len(child.locals)
	c.current = prev

	if len(child.upvalues) > 0 {
		c.compileClosureWiring(child)
	} else {
		constIdx := c.current.chunk.AddConstant(&runtime.Closure{Proto: child.proto})
		c.emit(bytecode.OpClosure)
		c.emitShort(constIdx)
	}
}
