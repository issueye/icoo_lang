package parser

import (
	"strings"
	"testing"

	"icoo_lang/internal/ast"
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

func TestFromImportParsesSpecs(t *testing.T) {
	input := `from "./math.ic" import add, version as mathVersion`
	tokens := lexer.LexAll(input)

	p := New(tokens)
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("expected no parser errors, got: %v", errs)
	}
	if len(prog.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(prog.Nodes))
	}

	decl, ok := prog.Nodes[0].(ast.Decl)
	if !ok {
		t.Fatalf("expected decl node, got %T", prog.Nodes[0])
	}
	importDecl, ok := decl.(*ast.ImportDecl)
	if !ok {
		t.Fatalf("expected ImportDecl, got %T", decl)
	}
	if importDecl.FromImport != true {
		t.Fatal("expected FromImport to be true")
	}
	if importDecl.Path != "./math.ic" {
		t.Fatalf("expected path ./math.ic, got %q", importDecl.Path)
	}
	if len(importDecl.Specs) != 2 {
		t.Fatalf("expected 2 import specs, got %d", len(importDecl.Specs))
	}
	if importDecl.Specs[0].Name != "add" || importDecl.Specs[0].Alias != "add" {
		t.Fatalf("unexpected first spec: %+v", importDecl.Specs[0])
	}
	if importDecl.Specs[1].Name != "version" || importDecl.Specs[1].Alias != "mathVersion" {
		t.Fatalf("unexpected second spec: %+v", importDecl.Specs[1])
	}
}

func TestNamedExportParsesSpecs(t *testing.T) {
	input := `export { version, add as plus }`
	tokens := lexer.LexAll(input)

	p := New(tokens)
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("expected no parser errors, got: %v", errs)
	}
	if len(prog.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(prog.Nodes))
	}

	decl, ok := prog.Nodes[0].(ast.Decl)
	if !ok {
		t.Fatalf("expected decl node, got %T", prog.Nodes[0])
	}
	exportDecl, ok := decl.(*ast.ExportDecl)
	if !ok {
		t.Fatalf("expected ExportDecl, got %T", decl)
	}
	if exportDecl.NamedExport != true {
		t.Fatal("expected NamedExport to be true")
	}
	if len(exportDecl.Specs) != 2 {
		t.Fatalf("expected 2 export specs, got %d", len(exportDecl.Specs))
	}
	if exportDecl.Specs[0].Name != "version" || exportDecl.Specs[0].Alias != "version" {
		t.Fatalf("unexpected first spec: %+v", exportDecl.Specs[0])
	}
	if exportDecl.Specs[1].Name != "add" || exportDecl.Specs[1].Alias != "plus" {
		t.Fatalf("unexpected second spec: %+v", exportDecl.Specs[1])
	}
}

func TestNamedExportParsesExpressionBinding(t *testing.T) {
	input := `export { request: clientApi.request }`
	tokens := lexer.LexAll(input)

	p := New(tokens)
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("expected no parser errors, got: %v", errs)
	}
	decl := prog.Nodes[0].(ast.Decl)
	exportDecl := decl.(*ast.ExportDecl)
	if len(exportDecl.Specs) != 1 {
		t.Fatalf("expected 1 export spec, got %d", len(exportDecl.Specs))
	}
	if exportDecl.Specs[0].Name != "request" {
		t.Fatalf("expected export name request, got %q", exportDecl.Specs[0].Name)
	}
	if exportDecl.Specs[0].Value == nil {
		t.Fatal("expected export expression value")
	}
}
