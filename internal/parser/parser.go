package parser

import (
	"fmt"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/token"
)

type Parser struct {
	tokens []token.Token
	pos    int
	errors []error
}

func New(tokens []token.Token) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) ParseProgram() *ast.Program {
	decls := make([]ast.Decl, 0, 16)
	var start token.Span
	var end token.Span
	started := false

	for !p.atEnd() {
		decl := p.parseTopLevelDecl()
		if decl == nil {
			p.synchronize()
			if p.atEnd() {
				break
			}
			continue
		}
		if !started {
			start = decl.Span()
			started = true
		}
		end = decl.Span()
		decls = append(decls, decl)
	}

	prog := &ast.Program{Decls: decls}
	if started {
		prog.Span_ = ast.MergeSpan(start, end)
	}
	return prog
}

func (p *Parser) Errors() []error {
	return p.errors
}

func (p *Parser) current() token.Token {
	if len(p.tokens) == 0 {
		return token.Token{Type: token.EOF}
	}
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *Parser) previous() token.Token {
	if p.pos == 0 || len(p.tokens) == 0 {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) peek(offset int) token.Token {
	idx := p.pos + offset
	if len(p.tokens) == 0 {
		return token.Token{Type: token.EOF}
	}
	if idx >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[idx]
}

func (p *Parser) advance() token.Token {
	cur := p.current()
	if !p.atEnd() {
		p.pos++
	}
	return cur
}

func (p *Parser) match(types ...token.Type) bool {
	for _, tt := range types {
		if p.check(tt) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) check(tt token.Type) bool {
	return p.current().Type == tt
}

func (p *Parser) expect(tt token.Type, msg string) token.Token {
	if p.check(tt) {
		return p.advance()
	}
	p.errorAtCurrent(msg)
	return token.Token{Type: token.Illegal, Span: p.current().Span}
}

func (p *Parser) atEnd() bool {
	return p.current().Type == token.EOF
}

func (p *Parser) synchronize() {
	for !p.atEnd() {
		if p.previous().Type == token.Semicolon {
			return
		}
		switch p.current().Type {
		case token.Const, token.Let, token.Fn, token.If, token.While, token.Return:
			return
		case token.RBrace:
			return
		}
		p.advance()
	}
}

func (p *Parser) errorAtCurrent(msg string) {
	tok := p.current()
	p.errors = append(p.errors, fmt.Errorf("%d:%d: %s (found %s)", tok.Span.Start.Line, tok.Span.Start.Column, msg, tok.Type.String()))
}
