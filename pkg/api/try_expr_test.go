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
