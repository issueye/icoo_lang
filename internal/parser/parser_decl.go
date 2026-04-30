package parser

import (
	"icoo_lang/internal/ast"
	"icoo_lang/internal/token"
)

func (p *Parser) parseTopLevelDecl() ast.Decl {
	switch p.current().Type {
	case token.Const, token.Let:
		return p.parseVarDecl()
	case token.Fn:
		return p.parseFnDecl()
	default:
		p.errorAtCurrent("expected declaration")
		return nil
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
