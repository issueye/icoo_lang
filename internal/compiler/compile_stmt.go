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
		c.emitExceptionScopeCleanup(-1)
		c.emit(bytecode.OpReturn)
	case *ast.IfStmt:
		c.compileIfStmt(s)
	case *ast.WhileStmt:
		c.compileWhileStmt(s)
	case *ast.ForStmt:
		c.compileForStmt(s)
	case *ast.ForInStmt:
		c.compileForInStmt(s)
	case *ast.TryCatchStmt:
		c.compileTryCatchStmt(s)
	case *ast.MatchStmt:
		c.compileMatchStmt(s)
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
	stepName := c.syntheticName("step")

	c.compileExpr(stmt.Iterable)
	iterMethodIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "iter"})
	c.emit(bytecode.OpGetProperty)
	c.emitShort(iterMethodIdx)
	c.emit(bytecode.OpCall)
	c.emitByte(0)
	c.addLocal(iterName, true)

	loopStart := len(c.current.chunk.Code)
	c.beginLoop(loopStart)
	c.beginScope()

	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(c.mustResolveLocal(iterName)))
	nextMethodIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "next"})
	c.emit(bytecode.OpGetProperty)
	c.emitShort(nextMethodIdx)
	c.emit(bytecode.OpCall)
	c.emitByte(0)
	c.addLocal(stepName, true)

	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(c.mustResolveLocal(stepName)))
	doneNameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "done"})
	c.emit(bytecode.OpGetProperty)
	c.emitShort(doneNameIdx)
	exitJump := c.emitJump(bytecode.OpJumpIfTrue)
	c.emit(bytecode.OpPop)

	if stmt.ValueName != "" {
		if stmt.Name != "_" {
			c.emit(bytecode.OpGetLocal)
			c.emitShort(uint16(c.mustResolveLocal(stepName)))
			keyNameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "key"})
			c.emit(bytecode.OpGetProperty)
			c.emitShort(keyNameIdx)
			c.addLocal(stmt.Name, false)
		}
		if stmt.ValueName != "_" {
			c.emit(bytecode.OpGetLocal)
			c.emitShort(uint16(c.mustResolveLocal(stepName)))
			valueFieldIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "value"})
			c.emit(bytecode.OpGetProperty)
			c.emitShort(valueFieldIdx)
			c.addLocal(stmt.ValueName, false)
		}
	} else if stmt.Name != "_" {
		c.emit(bytecode.OpGetLocal)
		c.emitShort(uint16(c.mustResolveLocal(stepName)))
		itemNameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "item"})
		c.emit(bytecode.OpGetProperty)
		c.emitShort(itemNameIdx)
		c.addLocal(stmt.Name, false)
	}

	exitCleanupCount := c.localsAboveDepth(c.current.loopStack[len(c.current.loopStack)-1].ScopeDepth)
	c.compileBlockStmt(stmt.Body, false)
	c.endScope()

	loop := c.endLoop()
	c.emitLoop(loop.ContinueTarget)
	c.patchJump(exitJump)
	for i := 0; i < exitCleanupCount; i++ {
		c.emit(bytecode.OpPop)
	}
	c.emit(bytecode.OpPop)
	c.patchBreakJumps(loop)
}

func (c *Compiler) compileMatchStmt(stmt *ast.MatchStmt) {
	matchName := c.syntheticName("match")
	c.beginScope()
	defer c.endScope()
	c.compileExpr(stmt.Value)
	c.addLocal(matchName, true)

	endJumps := make([]int, 0, len(stmt.Arms))
	for _, arm := range stmt.Arms {
		var nextArmJump int
		if !arm.IsWildcard {
			c.emit(bytecode.OpGetLocal)
			c.emitShort(uint16(c.mustResolveLocal(matchName)))
			c.compileExpr(arm.Pattern)
			c.emit(bytecode.OpEqual)
			nextArmJump = c.emitJump(bytecode.OpJumpIfFalse)
			c.emit(bytecode.OpPop)
		}

		c.compileBlockStmt(arm.Body, true)
		endJumps = append(endJumps, c.emitJump(bytecode.OpJump))

		if !arm.IsWildcard {
			c.patchJump(nextArmJump)
			c.emit(bytecode.OpPop)
		}
	}

	for _, jump := range endJumps {
		c.patchJump(jump)
	}
}

func (c *Compiler) compileTryCatchStmt(stmt *ast.TryCatchStmt) {
	c.emit(bytecode.OpPushExceptionHandler)
	catchAddrPos := len(c.current.chunk.Code)
	c.emitByte(0xff)
	c.emitByte(0xff)
	c.current.tryStack = append(c.current.tryStack, TryContext{ScopeDepth: c.current.scopeDepth})

	c.compileBlockStmt(stmt.Try, true)
	c.current.tryStack = c.current.tryStack[:len(c.current.tryStack)-1]
	c.emit(bytecode.OpPopExceptionHandler)
	endJump := c.emitJump(bytecode.OpJump)

	catchTarget := len(c.current.chunk.Code)
	c.patchAddress(catchAddrPos, catchTarget)

	c.beginScope()
	if stmt.CatchName != "" {
		c.addLocal(stmt.CatchName, true)
	}
	c.compileBlockStmt(stmt.Catch, false)
	c.endScope()

	c.patchJump(endJump)
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
	c.emitExceptionScopeCleanup(loop.ScopeDepth)
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
	c.emitExceptionScopeCleanup(loop.ScopeDepth)
	c.emitLoopScopeCleanup(loop.ScopeDepth)
	c.emitLoop(loop.ContinueTarget)
}

func (c *Compiler) localsAboveDepth(depth int) int {
	count := 0
	for i := len(c.current.locals) - 1; i >= 0; i-- {
		local := c.current.locals[i]
		if local.Depth <= depth {
			break
		}
		count++
	}
	return count
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
