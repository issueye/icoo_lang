package sema

type Symbol struct {
	Name    string
	IsConst bool
}

type Scope struct {
	parent  *Scope
	symbols map[string]Symbol
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]Symbol),
	}
}

func (s *Scope) Define(sym Symbol) bool {
	if _, exists := s.symbols[sym.Name]; exists {
		return false
	}
	s.symbols[sym.Name] = sym
	return true
}

func (s *Scope) Resolve(name string) (Symbol, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if sym, ok := scope.symbols[name]; ok {
			return sym, true
		}
	}
	return Symbol{}, false
}
