package ast

import (
	"icoo_lang/internal/token"
)

type IdentExpr struct {
	Name  string
	Span_ token.Span
}

func (*IdentExpr) node() {}
func (*IdentExpr) expr() {}
func (e *IdentExpr) Span() token.Span {
	return e.Span_
}

type IntLiteral struct {
	Raw   string
	Span_ token.Span
}

func (*IntLiteral) node() {}
func (*IntLiteral) expr() {}
func (e *IntLiteral) Span() token.Span {
	return e.Span_
}

type FloatLiteral struct {
	Raw   string
	Span_ token.Span
}

func (*FloatLiteral) node() {}
func (*FloatLiteral) expr() {}
func (e *FloatLiteral) Span() token.Span {
	return e.Span_
}

type StringLiteral struct {
	Raw   string
	Span_ token.Span
}

func (*StringLiteral) node() {}
func (*StringLiteral) expr() {}
func (e *StringLiteral) Span() token.Span {
	return e.Span_
}

type BoolLiteral struct {
	Value bool
	Span_ token.Span
}

func (*BoolLiteral) node() {}
func (*BoolLiteral) expr() {}
func (e *BoolLiteral) Span() token.Span {
	return e.Span_
}

type NullLiteral struct {
	Span_ token.Span
}

func (*NullLiteral) node() {}
func (*NullLiteral) expr() {}
func (e *NullLiteral) Span() token.Span {
	return e.Span_
}

type UnaryExpr struct {
	Op    token.Type
	Right Expr
	Span_ token.Span
}

func (*UnaryExpr) node() {}
func (*UnaryExpr) expr() {}
func (e *UnaryExpr) Span() token.Span {
	return e.Span_
}

type BinaryExpr struct {
	Left  Expr
	Op    token.Type
	Right Expr
	Span_ token.Span
}

func (*BinaryExpr) node() {}
func (*BinaryExpr) expr() {}
func (e *BinaryExpr) Span() token.Span {
	return e.Span_
}

type AssignExpr struct {
	Target Expr
	Op     token.Type
	Value  Expr
	Span_  token.Span
}

func (*AssignExpr) node() {}
func (*AssignExpr) expr() {}
func (e *AssignExpr) Span() token.Span {
	return e.Span_
}

type CallExpr struct {
	Callee Expr
	Args   []Expr
	Span_  token.Span
}

func (*CallExpr) node() {}
func (*CallExpr) expr() {}
func (e *CallExpr) Span() token.Span {
	return e.Span_
}

type MemberExpr struct {
	Object Expr
	Name   string
	Span_  token.Span
}

func (*MemberExpr) node() {}
func (*MemberExpr) expr() {}
func (e *MemberExpr) Span() token.Span {
	return e.Span_
}

type IndexExpr struct {
	Object Expr
	Index  Expr
	Span_  token.Span
}

func (*IndexExpr) node() {}
func (*IndexExpr) expr() {}
func (e *IndexExpr) Span() token.Span {
	return e.Span_
}

type ArrayLiteral struct {
	Items []Expr
	Span_ token.Span
}

func (*ArrayLiteral) node() {}
func (*ArrayLiteral) expr() {}
func (e *ArrayLiteral) Span() token.Span {
	return e.Span_
}

type ObjectField struct {
	Name  string
	Value Expr
	Span_ token.Span
}

type ObjectLiteral struct {
	Fields []ObjectField
	Span_  token.Span
}

func (*ObjectLiteral) node() {}
func (*ObjectLiteral) expr() {}
func (e *ObjectLiteral) Span() token.Span {
	return e.Span_
}

type FnExpr struct {
	Params []Param
	Body   *BlockStmt
	Span_  token.Span
}

func (*FnExpr) node() {}
func (*FnExpr) expr() {}
func (e *FnExpr) Span() token.Span {
	return e.Span_
}
