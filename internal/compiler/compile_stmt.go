package compiler

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
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
	case *ast.ForInStmt:
		c.compileForInStmt(s)
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

func (c *Compiler) compileForInStmt(stmt *ast.ForInStmt) {
	c.beginScope()
	defer c.endScope()

	iterName := c.syntheticName("iter")
	indexName := c.syntheticName("idx")

	c.compileExpr(stmt.Iterable)
	c.addLocal(iterName, true)
	c.emitConstant(runtime.IntValue{Value: -1})
	c.addLocal(indexName, false)

	loopStart := len(c.current.chunk.Code)

	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(c.mustResolveLocal(indexName)))
	c.emitConstant(runtime.IntValue{Value: 1})
	c.emit(bytecode.OpAdd)
	c.emit(bytecode.OpSetLocal)
	c.emitShort(uint16(c.mustResolveLocal(indexName)))
	c.emit(bytecode.OpPop)

	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(c.mustResolveLocal(indexName)))
	lenNameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "len"})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(lenNameIdx)
	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(c.mustResolveLocal(iterName)))
	c.emit(bytecode.OpCall)
	c.emitByte(1)
	c.emit(bytecode.OpLess)
	exitJump := c.emitJump(bytecode.OpJumpIfFalse)
	c.emit(bytecode.OpPop)

	c.beginLoop(loopStart)
	c.beginScope()
	if stmt.Name != "_" {
		c.emit(bytecode.OpGetLocal)
		c.emitShort(uint16(c.mustResolveLocal(iterName)))
		c.emit(bytecode.OpGetLocal)
		c.emitShort(uint16(c.mustResolveLocal(indexName)))
		c.emit(bytecode.OpGetIndex)
		c.addLocal(stmt.Name, false)
	}
	c.compileBlockStmt(stmt.Body, false)
	c.endScope()

	loop := c.endLoop()
	c.emitLoop(loop.ContinueTarget)
	c.patchJump(exitJump)
	c.emit(bytecode.OpPop)
	c.patchBreakJumps(loop)
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
	c.emitLoopScopeCleanup(loop.ScopeDepth)
	jump := c.emitJump(bytecode.OpJump)
	loop.BreakJumps = append(loop.BreakJumps, jump)
}

func (c *Compiler) compileContinueStmt(_ *ast.ContinueStmt) {
	if len(c.current.loopStack) == 0 {
		c.errorf("continue used outside loop")
		return
	}
	loop := c.current.loopStack[len(c.current.loopStack)-1]
	c.emitLoopScopeCleanup(loop.ScopeDepth)
	c.emitLoop(loop.ContinueTarget)
}

func (c *Compiler) syntheticName(prefix string) string {
	return fmt.Sprintf("<%s_%d>", prefix, len(c.current.locals))
}

func (c *Compiler) mustResolveLocal(name string) int {
	ref, ok := c.resolve(name)
	if !ok || ref.Kind != VarLocal {
		c.errorf("internal compiler error: local not found: %s", name)
		return 0
	}
	return ref.Index
}
