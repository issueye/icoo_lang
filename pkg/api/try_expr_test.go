package api

import (
	"testing"
)

func TestTryExprBasic(t *testing.T) {
	src := `
fn safe() { return 42 }
let result = safe()?
if result != 42 { panic("expected 42") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr basic failed: %v", err)
	}
}

func TestTryExprNonError(t *testing.T) {
	src := `
fn get() { return "hello" }
let result = get()?
if result != "hello" { panic("expected hello") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr non-error failed: %v", err)
	}
}

func TestTryExprIntValue(t *testing.T) {
	src := `
fn get() { return 100 }
let result = get()?
if result != 100 { panic("expected 100") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr int failed: %v", err)
	}
}

func TestTryExprObjectValue(t *testing.T) {
	src := `
fn get() { return {x: 1, y: 2} }
let result = get()?
if result.x != 1 { panic("expected x=1") }
if result.y != 2 { panic("expected y=2") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr object failed: %v", err)
	}
}

func TestTryExprDoesNotConsumeFollowingStatement(t *testing.T) {
	src := `
fn safe() { return 42 }
let result = safe()?
let after = 7
if result != 42 { panic("expected 42") }
if after != 7 { panic("expected following statement to run") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr following statement failed: %v", err)
	}
}

func TestTryExprPropagatesErrorValue(t *testing.T) {
	src := `
fn fail() { return error("boom") }
fn outer() {
  let value = fail()?
  return value
}
let result = outer()
if typeOf(result) != "error" { panic("expected error result") }
if result.message != "boom" { panic("expected propagated message") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr error propagation failed: %v", err)
	}
}

func TestTryExprRunsFinallyBeforePropagating(t *testing.T) {
	src := `
let trace = ""
fn fail() { return error("boom") }
fn outer() {
  try {
    trace = trace + "try"
    let value = fail()?
    return value
  } finally {
    trace = trace + "/finally"
  }
}
let result = outer()
if typeOf(result) != "error" { panic("expected error result") }
if result.message != "boom" { panic("expected propagated message") }
if trace != "try/finally" { panic("expected finally before propagation") }
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr finally propagation failed: %v", err)
	}
}
