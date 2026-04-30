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
	case *ast.ForStmt:
		c.compileForStmt(s)
	case *ast.BreakStmt:
		c.compileBreakStmt(s)
	case *ast.ContinueStmt:
		c.compileContinueStmt(s)
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
	c.compileLoop(stmt.Cond, stmt.Body)
}

func (c *Compiler) compileForStmt(stmt *ast.ForStmt) {
	c.compileLoop(stmt.Cond, stmt.Body)
}

func (c *Compiler) compileLoop(cond ast.Expr, body *ast.BlockStmt) {
	loopStart := len(c.current.chunk.Code)
	exitJump := -1
	if cond != nil {
		c.compileExpr(cond)
		exitJump = c.emitJump(bytecode.OpJumpIfFalse)
		c.emit(bytecode.OpPop)
	}

	c.beginLoop(loopStart)
	c.compileBlockStmt(body, true)
	loop := c.endLoop()
	c.emitLoop(loop.ContinueTarget)

	if exitJump >= 0 {
		c.patchJump(exitJump)
		c.emit(bytecode.OpPop)
	}
	c.patchBreakJumps(loop)
}

func (c *Compiler) compileBreakStmt(_ *ast.BreakStmt) {
	if len(c.current.loopStack) == 0 {
		c.errorf("break used outside loop")
		return
	}
	loop := &c.current.loopStack[len(c.current.loopStack)-1]
	jump := c.emitJump(bytecode.OpJump)
	loop.BreakJumps = append(loop.BreakJumps, jump)
}

func (c *Compiler) compileContinueStmt(_ *ast.ContinueStmt) {
	if len(c.current.loopStack) == 0 {
		c.errorf("continue used outside loop")
		return
	}
	loop := c.current.loopStack[len(c.current.loopStack)-1]
	c.emitLoop(loop.ContinueTarget)
}
