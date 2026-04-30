package compiler

import (
	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
)

func (c *Compiler) compileStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		c.compileDecl(s.Decl)
	case *ast.BlockStmt:
		c.compileBlockStmt(s, true)
	case *ast.ExprStmt:
		c.compileExpr(s.Expr)
		c.emit(bytecode.OpPop)
	case *ast.ReturnStmt:
		if s.Value != nil {
			c.compileExpr(s.Value)
		} else {
			c.emitNull()
		}
		c.emit(bytecode.OpReturn)
	case *ast.IfStmt:
		c.compileIfStmt(s)
	case *ast.WhileStmt:
		c.compileWhileStmt(s)
	default:
		c.errorf("unsupported statement")
	}
}

func (c *Compiler) compileBlockStmt(block *ast.BlockStmt, newScope bool) {
	if newScope {
		c.beginScope()
		defer c.endScope()
	}
	for _, stmt := range block.Stmts {
		c.compileStmt(stmt)
	}
}

func (c *Compiler) compileIfStmt(stmt *ast.IfStmt) {
	c.compileExpr(stmt.Cond)
	elseJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emit(bytecode.OpPop)
	c.compileBlockStmt(stmt.Then, true)
	endJump := c.emitJump(bytecode.OpJump)
	c.patchJump(elseJump)
	c.emit(bytecode.OpPop)
	if stmt.Else != nil {
		c.compileStmt(stmt.Else)
	}
	c.patchJump(endJump)
}

func (c *Compiler) compileWhileStmt(stmt *ast.WhileStmt) {
	loopStart := len(c.current.chunk.Code)
	c.compileExpr(stmt.Cond)
	exitJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emit(bytecode.OpPop)
	c.compileBlockStmt(stmt.Body, true)
	c.emitLoop(loopStart)
	c.patchJump(exitJump)
	c.emit(bytecode.OpPop)
}
