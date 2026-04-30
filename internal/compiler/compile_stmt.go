package compiler

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

func (c *Compiler) compileStmt(stmt ast.Stmt) {
	c.withNodeLine(stmt.Span().Start.Line, func() {
		switch s := stmt.(type) {
		case *ast.DeclStmt:
			c.compileDecl(s.Decl)
		case *ast.BlockStmt:
			c.compileBlockStmt(s, true)
		case *ast.ExprStmt:
			c.compileExpr(s.Expr)
			c.emit(bytecode.OpPop)
		case *ast.ReturnStmt:
			c.compileReturnStmt(s)
		case *ast.ThrowStmt:
			c.compileThrowStmt(s)
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
		case *ast.GoStmt:
			c.compileGoStmt(s)
		case *ast.SelectStmt:
			c.compileSelectStmt(s)
		default:
			c.errorf("unsupported statement")
		}
	})
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
	if stmt.Finally == nil {
		c.compileTryCatchWithoutFinally(stmt)
		return
	}
	c.compileTryCatchWithFinally(stmt)
}

func (c *Compiler) compileTryCatchWithoutFinally(stmt *ast.TryCatchStmt) {
	c.emit(bytecode.OpPushExceptionHandler)
	catchAddrPos := len(c.current.chunk.Code)
	c.emitByte(0xff)
	c.emitByte(0xff)
	ctx := &TryContext{ScopeDepth: c.current.scopeDepth, HandlerActive: true}
	c.current.tryStack = append(c.current.tryStack, ctx)

	c.compileBlockStmt(stmt.Try, true)
	c.current.tryStack = c.current.tryStack[:len(c.current.tryStack)-1]
	if ctx.HandlerActive {
		c.emit(bytecode.OpPopExceptionHandler)
		ctx.HandlerActive = false
	}
	endJump := c.emitJump(bytecode.OpJump)

	catchTarget := len(c.current.chunk.Code)
	c.patchAddress(catchAddrPos, catchTarget)
	ctx.HandlerActive = false

	if stmt.Catch != nil {
		c.beginScope()
		if stmt.CatchName != "" {
			c.addLocal(stmt.CatchName, true)
		}
		c.compileBlockStmt(stmt.Catch, false)
		c.endScope()
	}

	c.patchJump(endJump)
}

func (c *Compiler) compileTryCatchWithFinally(stmt *ast.TryCatchStmt) {
	c.beginScope()
	c.emitInt(CompletionKindNormal)
	completionKindSlot := c.addLocal(c.syntheticName("finally_kind"), true)
	c.emitNull()
	completionValueSlot := c.addLocal(c.syntheticName("finally_value"), false)

	ctx := &TryContext{
		ScopeDepth:          c.current.scopeDepth,
		HandlerActive:       true,
		FinallyBlock:        true,
		CompletionKindSlot:  completionKindSlot,
		CompletionValueSlot: completionValueSlot,
		NextActionCode:      2,
	}

	c.emit(bytecode.OpPushExceptionHandler)
	handlerAddrPos := len(c.current.chunk.Code)
	c.emitByte(0xff)
	c.emitByte(0xff)
	c.current.tryStack = append(c.current.tryStack, ctx)

	c.compileBlockStmt(stmt.Try, true)
	if ctx.HandlerActive {
		c.emit(bytecode.OpPopExceptionHandler)
		ctx.HandlerActive = false
	}
	c.emitStoreIntToLocal(completionKindSlot, CompletionKindNormal)
	c.emitJumpToFinally(ctx)

	catchTarget := len(c.current.chunk.Code)
	c.patchAddress(handlerAddrPos, catchTarget)
	ctx.HandlerActive = false

	if stmt.Catch != nil {
		c.beginScope()
		if stmt.CatchName != "" {
			c.addLocal(stmt.CatchName, true)
		} else {
			c.emit(bytecode.OpPop)
		}
		c.compileBlockStmt(stmt.Catch, false)
		c.endScope()
		c.emitStoreIntToLocal(completionKindSlot, CompletionKindNormal)
		c.emitJumpToFinally(ctx)
	} else {
		c.emitStoreTopToLocal(completionValueSlot)
		c.emitStoreIntToLocal(completionKindSlot, CompletionKindException)
		c.emitJumpToFinally(ctx)
	}

	c.patchJumpList(ctx.FinallyJumpPatches)
	ctx.InFinally = true
	c.compileBlockStmt(stmt.Finally, true)
	ctx.InFinally = false

	c.current.tryStack = c.current.tryStack[:len(c.current.tryStack)-1]
	c.emitFinallyDispatch(ctx)
	c.endScope()
}

func (c *Compiler) compileThrowStmt(stmt *ast.ThrowStmt) {
	if stmt.Value != nil {
		c.compileExpr(stmt.Value)
	} else {
		c.emitNull()
	}
	if ctx := c.currentFinallyContext(-1); ctx != nil && !ctx.HandlerActive {
		c.emitExitThroughFinally(ctx, ExitActionException, -1, 0, -1, true)
		return
	}
	c.emit(bytecode.OpThrow)
}

func (c *Compiler) compileReturnStmt(stmt *ast.ReturnStmt) {
	ctx := c.currentFinallyContext(-1)
	if ctx == nil {
		if stmt.Value != nil {
			c.compileExpr(stmt.Value)
		} else {
			c.emitNull()
		}
		c.emitExceptionScopeCleanup(-1)
		c.emit(bytecode.OpReturn)
		return
	}

	if stmt.Value != nil {
		c.compileExpr(stmt.Value)
	} else {
		c.emitNull()
	}
	c.emitExitThroughFinally(ctx, ExitActionReturn, -1, 0, -1, true)
}

func (c *Compiler) compileBreakStmt(_ *ast.BreakStmt) {
	if len(c.current.loopStack) == 0 {
		c.errorf("break used outside loop")
		return
	}
	loopIndex := len(c.current.loopStack) - 1
	loop := c.current.loopStack[loopIndex]
	if ctx := c.currentFinallyContext(loop.ScopeDepth); ctx != nil {
		c.emitExitThroughFinally(ctx, ExitActionBreak, loopIndex, 0, loop.ScopeDepth, false)
		return
	}
	c.emitExceptionScopeCleanup(loop.ScopeDepth)
	c.emitLoopScopeCleanup(loop.ScopeDepth)
	jump := c.emitJump(bytecode.OpJump)
	c.current.loopStack[loopIndex].BreakJumps = append(c.current.loopStack[loopIndex].BreakJumps, jump)
}

func (c *Compiler) compileContinueStmt(_ *ast.ContinueStmt) {
	if len(c.current.loopStack) == 0 {
		c.errorf("continue used outside loop")
		return
	}
	loopIndex := len(c.current.loopStack) - 1
	loop := c.current.loopStack[loopIndex]
	if ctx := c.currentFinallyContext(loop.ScopeDepth); ctx != nil {
		c.emitExitThroughFinally(ctx, ExitActionContinue, loopIndex, loop.ContinueTarget, loop.ScopeDepth, false)
		return
	}
	c.emitExceptionScopeCleanup(loop.ScopeDepth)
	c.emitLoopScopeCleanup(loop.ScopeDepth)
	c.emitLoop(loop.ContinueTarget)
}

func (c *Compiler) emitExitThroughFinally(ctx *TryContext, kind ExitActionKind, loopIndex, continueTarget, loopScopeDepth int, valueOnStack bool) {
	if valueOnStack {
		c.emitStoreTopToLocal(ctx.CompletionValueSlot)
	}
	actionCode := c.getOrAddFinallyAction(ctx, kind, loopIndex, continueTarget, loopScopeDepth)
	c.emitStoreIntToLocal(ctx.CompletionKindSlot, actionCode)
	c.emitExceptionScopeCleanup(ctx.ScopeDepth)
	c.emitScopeCleanup(ctx.ScopeDepth)
	if ctx.HandlerActive {
		c.emit(bytecode.OpPopExceptionHandler)
	}
	c.emitJumpToFinally(ctx)
}

func (c *Compiler) emitFinallyDispatch(ctx *TryContext) {
	exceptionDone := c.emitJumpIfCompletionKindMismatch(ctx.CompletionKindSlot, CompletionKindException)
	c.emit(bytecode.OpPop)
	c.emitGetLocal(ctx.CompletionValueSlot)
	c.emit(bytecode.OpThrow)
	c.patchJump(exceptionDone)
	c.emit(bytecode.OpPop)

	for _, action := range ctx.Actions {
		next := c.emitJumpIfCompletionKindMismatch(ctx.CompletionKindSlot, action.Code)
		c.emit(bytecode.OpPop)
		c.emitDispatchFinallyAction(ctx, action)
		c.patchJump(next)
		c.emit(bytecode.OpPop)
	}
}

func (c *Compiler) emitDispatchFinallyAction(ctx *TryContext, action FinallyAction) {
	switch action.Kind {
	case ExitActionReturn:
		c.emitGetLocal(ctx.CompletionValueSlot)
		outer := c.outerFinallyContext(ctx, -1)
		if outer != nil {
			c.emitExitThroughFinally(outer, ExitActionReturn, -1, 0, -1, true)
			return
		}
		c.emitExceptionScopeCleanup(-1)
		c.emit(bytecode.OpReturn)
	case ExitActionException:
		c.emitGetLocal(ctx.CompletionValueSlot)
		outer := c.outerFinallyContext(ctx, -1)
		if outer != nil {
			c.emitExitThroughFinally(outer, ExitActionException, -1, 0, -1, true)
			return
		}
		c.emit(bytecode.OpThrow)
	case ExitActionBreak:
		if action.LoopIndex < 0 || action.LoopIndex >= len(c.current.loopStack) {
			c.errorf("internal compiler error: invalid break loop index")
			return
		}
		loop := &c.current.loopStack[action.LoopIndex]
		outer := c.outerFinallyContext(ctx, action.LoopScopeDepth)
		if outer != nil {
			c.emitExitThroughFinally(outer, ExitActionBreak, action.LoopIndex, 0, action.LoopScopeDepth, false)
			return
		}
		c.emitExceptionScopeCleanup(action.LoopScopeDepth)
		c.emitLoopScopeCleanup(action.LoopScopeDepth)
		jump := c.emitJump(bytecode.OpJump)
		loop.BreakJumps = append(loop.BreakJumps, jump)
	case ExitActionContinue:
		outer := c.outerFinallyContext(ctx, action.LoopScopeDepth)
		if outer != nil {
			c.emitExitThroughFinally(outer, ExitActionContinue, action.LoopIndex, action.ContinueTarget, action.LoopScopeDepth, false)
			return
		}
		c.emitExceptionScopeCleanup(action.LoopScopeDepth)
		c.emitLoopScopeCleanup(action.LoopScopeDepth)
		c.emitLoop(action.ContinueTarget)
	default:
		c.errorf("internal compiler error: unsupported finally action")
	}
}

func (c *Compiler) emitJumpIfCompletionKindMismatch(slot int, expected int) int {
	c.emitGetLocal(slot)
	c.emitInt(int64(expected))
	c.emit(bytecode.OpEqual)
	return c.emitJump(bytecode.OpJumpIfFalse)
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

func (c *Compiler) compileGoStmt(stmt *ast.GoStmt) {
	switch call := stmt.Expr.(type) {
	case *ast.CallExpr:
		c.compileExpr(call.Callee)
		for _, arg := range call.Args {
			c.compileExpr(arg)
		}
		c.emit(bytecode.OpGo)
		c.emitByte(byte(len(call.Args)))
	default:
		c.compileExpr(stmt.Expr)
		c.emit(bytecode.OpGo)
		c.emitByte(0)
	}
}

func (c *Compiler) compileSelectStmt(stmt *ast.SelectStmt) {
	selectResultName := c.syntheticName("select")
	c.beginScope()
	defer c.endScope()

	// Push __select builtin first (will be callee below cases array)
	selectNameIdx := c.current.chunk.AddConstant(runtime.StringValue{Value: "__select"})
	c.emit(bytecode.OpGetGlobal)
	c.emitShort(selectNameIdx)

	// Build cases array for __select
	for _, selCase := range stmt.Cases {
		switch selCase.Kind {
		case ast.SelectRecvCaseKind:
			c.emitStringConst("kind")
			c.emitStringConst("recv")
			c.emitStringConst("chan")
			c.compileExpr(selCase.Channel)
			c.emitStringConst("hasOk")
			if selCase.OkName != "" {
				c.emitInt(1)
			} else {
				c.emitInt(0)
			}
			c.emit(bytecode.OpObject)
			c.emitShort(3)
		case ast.SelectSendCaseKind:
			c.emitStringConst("kind")
			c.emitStringConst("send")
			c.emitStringConst("chan")
			c.compileExpr(selCase.Channel)
			c.emitStringConst("value")
			c.compileExpr(selCase.Value)
			c.emit(bytecode.OpObject)
			c.emitShort(3)
		case ast.SelectElseCaseKind:
			c.emitStringConst("kind")
			c.emitStringConst("else")
			c.emit(bytecode.OpObject)
			c.emitShort(1)
		}
	}

	// Build array
	c.emit(bytecode.OpArray)
	c.emitShort(uint16(len(stmt.Cases)))

	// Call __select(cases): callee is below array on stack
	c.emit(bytecode.OpCall)
	c.emitByte(1)

	// result = {index, value, ok} on stack
	c.addLocal(selectResultName, true)

	// Dispatch: for each case, check index and jump to body
	endJumps := make([]int, 0, len(stmt.Cases))
	for i, selCase := range stmt.Cases {
		nextCaseJump := -1
		if i < len(stmt.Cases)-1 {
			c.emit(bytecode.OpGetLocal)
			c.emitShort(uint16(c.mustResolveLocal(selectResultName)))
			idxProp := c.current.chunk.AddConstant(runtime.StringValue{Value: "index"})
			c.emit(bytecode.OpGetProperty)
			c.emitShort(idxProp)
			c.emitInt(int64(i))
			c.emit(bytecode.OpEqual)
			nextCaseJump = c.emitJump(bytecode.OpJumpIfFalse)
			c.emit(bytecode.OpPop)
		}

		// Compile case body with bindings
		c.beginScope()

		switch selCase.Kind {
		case ast.SelectRecvCaseKind:
			if selCase.BindName != "_" {
				c.emit(bytecode.OpGetLocal)
				c.emitShort(uint16(c.mustResolveLocal(selectResultName)))
				valProp := c.current.chunk.AddConstant(runtime.StringValue{Value: "value"})
				c.emit(bytecode.OpGetProperty)
				c.emitShort(valProp)
				c.addLocal(selCase.BindName, false)
			}
			if selCase.OkName != "" && selCase.OkName != "_" {
				c.emit(bytecode.OpGetLocal)
				c.emitShort(uint16(c.mustResolveLocal(selectResultName)))
				okProp := c.current.chunk.AddConstant(runtime.StringValue{Value: "ok"})
				c.emit(bytecode.OpGetProperty)
				c.emitShort(okProp)
				c.addLocal(selCase.OkName, false)
			}
		case ast.SelectSendCaseKind:
		case ast.SelectElseCaseKind:
		}

		// Compile case body with bindings
		c.beginScope()

		switch selCase.Kind {
		case ast.SelectRecvCaseKind:
			if selCase.BindName != "_" {
				c.emit(bytecode.OpGetLocal)
				c.emitShort(uint16(c.mustResolveLocal(selectResultName)))
				valProp := c.current.chunk.AddConstant(runtime.StringValue{Value: "value"})
				c.emit(bytecode.OpGetProperty)
				c.emitShort(valProp)
				c.addLocal(selCase.BindName, false)
			}
			if selCase.OkName != "" && selCase.OkName != "_" {
				c.emit(bytecode.OpGetLocal)
				c.emitShort(uint16(c.mustResolveLocal(selectResultName)))
				okProp := c.current.chunk.AddConstant(runtime.StringValue{Value: "ok"})
				c.emit(bytecode.OpGetProperty)
				c.emitShort(okProp)
				c.addLocal(selCase.OkName, false)
			}
		case ast.SelectSendCaseKind:
			// No bindings for send case
		case ast.SelectElseCaseKind:
			// No bindings for else case
		}

		c.compileBlockStmt(selCase.Body, false)
		c.endScope()

		endJumps = append(endJumps, c.emitJump(bytecode.OpJump))

		if nextCaseJump >= 0 {
			c.patchJump(nextCaseJump)
			c.emit(bytecode.OpPop)
		}
	}

	for _, jump := range endJumps {
		c.patchJump(jump)
	}
}

func (c *Compiler) emitStringConst(s string) {
	c.emitConstant(runtime.StringValue{Value: s})
}
