package parser

import (
	"path/filepath"
	"strings"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/token"
)

func (p *Parser) parseTopLevelNode() ast.Node {
	switch p.current().Type {
	case token.Const, token.Let:
		return p.parseVarDecl()
	case token.Fn:
		return p.parseFnDecl()
	case token.At:
		return p.parseDecoratedDecl()
	case token.Import:
		return p.parseImportDecl()
	case token.Export:
		return p.parseExportDecl()
	case token.Class:
		return p.parseClassDecl()
	case token.TypeKw:
		return p.parseTypeDecl()
	case token.Interface:
		return p.parseInterfaceDecl()
	default:
		return p.parseStatement()
	}
}

func (p *Parser) parseVarDecl() ast.Decl {
	startTok := p.advance()
	nameTok := p.expect(token.Ident, "expected variable name")
	p.expect(token.Assign, "expected '=' after variable name")
	value := p.parseExpression(PrecLowest)

	kind := ast.LetVar
	if startTok.Type == token.Const {
		kind = ast.ConstVar
	}

	return &ast.VarDecl{
		Kind:  kind,
		Name:  nameTok.Lexeme,
		Value: value,
		Span_: token.Span{Start: startTok.Span.Start, End: value.Span().End},
	}
}

func (p *Parser) parseFnDecl() ast.Decl {
	startTok := p.expect(token.Fn, "expected 'fn'")
	nameTok := p.expect(token.Ident, "expected function name")
	params := p.parseParamList()
	body := p.parseBlockStmt()
	if body == nil {
		return nil
	}
	return &ast.FnDecl{
		Name:   nameTok.Lexeme,
		Params: params,
		Body:   body,
		Span_:  token.Span{Start: startTok.Span.Start, End: body.Span().End},
	}
}

func (p *Parser) parseImportDecl() ast.Decl {
	startTok := p.expect(token.Import, "expected 'import'")
	path, end := p.parseImportPath()
	alias := moduleAliasFromPath(path)
	if p.match(token.As) {
		aliasTok := p.expect(token.Ident, "expected import alias")
		alias = aliasTok.Lexeme
		end = aliasTok.Span.End
	}
	return &ast.ImportDecl{
		Path:  path,
		Alias: alias,
		Span_: token.Span{Start: startTok.Span.Start, End: end},
	}
}

func (p *Parser) parseExportDecl() ast.Decl {
	startTok := p.expect(token.Export, "expected 'export'")
	var decl ast.Decl
	switch p.current().Type {
	case token.Const, token.Let:
		decl = p.parseVarDecl()
	case token.Fn:
		decl = p.parseFnDecl()
	case token.Class:
		decl = p.parseClassDecl()
	case token.At:
		decl = p.parseDecoratedDecl()
	default:
		p.errorAtCurrent("expected declaration after export")
		return nil
	}
	return &ast.ExportDecl{
		Decl:  decl,
		Span_: token.Span{Start: startTok.Span.Start, End: decl.Span().End},
	}
}

func (p *Parser) parseParamList() []ast.Param {
	p.expect(token.LParen, "expected '('")
	params := make([]ast.Param, 0, 4)
	if p.match(token.RParen) {
		return params
	}
	for {
		nameTok := p.expectIdentOrKeyword("parameter name")
		params = append(params, ast.Param{Name: nameTok.Lexeme, Span_: nameTok.Span})
		if p.match(token.RParen) {
			break
		}
		p.expect(token.Comma, "expected ',' or ')' in parameter list")
	}
	return params
}

func (p *Parser) parseImportPath() (string, token.Position) {
	if p.check(token.String) {
		pathTok := p.advance()
		return strings.Trim(pathTok.Lexeme, "\""), pathTok.Span.End
	}

	first := p.expect(token.Ident, "expected module path")
	parts := []string{first.Lexeme}
	end := first.Span.End
	for p.match(token.Dot) {
		part := p.expect(token.Ident, "expected module path segment after '.'")
		parts = append(parts, part.Lexeme)
		end = part.Span.End
	}
	return strings.Join(parts, "."), end
}

func moduleAliasFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "module"
	}
	return base
}

func (p *Parser) parseClassDecl() ast.Decl {
	startTok := p.expect(token.Class, "expected 'class'")
	nameTok := p.expect(token.Ident, "expected class name")
	var super ast.Expr
	if p.match(token.Lt) {
		super = p.parseExpression(PrecLowest)
	}
	p.expect(token.LBrace, "expected '{' after class name")
	methods := make([]ast.ClassMethod, 0, 4)
	for !p.check(token.RBrace) && !p.atEnd() {
		var decorators []ast.Expr
		var methodStart token.Position
		if p.check(token.At) {
			startTok, parsed := p.parseDecoratorList()
			decorators = parsed
			methodStart = startTok.Span.Start
		}
		methodNameTok := p.expectIdentOrKeyword("method name")
		methodParams := p.parseParamList()
		methodBody := p.parseBlockStmt()
		if methodBody == nil {
			return nil
		}
		if methodStart.Line == 0 {
			methodStart = methodNameTok.Span.Start
		}
		methods = append(methods, ast.ClassMethod{
			Name:       methodNameTok.Lexeme,
			Params:     methodParams,
			Body:       methodBody,
			Decorators: decorators,
			Span_:      token.Span{Start: methodStart, End: methodBody.Span().End},
		})
	}
	endTok := p.expect(token.RBrace, "expected '}' after class body")
	return &ast.ClassDecl{
		Name:    nameTok.Lexeme,
		Super:   super,
		Methods: methods,
		Span_:   token.Span{Start: startTok.Span.Start, End: endTok.Span.End},
	}
}

func (p *Parser) parseDecoratedDecl() ast.Decl {
	startTok, decorators := p.parseDecoratorList()

	var decl ast.Decl
	switch p.current().Type {
	case token.Fn:
		decl = p.parseFnDecl()
	case token.Class:
		decl = p.parseClassDecl()
	default:
		p.errorAtCurrent("expected function or class declaration after decorators")
		return nil
	}
	if decl == nil {
		return nil
	}

	return &ast.DecoratedDecl{
		Decl:       decl,
		Decorators: decorators,
		Span_:      token.Span{Start: startTok.Span.Start, End: decl.Span().End},
	}
}

func (p *Parser) parseDecoratorList() (token.Token, []ast.Expr) {
	startTok := p.expect(token.At, "expected '@'")
	decorators := make([]ast.Expr, 0, 2)
	decorators = append(decorators, p.parseExpression(PrecLowest))
	for p.match(token.At) {
		decorators = append(decorators, p.parseExpression(PrecLowest))
	}
	return startTok, decorators
}

func (p *Parser) parseTypeDecl() ast.Decl {
	startTok := p.expect(token.TypeKw, "expected 'type'")
	nameTok := p.expect(token.Ident, "expected type name")
	p.expect(token.Assign, "expected '=' in type declaration")
	typeExpr := p.parseTypeExpr()
	return &ast.TypeDecl{
		Name:    nameTok.Lexeme,
		TypeDef: typeExpr,
		Span_:   token.Span{Start: startTok.Span.Start, End: typeExpr.Span().End},
	}
}

func (p *Parser) parseInterfaceDecl() ast.Decl {
	startTok := p.expect(token.Interface, "expected 'interface'")
	nameTok := p.expect(token.Ident, "expected interface name")
	p.expect(token.LBrace, "expected '{' after interface name")
	methods := make([]ast.InterfaceMethod, 0, 4)
	for !p.check(token.RBrace) && !p.atEnd() {
		methodNameTok := p.expectIdentOrKeyword("method name")
		p.expect(token.LParen, "expected '('")
		paramTypes := p.parseParamTypeList()
		p.expect(token.RParen, "expected ')'")
		var returnType ast.TypeExpr
		if p.check(token.Ident) || p.check(token.Fn) || p.check(token.LBracket) {
			returnType = p.parseTypeExpr()
		}
		methods = append(methods, ast.InterfaceMethod{
			Name:       methodNameTok.Lexeme,
			ParamTypes: paramTypes,
			ReturnType: returnType,
			Span_:      token.Span{Start: methodNameTok.Span.Start},
		})
	}
	endTok := p.expect(token.RBrace, "expected '}' after interface body")
	return &ast.InterfaceDecl{
		Name:    nameTok.Lexeme,
		Methods: methods,
		Span_:   token.Span{Start: startTok.Span.Start, End: endTok.Span.End},
	}
}

func (p *Parser) parseParamTypeList() []ast.TypeExpr {
	types := make([]ast.TypeExpr, 0, 4)
	if p.check(token.RParen) {
		return types
	}
	for {
		p.expectIdentOrKeyword("parameter name")
		types = append(types, p.parseTypeExpr())
		if !p.match(token.Comma) {
			break
		}
	}
	return types
}

func (p *Parser) parseTypeExpr() ast.TypeExpr {
	switch p.current().Type {
	case token.Fn:
		return p.parseFuncTypeExpr()
	case token.LBracket:
		return p.parseArrayTypeExpr()
	default:
		nameTok := p.expectIdentOrKeyword("type name")
		return &ast.SimpleTypeExpr{Name: nameTok.Lexeme, Span_: nameTok.Span}
	}
}

func (p *Parser) parseFuncTypeExpr() ast.TypeExpr {
	startTok := p.expect(token.Fn, "expected 'fn'")
	p.expect(token.LParen, "expected '('")
	params := p.parseTypeExprList()
	p.expect(token.RParen, "expected ')'")
	var returnType ast.TypeExpr
	if p.check(token.Ident) || p.check(token.Fn) || p.check(token.LBracket) {
		returnType = p.parseTypeExpr()
	}
	return &ast.FuncTypeExpr{
		Params: params,
		Return: returnType,
		Span_:  token.Span{Start: startTok.Span.Start},
	}
}

func (p *Parser) parseArrayTypeExpr() ast.TypeExpr {
	startTok := p.expect(token.LBracket, "expected '['")
	elemType := p.parseTypeExpr()
	p.expect(token.RBracket, "expected ']'")
	return &ast.SimpleTypeExpr{Name: "[]" + elemType.(*ast.SimpleTypeExpr).Name, Span_: startTok.Span}
}

func (p *Parser) parseTypeExprList() []ast.TypeExpr {
	types := make([]ast.TypeExpr, 0, 4)
	if p.check(token.RParen) {
		return types
	}
	for {
		types = append(types, p.parseTypeExpr())
		if !p.match(token.Comma) {
			break
		}
	}
	return types
}
