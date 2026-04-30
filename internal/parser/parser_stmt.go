package parser

import (
	"icoo_lang/internal/ast"
	"icoo_lang/internal/token"
)

func (p *Parser) parseStatement() ast.Stmt {
	switch p.current().Type {
	case token.Const, token.Let:
		decl := p.parseVarDecl()
		if decl == nil {
			return nil
		}
		return &ast.DeclStmt{Decl: decl, Span_: decl.Span()}
	case token.LBrace:
		return p.parseBlockStmt()
	case token.If:
		return p.parseIfStmt()
	case token.While:
		return p.parseWhileStmt()
	case token.Return:
		return p.parseReturnStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	startTok := p.expect(token.LBrace, "expected '{'")
	stmts := make([]ast.Stmt, 0, 8)
	for !p.check(token.RBrace) && !p.atEnd() {
		stmt := p.parseStatement()
		if stmt == nil {
			p.synchronize()
			continue
		}
		stmts = append(stmts, stmt)
	}
	endTok := p.expect(token.RBrace, "expected '}'")
	return &ast.BlockStmt{
		Stmts: stmts,
		Span_: token.Span{Start: startTok.Span.Start, End: endTok.Span.End},
	}
}

func (p *Parser) parseIfStmt() ast.Stmt {
	startTok := p.expect(token.If, "expected 'if'")
	cond := p.parseExpression(PrecLowest)
	thenBlock := p.parseBlockStmt()
	if thenBlock == nil {
		return nil
	}
	var elseStmt ast.Stmt
	end := thenBlock.Span().End
	if p.match(token.Else) {
		if p.check(token.If) {
			elseStmt = p.parseIfStmt()
		} else {
			elseStmt = p.parseBlockStmt()
		}
		if elseStmt != nil {
			end = elseStmt.Span().End
		}
	}
	return &ast.IfStmt{
		Cond:  cond,
		Then:  thenBlock,
		Else:  elseStmt,
		Span_: token.Span{Start: startTok.Span.Start, End: end},
	}
}

func (p *Parser) parseWhileStmt() ast.Stmt {
	startTok := p.expect(token.While, "expected 'while'")
	cond := p.parseExpression(PrecLowest)
	body := p.parseBlockStmt()
	if body == nil {
		return nil
	}
	return &ast.WhileStmt{
		Cond:  cond,
		Body:  body,
		Span_: token.Span{Start: startTok.Span.Start, End: body.Span().End},
	}
}

func (p *Parser) parseReturnStmt() ast.Stmt {
	startTok := p.expect(token.Return, "expected 'return'")
	if p.check(token.RBrace) || p.check(token.EOF) {
		return &ast.ReturnStmt{Span_: startTok.Span}
	}
	value := p.parseExpression(PrecLowest)
	return &ast.ReturnStmt{
		Value: value,
		Span_: token.Span{Start: startTok.Span.Start, End: value.Span().End},
	}
}

func (p *Parser) parseExprStmt() ast.Stmt {
	expr := p.parseExpression(PrecLowest)
	return &ast.ExprStmt{
		Expr:  expr,
		Span_: expr.Span(),
	}
}
