package compiler

type VarRefKind int

const (
	VarLocal VarRefKind = iota
	VarGlobal
)

type VarRef struct {
	Kind  VarRefKind
	Index int
	Name  string
}

func (c *Compiler) resolve(name string) (VarRef, bool) {
	for i := len(c.current.locals) - 1; i >= 0; i-- {
		local := c.current.locals[i]
		if local.Name == name {
			return VarRef{Kind: VarLocal, Index: local.Slot, Name: name}, true
		}
	}
	return VarRef{Kind: VarGlobal, Name: name}, true
}
