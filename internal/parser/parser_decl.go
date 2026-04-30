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
	case token.Import:
		return p.parseImportDecl()
	case token.Export:
		return p.parseExportDecl()
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
	pathTok := p.expect(token.String, "expected module path string")
	path := strings.Trim(pathTok.Lexeme, "\"")
	alias := moduleAliasFromPath(path)
	end := pathTok.Span.End
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
		nameTok := p.expect(token.Ident, "expected parameter name")
		params = append(params, ast.Param{Name: nameTok.Lexeme, Span_: nameTok.Span})
		if p.match(token.RParen) {
			break
		}
		p.expect(token.Comma, "expected ',' or ')' in parameter list")
	}
	return params
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
