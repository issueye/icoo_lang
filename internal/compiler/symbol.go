package compiler

import (
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

type VarRefKind int

const (
	VarLocal VarRefKind = iota
	VarGlobal
	VarUpvalue
)

type VarRef struct {
	Kind  VarRefKind
	Index int
	Name  string
}

type UpvalueRef struct {
	Index   int
	IsLocal bool
}

func (fc *FuncCompiler) addUpvalue(parentSlot int, isLocal bool) int {
	for i, uv := range fc.upvalues {
		if uv.Index == parentSlot && uv.IsLocal == isLocal {
			return i
		}
	}
	idx := len(fc.upvalues)
	fc.upvalues = append(fc.upvalues, UpvalueRef{Index: parentSlot, IsLocal: isLocal})
	fc.proto.UpvalueCount = idx + 1
	return idx
}

func (c *Compiler) resolve(name string) (VarRef, bool) {
	for i := len(c.current.locals) - 1; i >= 0; i-- {
		local := c.current.locals[i]
		if local.Name == name {
			return VarRef{Kind: VarLocal, Index: local.Slot, Name: name}, true
		}
	}

	if c.current.parent != nil {
		idx, isLocal := c.resolveUpvalue(c.current, name)
		if idx >= 0 {
			uvIdx := c.current.addUpvalue(idx, isLocal)
			return VarRef{Kind: VarUpvalue, Index: uvIdx, Name: name}, true
		}
	}

	return VarRef{Kind: VarGlobal, Name: name}, true
}

func (c *Compiler) resolveUpvalue(fc *FuncCompiler, name string) (int, bool) {
	if fc.parent == nil {
		return -1, false
	}

	for i := len(fc.parent.locals) - 1; i >= 0; i-- {
		local := fc.parent.locals[i]
		if local.Name == name {
			return local.Slot, true
		}
	}

	if fc.parent.parent != nil {
		parentIdx, parentIsLocal := c.resolveUpvalue(fc.parent, name)
		if parentIdx >= 0 {
			uvIdx := fc.parent.addUpvalue(parentIdx, parentIsLocal)
			return uvIdx, false
		}
	}

	return -1, false
}

func (c *Compiler) emitClosure(protoConstIdx uint16) {
	c.emit(bytecode.OpClosure)
	c.emitShort(protoConstIdx)

	for _, uv := range c.current.upvalues {
		if uv.IsLocal {
			c.emitByte(1)
		} else {
			c.emitByte(0)
		}
		c.emitByte(byte(uv.Index))
	}
}

func (c *Compiler) compileClosureWiring(child *FuncCompiler) {
	closureValue := &runtime.Closure{Proto: child.proto}
	constIdx := c.current.chunk.AddConstant(closureValue)
	c.emit(bytecode.OpClosure)
	c.emitShort(constIdx)
	for _, uv := range child.upvalues {
		if uv.IsLocal {
			c.emitByte(1)
		} else {
			c.emitByte(0)
		}
		c.emitByte(byte(uv.Index))
	}
}
