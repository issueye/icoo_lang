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

type CompiledModule struct {
	Proto *runtime.FunctionProto
	Chunk *bytecode.Chunk
}

type Compiler struct {
	errors  []error
	current *FuncCompiler
}

type FuncCompiler struct {
	parent     *FuncCompiler
	proto      *runtime.FunctionProto
	chunk      *bytecode.Chunk
	locals     []Local
	scopeDepth int
	loopStack  []LoopContext
}

func Compile(program *ast.Program) (*CompiledModule, []error) {
	c := &Compiler{}
	fc := newFuncCompiler(nil, "__module_init__")
	c.current = fc

	for _, decl := range program.Decls {
		c.compileDecl(decl)
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
	return &FuncCompiler{
		parent: parent,
		proto:  proto,
		chunk:  chunk,
		locals: make([]Local, 0, 16),
	}
}

func (c *Compiler) errorf(format string, args ...any) {
	c.errors = append(c.errors, fmt.Errorf(format, args...))
}

func (c *Compiler) emit(op bytecode.Opcode) {
	c.current.chunk.Write(byte(op), 0)
}

func (c *Compiler) emitByte(b byte) {
	c.current.chunk.Write(b, 0)
}

func (c *Compiler) emitShort(v uint16) {
	c.current.chunk.WriteShort(v, 0)
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
