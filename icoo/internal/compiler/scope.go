package compiler

import "icoo_lang/internal/bytecode"

func (c *Compiler) beginScope() {
	c.current.scopeDepth++
}

func (c *Compiler) endScope() {
	c.current.scopeDepth--
	for len(c.current.locals) > 0 {
		last := c.current.locals[len(c.current.locals)-1]
		if last.Depth <= c.current.scopeDepth {
			break
		}
		c.emitPopLocal()
		c.current.locals = c.current.locals[:len(c.current.locals)-1]
	}
}

func (c *Compiler) addLocal(name string, isConst bool) int {
	slot := len(c.current.locals)
	c.current.locals = append(c.current.locals, Local{
		Name:    name,
		Depth:   c.current.scopeDepth,
		Slot:    slot,
		IsConst: isConst,
	})
	if slot+1 > c.current.proto.LocalCount {
		c.current.proto.LocalCount = slot + 1
	}
	return slot
}

func (c *Compiler) emitPopLocal() {
	c.emit(bytecode.OpPop)
}
