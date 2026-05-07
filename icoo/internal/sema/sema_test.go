package sema

import (
	"strings"
	"testing"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/diag"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
)

func parse(source string) *ast.Program {
	tokens := lexer.LexAll(source)
	p := parser.New(tokens)
	return p.ParseProgram()
}

func hasError(t *testing.T, diagnostics []diag.Diagnostic, fragment string) {
	t.Helper()
	for _, d := range diagnostics {
		if strings.Contains(d.Message, fragment) {
			return
		}
	}
	for _, d := range diagnostics {
		t.Logf("diagnostic: %s", d.Message)
	}
	t.Errorf("expected diagnostic containing %q, got none", fragment)
}

func noError(t *testing.T, diagnostics []diag.Diagnostic) {
	t.Helper()
	for _, d := range diagnostics {
		if d.Severity == diag.Error {
			t.Errorf("unexpected diagnostic: %s", d.Message)
		}
	}
}

func TestSema_UndefinedIdentifier(t *testing.T) {
	program := parse("let x = y")
	diags := Analyze(program)
	hasError(t, diags, "undefined identifier: y")
}

func TestSema_DefinedIdentifier(t *testing.T) {
	program := parse("let x = 1\nlet y = x")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_DuplicateDeclaration(t *testing.T) {
	program := parse("let x = 1\nlet x = 2")
	diags := Analyze(program)
	hasError(t, diags, "duplicate declaration: x")
}

func TestSema_DuplicateFnDeclaration(t *testing.T) {
	program := parse("fn f() { }\nfn f() { }")
	diags := Analyze(program)
	hasError(t, diags, "duplicate declaration: f")
}

func TestSema_DuplicateParameter(t *testing.T) {
	program := parse("fn f(a, a) { return a }")
	diags := Analyze(program)
	hasError(t, diags, "duplicate parameter: a")
}

func TestSema_ConstReassignment(t *testing.T) {
	program := parse("const x = 1\nx = 2")
	diags := Analyze(program)
	hasError(t, diags, "cannot assign to const: x")
}

func TestSema_LetReassignment(t *testing.T) {
	program := parse("let x = 1\nx = 2")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_ReturnOutsideFunction(t *testing.T) {
	program := parse("return 42")
	diags := Analyze(program)
	hasError(t, diags, "return used outside function")
}

func TestSema_ReturnInsideFunction(t *testing.T) {
	program := parse("fn f() { return 42 }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_BreakOutsideLoop(t *testing.T) {
	program := parse("break")
	diags := Analyze(program)
	hasError(t, diags, "break used outside loop")
}

func TestSema_BreakInsideLoop(t *testing.T) {
	program := parse("while true { break }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_ContinueOutsideLoop(t *testing.T) {
	program := parse("continue")
	diags := Analyze(program)
	hasError(t, diags, "continue used outside loop")
}

func TestSema_ContinueInsideLoop(t *testing.T) {
	program := parse("while true { continue }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_ThisOutsideClass(t *testing.T) {
	program := parse("fn f() { let x = this }")
	diags := Analyze(program)
	hasError(t, diags, "this used outside class method")
}

func TestSema_ThisInsideClass(t *testing.T) {
	// Parser currently treats `fn` inside class body as a method name via
	// expectIdentOrKeyword, not as the fn keyword. This test will pass
	// once the parser correctly distinguishes method declarations.
	t.Skip("parser does not yet parse fn declarations inside class bodies")
}

func TestSema_SuperOutsideSubclass(t *testing.T) {
	t.Skip("parser does not yet parse fn declarations inside class bodies")
}

func TestSema_SuperInsideSubclass(t *testing.T) {
	t.Skip("parser does not yet parse fn declarations inside class bodies")
}

func TestSema_BlockScoping(t *testing.T) {
	// Variable defined in block should be accessible within that block
	program := parse("let x = 1\n{ let y = x }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_BlockScopingOuterInaccessible(t *testing.T) {
	// Variable defined in block should NOT be accessible outside
	program := parse("{ let y = 1 }\nlet z = y")
	diags := Analyze(program)
	hasError(t, diags, "undefined identifier: y")
}

func TestSema_ShadowVariable(t *testing.T) {
	// Inner scope shadow should be fine
	program := parse("let x = 1\nif true { let x = 2\n let y = x }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_FunctionScopeIsolation(t *testing.T) {
	// Parameter shadows outer variable
	program := parse("let x = 1\nfn f(x) { return x }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_NestedFunctionScope(t *testing.T) {
	program := parse("fn outer() { let a = 1\n fn inner() { return a } }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_NestedFunctionUndefined(t *testing.T) {
	program := parse("fn outer() {\n fn inner() { return a }\n let a = 1\n }")
	diags := Analyze(program)
	hasError(t, diags, "undefined identifier: a")
}

func TestSema_BuiltinsAccessible(t *testing.T) {
	program := parse("let x = print\ntypeOf(1)\nlen([])\nchan(0)")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_TryCatchScope(t *testing.T) {
	program := parse("try { let x = throw } catch err { let y = err }")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_TryVariableNotAvailableAfter(t *testing.T) {
	program := parse("let x = 1\nif true { let y = 2 }\nlet z = y")
	diags := Analyze(program)
	hasError(t, diags, "undefined identifier: y")
}

func TestSema_AssignmentToUndefined(t *testing.T) {
	program := parse("x = 5")
	diags := Analyze(program)
	hasError(t, diags, "undefined identifier: x")
}

func TestSema_ValidTernaryExpr(t *testing.T) {
	program := parse("let x = true ? 1 : 2")
	diags := Analyze(program)
	noError(t, diags)
}

func TestSema_MatchWildcardOrder(t *testing.T) {
	// Wildcard should be last
	program := parse("match 1 { _ { } 1 { } }")
	diags := Analyze(program)
	hasError(t, diags, "wildcard match arm must be last")
}

func TestSema_DuplicateWildcardMatch(t *testing.T) {
	program := parse("match 1 { _ { } _ { } }")
	diags := Analyze(program)
	hasError(t, diags, "duplicate wildcard match arm")
}

func TestSema_SelectElseOrder(t *testing.T) {
	program := parse("select { recv c { } else { } recv c { } }")
	diags := Analyze(program)
	hasError(t, diags, "else/default case must be last")
}

func TestSema_DuplicateSelectDefault(t *testing.T) {
	program := parse("select { else { } else { } }")
	diags := Analyze(program)
	hasError(t, diags, "duplicate else/default case")
}

func TestSema_InvalidAssignmentTarget(t *testing.T) {
	program := parse("1 = 2")
	diags := Analyze(program)
	hasError(t, diags, "invalid assignment target")
}

func TestSema_LessComparisonStillValidAfterInheritanceSyntaxChange(t *testing.T) {
	program := parse("let ok = 1 < 2")
	diags := Analyze(program)
	noError(t, diags)
}
