package ast

import "icoo_lang/internal/token"

type DeclStmt struct {
	Decl  Decl
	Span_ token.Span
}

func (*DeclStmt) node() {}
func (*DeclStmt) stmt() {}
func (s *DeclStmt) Span() token.Span {
	return s.Span_
}

type BlockStmt struct {
	Stmts []Stmt
	Span_ token.Span
}

func (*BlockStmt) node() {}
func (*BlockStmt) stmt() {}
func (s *BlockStmt) Span() token.Span {
	return s.Span_
}

type ExprStmt struct {
	Expr  Expr
	Span_ token.Span
}

func (*ExprStmt) node() {}
func (*ExprStmt) stmt() {}
func (s *ExprStmt) Span() token.Span {
	return s.Span_
}

type ReturnStmt struct {
	Value Expr
	Span_ token.Span
}

func (*ReturnStmt) node() {}
func (*ReturnStmt) stmt() {}
func (s *ReturnStmt) Span() token.Span {
	return s.Span_
}

type IfStmt struct {
	Cond Expr
	Then *BlockStmt
	Else Stmt
	Span_ token.Span
}

func (*IfStmt) node() {}
func (*IfStmt) stmt() {}
func (s *IfStmt) Span() token.Span {
	return s.Span_
}

type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
	Span_ token.Span
}

func (*WhileStmt) node() {}
func (*WhileStmt) stmt() {}
func (s *WhileStmt) Span() token.Span {
	return s.Span_
}

type ForStmt struct {
	Cond Expr
	Body *BlockStmt
	Span_ token.Span
}

func (*ForStmt) node() {}
func (*ForStmt) stmt() {}
func (s *ForStmt) Span() token.Span {
	return s.Span_
}

type ForInStmt struct {
	Name     string
	Iterable Expr
	Body     *BlockStmt
	Span_    token.Span
}

func (*ForInStmt) node() {}
func (*ForInStmt) stmt() {}
func (s *ForInStmt) Span() token.Span {
	return s.Span_
}

type BreakStmt struct {
	Span_ token.Span
}

func (*BreakStmt) node() {}
func (*BreakStmt) stmt() {}
func (s *BreakStmt) Span() token.Span {
	return s.Span_
}

type ContinueStmt struct {
	Span_ token.Span
}

func (*ContinueStmt) node() {}
func (*ContinueStmt) stmt() {}
func (s *ContinueStmt) Span() token.Span {
	return s.Span_
}
