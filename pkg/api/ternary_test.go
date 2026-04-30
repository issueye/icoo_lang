package api

import "testing"

func TestTernaryExprBasic(t *testing.T) {
	src := `
let a = true ? 1 : 2
let b = false ? 1 : 2
if a != 1 { panic("expected true branch") }
if b != 2 { panic("expected false branch") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("ternary basic failed: %v", err)
	}
}

func TestTernaryExprShortCircuits(t *testing.T) {
	src := `
fn boom() { panic("unselected branch ran") }
let a = true ? 1 : boom()
let b = false ? boom() : 2
if a != 1 { panic("expected true branch") }
if b != 2 { panic("expected false branch") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("ternary short-circuit failed: %v", err)
	}
}

func TestTernaryExprPrecedenceAndRightAssociativity(t *testing.T) {
	src := `
let a = false || true ? 1 + 2 * 3 : 0
let b = false ? 1 : true ? 2 : 3
if a != 7 { panic("expected ternary to bind below logical and arithmetic") }
if b != 2 { panic("expected ternary to be right associative") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("ternary precedence failed: %v", err)
	}
}

func TestTernaryExprWithTryPostfix(t *testing.T) {
	src := `
fn getArray() { return [10, 20] }
fn fail() { return error("boom") }
let index = true ? 0 : 1
let value = getArray()?[index]
let result = false ? fail()? : value
if value != 10 { panic("expected try postfix before index") }
if result != 10 { panic("expected ternary branch with try postfix") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("ternary with try postfix failed: %v", err)
	}
}
