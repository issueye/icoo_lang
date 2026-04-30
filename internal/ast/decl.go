package ast

import "icoo_lang/internal/token"

type VarKind int

const (
	ConstVar VarKind = iota
	LetVar
)

type Param struct {
	Name  string
	Span_ token.Span
}

func (p Param) Span() token.Span {
	return p.Span_
}

type VarDecl struct {
	Kind  VarKind
	Name  string
	Value Expr
	Span_ token.Span
}

func (*VarDecl) node() {}
func (*VarDecl) decl() {}
func (d *VarDecl) Span() token.Span {
	return d.Span_
}

type FnDecl struct {
	Name   string
	Params []Param
	Body   *BlockStmt
	Span_  token.Span
}

func (*FnDecl) node() {}
func (*FnDecl) decl() {}
func (d *FnDecl) Span() token.Span {
	return d.Span_
}
