package sema

import (
	"icoo_lang/internal/ast"
	"icoo_lang/internal/diag"
	"icoo_lang/internal/token"
)

type Analyzer struct {
	diagnostics []diag.Diagnostic
	scope       *Scope
	inFunction  int
	loopDepth   int
}

func Analyze(program *ast.Program) []diag.Diagnostic {
	a := &Analyzer{
		scope: NewScope(nil),
	}
	a.defineBuiltins()
	a.visitProgram(program)
	return a.diagnostics
}

func (a *Analyzer) defineBuiltins() {
	builtins := []string{"print", "println", "len", "typeOf", "chan", "panic"}
	for _, name := range builtins {
		a.scope.Define(Symbol{Name: name})
	}
}

func (a *Analyzer) visitProgram(program *ast.Program) {
	for _, node := range program.Nodes {
		switch n := node.(type) {
		case ast.Decl:
			a.visitDecl(n)
		case ast.Stmt:
			a.visitStmt(n)
		}
	}
}

func (a *Analyzer) visitDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.VarDecl:
		a.visitVarDecl(d)
	case *ast.FnDecl:
		a.visitFnDecl(d)
	case *ast.ImportDecl:
		a.visitImportDecl(d)
	case *ast.ExportDecl:
		a.visitExportDecl(d)
	}
}

func (a *Analyzer) visitVarDecl(d *ast.VarDecl) {
	if d.Value != nil {
		a.visitExpr(d.Value)
	}
	if !a.scope.Define(Symbol{Name: d.Name, IsConst: d.Kind == ast.ConstVar}) {
		a.report(d.Span(), "duplicate declaration: "+d.Name)
	}
}

func (a *Analyzer) visitFnDecl(d *ast.FnDecl) {
	if !a.scope.Define(Symbol{Name: d.Name, IsConst: true}) {
		a.report(d.Span(), "duplicate declaration: "+d.Name)
		return
	}

	prevScope := a.scope
	a.scope = NewScope(prevScope)
	a.inFunction++
	defer func() {
		a.inFunction--
		a.scope = prevScope
	}()

	for _, param := range d.Params {
		if !a.scope.Define(Symbol{Name: param.Name}) {
			a.report(param.Span(), "duplicate parameter: "+param.Name)
		}
	}

	if d.Body != nil {
		a.visitBlockStmt(d.Body)
	}
}

func (a *Analyzer) visitImportDecl(d *ast.ImportDecl) {
	if !a.scope.Define(Symbol{Name: d.Alias, IsConst: true}) {
		a.report(d.Span(), "duplicate declaration: "+d.Alias)
	}
}

func (a *Analyzer) visitExportDecl(d *ast.ExportDecl) {
	if d.Decl != nil {
		a.visitDecl(d.Decl)
	}
}

func (a *Analyzer) visitStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		a.visitDecl(s.Decl)
	case *ast.BlockStmt:
		a.visitNestedBlockStmt(s)
	case *ast.ExprStmt:
		a.visitExpr(s.Expr)
	case *ast.ReturnStmt:
		if a.inFunction == 0 {
			a.report(s.Span(), "return used outside function")
		}
		if s.Value != nil {
			a.visitExpr(s.Value)
		}
	case *ast.IfStmt:
		a.visitExpr(s.Cond)
		if s.Then != nil {
			a.visitNestedBlockStmt(s.Then)
		}
		if s.Else != nil {
			a.visitStmt(s.Else)
		}
	case *ast.WhileStmt:
		a.visitLoopStmt(s.Cond, s.Body)
	case *ast.ForStmt:
		a.visitLoopStmt(s.Cond, s.Body)
	case *ast.ForInStmt:
		a.visitForInStmt(s)
	case *ast.BreakStmt:
		if a.loopDepth == 0 {
			a.report(s.Span(), "break used outside loop")
		}
	case *ast.ContinueStmt:
		if a.loopDepth == 0 {
			a.report(s.Span(), "continue used outside loop")
		}
	}
}

func (a *Analyzer) visitLoopStmt(cond ast.Expr, body *ast.BlockStmt) {
	if cond != nil {
		a.visitExpr(cond)
	}
	if body != nil {
		a.loopDepth++
		defer func() { a.loopDepth-- }()
		a.visitNestedBlockStmt(body)
	}
}

func (a *Analyzer) visitForInStmt(stmt *ast.ForInStmt) {
	if stmt.Iterable != nil {
		a.visitExpr(stmt.Iterable)
	}
	prevScope := a.scope
	a.scope = NewScope(prevScope)
	defer func() { a.scope = prevScope }()
	if stmt.Name != "_" {
		a.scope.Define(Symbol{Name: stmt.Name})
	}
	a.loopDepth++
	defer func() { a.loopDepth-- }()
	if stmt.Body != nil {
		a.visitBlockStmt(stmt.Body)
	}
}

func (a *Analyzer) visitBlockStmt(block *ast.BlockStmt) {
	for _, stmt := range block.Stmts {
		a.visitStmt(stmt)
	}
}

func (a *Analyzer) visitNestedBlockStmt(block *ast.BlockStmt) {
	prevScope := a.scope
	a.scope = NewScope(prevScope)
	defer func() { a.scope = prevScope }()
	a.visitBlockStmt(block)
}

func (a *Analyzer) visitExpr(expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		if _, ok := a.scope.Resolve(e.Name); !ok {
			a.report(e.Span(), "undefined identifier: "+e.Name)
		}
	case *ast.IntLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BoolLiteral, *ast.NullLiteral:
		return
	case *ast.UnaryExpr:
		a.visitExpr(e.Right)
	case *ast.BinaryExpr:
		a.visitExpr(e.Left)
		a.visitExpr(e.Right)
	case *ast.AssignExpr:
		a.visitAssignExpr(e)
	case *ast.CallExpr:
		a.visitExpr(e.Callee)
		for _, arg := range e.Args {
			a.visitExpr(arg)
		}
	case *ast.MemberExpr:
		a.visitExpr(e.Object)
	case *ast.IndexExpr:
		a.visitExpr(e.Object)
		a.visitExpr(e.Index)
	case *ast.ArrayLiteral:
		for _, item := range e.Items {
			a.visitExpr(item)
		}
	case *ast.ObjectLiteral:
		for _, field := range e.Fields {
			a.visitExpr(field.Value)
		}
	case *ast.FnExpr:
		a.visitFnExpr(e)
	}
}

func (a *Analyzer) visitAssignExpr(e *ast.AssignExpr) {
	switch target := e.Target.(type) {
	case *ast.IdentExpr:
		sym, ok := a.scope.Resolve(target.Name)
		if !ok {
			a.report(target.Span(), "undefined identifier: "+target.Name)
		} else if sym.IsConst {
			a.report(target.Span(), "cannot assign to const: "+target.Name)
		}
	case *ast.MemberExpr:
		a.visitExpr(target.Object)
	case *ast.IndexExpr:
		a.visitExpr(target.Object)
		a.visitExpr(target.Index)
	default:
		a.report(e.Span(), "invalid assignment target")
	}
	if e.Value != nil {
		a.visitExpr(e.Value)
	}
}

func (a *Analyzer) visitFnExpr(e *ast.FnExpr) {
	prevScope := a.scope
	a.scope = NewScope(prevScope)
	a.inFunction++
	defer func() {
		a.inFunction--
		a.scope = prevScope
	}()

	for _, param := range e.Params {
		if !a.scope.Define(Symbol{Name: param.Name}) {
			a.report(param.Span(), "duplicate parameter: "+param.Name)
		}
	}

	if e.Body != nil {
		a.visitBlockStmt(e.Body)
	}
}

func (a *Analyzer) report(span token.Span, message string) {
	a.diagnostics = append(a.diagnostics, diag.Diagnostic{
		Severity: diag.Error,
		Message:  message,
		Span:     span,
	})
}
