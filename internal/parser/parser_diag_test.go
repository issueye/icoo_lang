package parser

import (
	"strings"
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

func TestClassInheritanceOldSyntaxDiagnostic(t *testing.T) {
	input := `class Dog < Animal {}`
	tokens := lexer.LexAll(input)

	p := New(tokens)
	_ = p.ParseProgram()
	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatal("expected parser error for old inheritance syntax")
	}

	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "class inheritance uses '<-' instead of '<'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected targeted inheritance diagnostic, got: %v", errs)
	}
}

func TestClassInheritanceArrowAndLessComparisonCoexist(t *testing.T) {
	input := `
class Base {
  less(a, b) {
    return a < b
  }
}

class Child <- Base {
  check() {
    return this.less(1, 2)
  }
}
`
	tokens := lexer.LexAll(input)

	p := New(tokens)
	_ = p.ParseProgram()
	errs := p.Errors()
	if len(errs) > 0 {
		t.Fatalf("expected no parser errors, got: %v", errs)
	}
}
