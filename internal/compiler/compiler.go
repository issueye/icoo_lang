package compiler

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

type Local struct {
	Name    string
	Depth   int
	Slot    int
	IsConst bool
}

type LoopContext struct {
	BreakJumps     []int
	ContinueTarget int
	ScopeDepth     int
}

type ExitActionKind int

const (
	CompletionKindNormal = 0
	CompletionKindException = 1
)

const (
	ExitActionReturn ExitActionKind = iota
	ExitActionBreak
	ExitActionContinue
	ExitActionException
)

type FinallyAction struct {
	Code           int
	Kind           ExitActionKind
	LoopIndex      int
	ContinueTarget int
	LoopScopeDepth int
}

type TryContext struct {
	ScopeDepth          int
	HandlerActive       bool
	FinallyBlock        bool
	InFinally           bool
	CompletionKindSlot  int
	CompletionValueSlot int
	FinallyJumpPatches  []int
	Actions             []FinallyAction
	NextActionCode      int
}

type CompiledModule struct {
	Proto *runtime.FunctionProto
	Chunk *bytecode.Chunk
}

type Compiler struct {
	errors      []error
	current     *FuncCompiler
	currentLine int
}

type FuncCompiler struct {
	parent     *FuncCompiler
	proto      *runtime.FunctionProto
	chunk      *bytecode.Chunk
	locals     []Local
	scopeDepth int
	loopStack  []LoopContext
	tryStack   []*TryContext
}

func Compile(program *ast.Program) (*CompiledModule, []error) {
	c := &Compiler{}
	fc := newFuncCompiler(nil, "__module_init__")
	c.current = fc

	for _, node := range program.Nodes {
		switch n := node.(type) {
		case ast.Decl:
			c.compileDecl(n)
		case ast.Stmt:
			c.compileStmt(n)
		default:
			c.errorf("unsupported top-level node")
		}
	}

	c.emitNull()
	c.emit(bytecode.OpReturn)

	module := &CompiledModule{
		Proto: fc.proto,
		Chunk: fc.chunk,
	}
	module.Proto.LocalCount = len(fc.locals)
	return module, c.errors
}

func newFuncCompiler(parent *FuncCompiler, name string) *FuncCompiler {
	chunk := bytecode.NewChunk()
	proto := &runtime.FunctionProto{
		Name:  name,
		Chunk: chunk,
	}
	fc := &FuncCompiler{
		parent: parent,
		proto:  proto,
		chunk:  chunk,
		locals: make([]Local, 0, 16),
	}
	if parent != nil {
		fc.locals = append(fc.locals, Local{
			Name:  "<fn>",
			Depth: 0,
			Slot:  0,
		})
		fc.proto.LocalCount = 1
	}
	return fc
}

func (c *Compiler) errorf(format string, args ...any) {
	c.errors = append(c.errors, fmt.Errorf(format, args...))
}

func (c *Compiler) emit(op bytecode.Opcode) {
	c.current.chunk.Write(byte(op), c.currentLine)
}

func (c *Compiler) emitByte(b byte) {
	c.current.chunk.Write(b, c.currentLine)
}

func (c *Compiler) emitShort(v uint16) {
	c.current.chunk.WriteShort(v, c.currentLine)
}

func (c *Compiler) emitConstant(v runtime.Value) uint16 {
	idx := c.current.chunk.AddConstant(v)
	c.emit(bytecode.OpConstant)
	c.emitShort(idx)
	return idx
}

func (c *Compiler) emitNull() {
	c.emit(bytecode.OpNull)
}

func (c *Compiler) emitJump(op bytecode.Opcode) int {
	c.emit(op)
	pos := len(c.current.chunk.Code)
	c.emitByte(0xff)
	c.emitByte(0xff)
	return pos
}

func (c *Compiler) patchAddress(pos int, value int) {
	c.current.chunk.Code[pos] = byte(value >> 8)
	c.current.chunk.Code[pos+1] = byte(value)
}

func (c *Compiler) patchJump(pos int) {
	jump := len(c.current.chunk.Code) - pos - 2
	c.current.chunk.Code[pos] = byte(jump >> 8)
	c.current.chunk.Code[pos+1] = byte(jump)
}

func (c *Compiler) emitLoop(loopStart int) {
	c.emit(bytecode.OpLoop)
	offset := len(c.current.chunk.Code) - loopStart + 2
	c.emitShort(uint16(offset))
}

func (c *Compiler) beginLoop(continueTarget int) {
	c.current.loopStack = append(c.current.loopStack, LoopContext{
		BreakJumps:     make([]int, 0, 4),
		ContinueTarget: continueTarget,
		ScopeDepth:     c.current.scopeDepth,
	})
}

func (c *Compiler) endLoop() LoopContext {
	idx := len(c.current.loopStack) - 1
	loop := c.current.loopStack[idx]
	c.current.loopStack = c.current.loopStack[:idx]
	return loop
}

func (c *Compiler) patchBreakJumps(loop LoopContext) {
	for _, jump := range loop.BreakJumps {
		c.patchJump(jump)
	}
}

func (c *Compiler) emitGetLocal(slot int) {
	c.emit(bytecode.OpGetLocal)
	c.emitShort(uint16(slot))
}

func (c *Compiler) emitSetLocal(slot int) {
	c.emit(bytecode.OpSetLocal)
	c.emitShort(uint16(slot))
}

func (c *Compiler) emitInt(value int64) {
	c.emitConstant(runtime.IntValue{Value: value})
}

func (c *Compiler) emitStoreTopToLocal(slot int) {
	c.emitSetLocal(slot)
	c.emit(bytecode.OpPop)
}

func (c *Compiler) emitStoreIntToLocal(slot int, value int) {
	c.emitInt(int64(value))
	c.emitStoreTopToLocal(slot)
}

func (c *Compiler) withLine(line int, fn func()) {
	prev := c.currentLine
	if line > 0 {
		c.currentLine = line
	}
	fn()
	c.currentLine = prev
}

func (c *Compiler) withNodeLine(spanLine int, fn func()) {
	c.withLine(spanLine, fn)
}

func (c *Compiler) patchAddressList(positions []int, value int) {
	for _, pos := range positions {
		c.patchAddress(pos, value)
	}
}

func (c *Compiler) patchJumpList(positions []int) {
	for _, pos := range positions {
		c.patchJump(pos)
	}
}

func (c *Compiler) currentFinallyContext(boundaryScopeDepth int) *TryContext {
	for i := len(c.current.tryStack) - 1; i >= 0; i-- {
		ctx := c.current.tryStack[i]
		if ctx.FinallyBlock && !ctx.InFinally && ctx.ScopeDepth > boundaryScopeDepth {
			return ctx
		}
	}
	return nil
}

func (c *Compiler) outerFinallyContext(current *TryContext, boundaryScopeDepth int) *TryContext {
	seenCurrent := false
	for i := len(c.current.tryStack) - 1; i >= 0; i-- {
		ctx := c.current.tryStack[i]
		if ctx == current {
			seenCurrent = true
			continue
		}
		if !seenCurrent {
			continue
		}
		if ctx.FinallyBlock && ctx.ScopeDepth > boundaryScopeDepth {
			return ctx
		}
	}
	return nil
}

func (c *Compiler) getOrAddFinallyAction(ctx *TryContext, kind ExitActionKind, loopIndex, continueTarget, loopScopeDepth int) int {
	for _, action := range ctx.Actions {
		if action.Kind == kind && action.LoopIndex == loopIndex && action.ContinueTarget == continueTarget && action.LoopScopeDepth == loopScopeDepth {
			return action.Code
		}
	}
	code := ctx.NextActionCode
	ctx.NextActionCode++
	ctx.Actions = append(ctx.Actions, FinallyAction{Code: code, Kind: kind, LoopIndex: loopIndex, ContinueTarget: continueTarget, LoopScopeDepth: loopScopeDepth})
	return code
}

func (c *Compiler) emitJumpToFinally(ctx *TryContext) {
	jump := c.emitJump(bytecode.OpJump)
	ctx.FinallyJumpPatches = append(ctx.FinallyJumpPatches, jump)
}

func (c *Compiler) emitExceptionScopeCleanup(scopeDepth int) {
	for i := len(c.current.tryStack) - 1; i >= 0; i-- {
		ctx := c.current.tryStack[i]
		if ctx.ScopeDepth <= scopeDepth {
			break
		}
		if ctx.HandlerActive {
			c.emit(bytecode.OpPopExceptionHandler)
		}
	}
}

func (c *Compiler) emitScopeCleanup(scopeDepth int) {
	for i := len(c.current.locals) - 1; i >= 0; i-- {
		local := c.current.locals[i]
		if local.Depth <= scopeDepth {
			break
		}
		c.emitPopLocal()
	}
}

func (c *Compiler) emitLoopScopeCleanup(scopeDepth int) {
	c.emitScopeCleanup(scopeDepth)
}
