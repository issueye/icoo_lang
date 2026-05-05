package parser

import (
	"testing"

	"icoo_lang/internal/lexer"
)

func TestClassParse(t *testing.T) {
	input := `class C { fn m() { let x = this } }`
	tokens := lexer.LexAll(input)
	t.Logf("Tokens: %d", len(tokens))
	for i, tok := range tokens {
		if i > 30 {
			t.Log("...")
			break
		}
		t.Logf("  %d: %s %q", i, tok.Type, tok.Lexeme)
	}

	p := New(tokens)
	prog := p.ParseProgram()
	t.Logf("Nodes: %d", len(prog.Nodes))
	for _, err := range p.Errors() {
		t.Logf("Error: %v", err)
	}
	t.Log("Done")
}

func TestObjectLiteralParse(t *testing.T) {
	input := `{a: 1, b: 2}`
	tokens := lexer.LexAll(input)
	t.Logf("Tokens: %d", len(tokens))
	for i, tok := range tokens {
		t.Logf("  %d: %s %q", i, tok.Type, tok.Lexeme)
	}

	p := New(tokens)
	prog := p.ParseProgram()
	t.Logf("Nodes: %d", len(prog.Nodes))
	for _, err := range p.Errors() {
		t.Logf("Error: %v", err)
	}
	t.Log("Done")
}
