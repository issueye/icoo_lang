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

type ImportDecl struct {
	Path       string
	Alias      string
	Specs      []ImportSpec
	FromImport bool
	Span_      token.Span
}

func (*ImportDecl) node() {}
func (*ImportDecl) decl() {}
func (d *ImportDecl) Span() token.Span {
	return d.Span_
}

type ImportSpec struct {
	Name  string
	Alias string
}

type ExportDecl struct {
	Decl  Decl
	Span_ token.Span
}

func (*ExportDecl) node() {}
func (*ExportDecl) decl() {}
func (d *ExportDecl) Span() token.Span {
	return d.Span_
}

type DecoratedDecl struct {
	Decl       Decl
	Decorators []Expr
	Span_      token.Span
}

func (*DecoratedDecl) node() {}
func (*DecoratedDecl) decl() {}
func (d *DecoratedDecl) Span() token.Span {
	return d.Span_
}

type ClassMethod struct {
	Name       string
	Params     []Param
	Body       *BlockStmt
	Decorators []Expr
	Span_      token.Span
}

func (m ClassMethod) Span() token.Span {
	return m.Span_
}

type ClassDecl struct {
	Name    string
	Super   Expr
	Methods []ClassMethod
	Span_   token.Span
}

func (*ClassDecl) node() {}
func (*ClassDecl) decl() {}
func (d *ClassDecl) Span() token.Span {
	return d.Span_
}

type TypeExpr interface {
	Node
	typeExpr()
}

type SimpleTypeExpr struct {
	Name  string
	Span_ token.Span
}

func (*SimpleTypeExpr) node()              {}
func (*SimpleTypeExpr) typeExpr()          {}
func (e *SimpleTypeExpr) Span() token.Span { return e.Span_ }

type FuncTypeExpr struct {
	Params []TypeExpr
	Return TypeExpr
	Span_  token.Span
}

func (*FuncTypeExpr) node()              {}
func (*FuncTypeExpr) typeExpr()          {}
func (e *FuncTypeExpr) Span() token.Span { return e.Span_ }

type TypeDecl struct {
	Name    string
	TypeDef TypeExpr
	Span_   token.Span
}

func (*TypeDecl) node()              {}
func (*TypeDecl) decl()              {}
func (d *TypeDecl) Span() token.Span { return d.Span_ }

type InterfaceMethod struct {
	Name       string
	ParamTypes []TypeExpr
	ReturnType TypeExpr
	Span_      token.Span
}

type InterfaceDecl struct {
	Name    string
	Methods []InterfaceMethod
	Span_   token.Span
}

func (*InterfaceDecl) node()              {}
func (*InterfaceDecl) decl()              {}
func (d *InterfaceDecl) Span() token.Span { return d.Span_ }
