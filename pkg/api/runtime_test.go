package api

import (
	"path/filepath"
	"testing"
)

func TestRuntimeCheckSource_AllowsTopLevelExprStmt(t *testing.T) {
	src := `
fn main() {
  println("ok")
}

main()
`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) > 0 {
		t.Fatalf("expected no check errors, got %v", errs)
	}
}

func TestRuntimeRunSource_FunctionCallUsesCorrectArguments(t *testing.T) {
	src := `
fn add(a, b) {
  return a + b
}

let result = add(1, 2)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryCatchCatchesPanic(t *testing.T) {
	src := `
let message = ""

try {
  panic("boom")
} catch err {
  message = err.message
}

if message != "panic: boom" {
  panic("unexpected caught panic message")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected try/catch panic run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryCatchCatchesRuntimeError(t *testing.T) {
	src := `
let kind = ""
let message = ""

try {
  len(1)
} catch err {
  kind = typeOf(err)
  message = err.message
}

if kind != "error" {
  panic("unexpected caught error kind")
}
if message == "" {
  panic("expected caught runtime error message")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected try/catch runtime error run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryCatchPrefersInnerHandler(t *testing.T) {
	src := `
let out = ""

try {
  try {
    panic("inner")
  } catch err {
    out = out + err.message
  }
} catch outer {
  out = "outer"
}

if out != "panic: inner" {
  panic("unexpected nested try/catch result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected nested try/catch run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_UncaughtTryErrorStillReturnsHostError(t *testing.T) {
	src := `
try {
  let x = 1
} catch err {
  let y = err
}

len(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatalf("expected uncaught runtime error to be returned to host")
	}
}

func TestRuntimeRunFile_StdlibIntegrationScript(t *testing.T) {
	rt := NewRuntime()
	path := filepath.Join("..", "..", "testdata", "integration", "stdlib.ic")
	if _, err := rt.RunFile(path); err != nil {
		t.Fatalf("expected stdlib integration script to succeed, got error: %v", err)
	}
}

func TestRuntimeRunFile_TryCatchIntegrationScript(t *testing.T) {
	rt := NewRuntime()
	path := filepath.Join("..", "..", "testdata", "integration", "trycatch.ic")
	if _, err := rt.RunFile(path); err != nil {
		t.Fatalf("expected try/catch integration script to succeed, got error: %v", err)
	}
}
