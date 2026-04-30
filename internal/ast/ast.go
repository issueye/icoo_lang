package ast

import "icoo_lang/internal/token"

type Node interface {
	node()
	Span() token.Span
}

type Decl interface {
	Node
	decl()
}

type Stmt interface {
	Node
	stmt()
}

type Expr interface {
	Node
	expr()
}

type Program struct {
	Decls []Decl
	Span_ token.Span
}

func (*Program) node() {}
func (p *Program) Span() token.Span {
	return p.Span_
}

func MergeSpan(start, end token.Span) token.Span {
	return token.Span{
		Start: start.Start,
		End:   end.End,
	}
}
