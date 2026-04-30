package parser

import (
	"icoo_lang/internal/ast"
	"icoo_lang/internal/token"
)

func (p *Parser) parseExpression(precedence Precedence) ast.Expr {
	left := p.parsePrefix()
	if left == nil {
		return &ast.NullLiteral{Span_: p.current().Span}
	}

	for !p.atEnd() && !p.check(token.RParen) && !p.check(token.RBracket) && !p.check(token.RBrace) {
		nextPrec := p.currentPrecedence()
		if nextPrec <= precedence {
			break
		}
		left = p.parseInfix(left, nextPrec)
		if left == nil {
			return &ast.NullLiteral{Span_: p.current().Span}
		}
	}

	return left
}

func (p *Parser) currentPrecedence() Precedence {
	if p.check(token.Question) {
		if p.isTernaryQuestion() {
			return PrecTernary
		}
		return PrecPostfix
	}
	return precedenceOf(p.current().Type)
}

func (p *Parser) isTernaryQuestion() bool {
	if !p.startsExpression(p.peek(1).Type) {
		return false
	}

	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for offset := 1; ; offset++ {
		tok := p.peek(offset)
		if offset > 1 && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && p.isTernaryScanBoundary(tok.Type) {
			return false
		}
		switch tok.Type {
		case token.EOF:
			return false
		case token.LParen:
			parenDepth++
		case token.RParen:
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return false
			}
			parenDepth--
		case token.LBracket:
			bracketDepth++
		case token.RBracket:
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return false
			}
			bracketDepth--
		case token.LBrace:
			braceDepth++
		case token.RBrace:
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return false
			}
			braceDepth--
		case token.Colon:
			return parenDepth == 0 && bracketDepth == 0 && braceDepth == 0
		case token.Comma, token.Semicolon:
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return false
			}
		}
	}
}

func (p *Parser) isTernaryScanBoundary(tt token.Type) bool {
	switch tt {
	case token.Const, token.Let, token.Fn,
		token.Return, token.If, token.Else,
		token.For, token.While, token.Match,
		token.Break, token.Continue,
		token.Import, token.Export,
		token.Try, token.Catch, token.Finally, token.Throw,
		token.Go, token.Select,
		token.Interface, token.TypeKw, token.Class:
		return true
	default:
		return false
	}
}

func (p *Parser) startsExpression(tt token.Type) bool {
	switch tt {
	case token.Ident, token.Underscore,
		token.Int, token.Float, token.String,
		token.True, token.False, token.Null,
		token.Bang, token.Minus,
		token.LParen, token.LBracket, token.LBrace,
		token.Fn, token.This:
		return true
	default:
		return false
	}
}

func (p *Parser) parsePrefix() ast.Expr {
	tok := p.current()
	switch tok.Type {
	case token.Ident, token.Underscore:
		p.advance()
		return &ast.IdentExpr{Name: tok.Lexeme, Span_: tok.Span}
	case token.Int:
		p.advance()
		return &ast.IntLiteral{Raw: tok.Lexeme, Span_: tok.Span}
	case token.Float:
		p.advance()
		return &ast.FloatLiteral{Raw: tok.Lexeme, Span_: tok.Span}
	case token.String:
		p.advance()
		return &ast.StringLiteral{Raw: tok.Lexeme, Span_: tok.Span}
	case token.True:
		p.advance()
		return &ast.BoolLiteral{Value: true, Span_: tok.Span}
	case token.False:
		p.advance()
		return &ast.BoolLiteral{Value: false, Span_: tok.Span}
	case token.Null:
		p.advance()
		return &ast.NullLiteral{Span_: tok.Span}
	case token.Bang, token.Minus:
		p.advance()
		right := p.parseExpression(PrecUnary)
		return &ast.UnaryExpr{Op: tok.Type, Right: right, Span_: token.Span{Start: tok.Span.Start, End: right.Span().End}}
	case token.LParen:
		p.advance()
		expr := p.parseExpression(PrecLowest)
		p.expect(token.RParen, "expected ')' after expression")
		return expr
	case token.LBracket:
		return p.parseArrayLiteral()
	case token.LBrace:
		return p.parseObjectLiteral()
	case token.Fn:
		return p.parseFnExpr()
	case token.This:
		tok := p.advance()
		return &ast.ThisExpr{Span_: tok.Span}
	default:
		p.errorAtCurrent("expected expression")
		return nil
	}
}

func (p *Parser) parseInfix(left ast.Expr, precedence Precedence) ast.Expr {
	tok := p.advance()
	switch tok.Type {
	case token.Assign, token.PlusAssign, token.MinusAssign, token.StarAssign, token.SlashAssign:
		right := p.parseExpression(PrecAssign - 1)
		return &ast.AssignExpr{Target: left, Op: tok.Type, Value: right, Span_: token.Span{Start: left.Span().Start, End: right.Span().End}}
	case token.Plus, token.Minus, token.Star, token.Slash, token.Percent,
		token.Eq, token.Neq, token.Lt, token.Lte, token.Gt, token.Gte,
		token.AndAnd, token.OrOr:
		right := p.parseExpression(precedence)
		return &ast.BinaryExpr{Left: left, Op: tok.Type, Right: right, Span_: token.Span{Start: left.Span().Start, End: right.Span().End}}
	case token.LParen:
		return p.finishCall(left, tok)
	case token.Dot:
		nameTok := p.expectIdentOrKeyword("property name")
		return &ast.MemberExpr{Object: left, Name: nameTok.Lexeme, Span_: token.Span{Start: left.Span().Start, End: nameTok.Span.End}}
	case token.LBracket:
		index := p.parseExpression(PrecLowest)
		endTok := p.expect(token.RBracket, "expected ']' after index")
		return &ast.IndexExpr{Object: left, Index: index, Span_: token.Span{Start: left.Span().Start, End: endTok.Span.End}}
	case token.Question:
		if precedence == PrecTernary {
			thenExpr := p.parseExpression(PrecLowest)
			p.expect(token.Colon, "expected ':' in ternary expression")
			elseExpr := p.parseExpression(PrecTernary - 1)
			return &ast.TernaryExpr{Cond: left, Then: thenExpr, Else: elseExpr, Span_: token.Span{Start: left.Span().Start, End: elseExpr.Span().End}}
		}
		return &ast.TryExpr{Expr: left, Span_: token.Span{Start: left.Span().Start, End: tok.Span.End}}
	default:
		p.errorAtCurrent("unexpected infix operator")
		return left
	}
}

func (p *Parser) finishCall(callee ast.Expr, start token.Token) ast.Expr {
	args := make([]ast.Expr, 0, 4)
	if !p.check(token.RParen) {
		for {
			args = append(args, p.parseExpression(PrecLowest))
			if !p.match(token.Comma) {
				break
			}
		}
	}
	endTok := p.expect(token.RParen, "expected ')' after arguments")
	return &ast.CallExpr{Callee: callee, Args: args, Span_: token.Span{Start: callee.Span().Start, End: endTok.Span.End}}
}

func (p *Parser) parseArrayLiteral() ast.Expr {
	startTok := p.expect(token.LBracket, "expected '['")
	items := make([]ast.Expr, 0, 4)
	if !p.check(token.RBracket) {
		for {
			items = append(items, p.parseExpression(PrecLowest))
			if !p.match(token.Comma) {
				break
			}
		}
	}
	endTok := p.expect(token.RBracket, "expected ']' after array literal")
	return &ast.ArrayLiteral{Items: items, Span_: token.Span{Start: startTok.Span.Start, End: endTok.Span.End}}
}

func (p *Parser) parseObjectLiteral() ast.Expr {
	startTok := p.expect(token.LBrace, "expected '{'")
	fields := make([]ast.ObjectField, 0, 4)
	if !p.check(token.RBrace) {
		for {
			nameTok := p.expectIdentOrKeyword("object field name")
			p.expect(token.Colon, "expected ':' after object field name")
			value := p.parseExpression(PrecLowest)
			fields = append(fields, ast.ObjectField{Name: nameTok.Lexeme, Value: value, Span_: token.Span{Start: nameTok.Span.Start, End: value.Span().End}})
			if !p.match(token.Comma) {
				break
			}
		}
	}
	endTok := p.expect(token.RBrace, "expected '}' after object literal")
	return &ast.ObjectLiteral{Fields: fields, Span_: token.Span{Start: startTok.Span.Start, End: endTok.Span.End}}
}

func (p *Parser) parseFnExpr() ast.Expr {
	startTok := p.expect(token.Fn, "expected 'fn'")
	params := p.parseParamList()
	body := p.parseBlockStmt()
	return &ast.FnExpr{Params: params, Body: body, Span_: token.Span{Start: startTok.Span.Start, End: body.Span().End}}
}
