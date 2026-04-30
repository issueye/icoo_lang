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

func TestRuntimeRunSource_TryCatchCatchesThrowString(t *testing.T) {
	src := `
let message = ""

try {
  throw "boom"
} catch err {
  message = err.message
}

if message != "boom" {
  panic("unexpected caught throw message")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected throw string run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryCatchNormalizesThrowValue(t *testing.T) {
	src := `
let kind = ""
let message = ""

try {
  throw 123
} catch err {
  kind = typeOf(err)
  message = err.message
}

if kind != "error" {
  panic("unexpected normalized throw kind")
}
if message != "123" {
  panic("unexpected normalized throw message")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected throw normalization run to succeed, got error: %v", err)
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

func TestRuntimeRunSource_UncaughtThrowStillReturnsHostError(t *testing.T) {
	src := `throw "boom"`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatalf("expected uncaught throw to be returned to host")
	} else {
		if err.Error() != "boom\n  at __module_init__ (unknown)" {
			t.Fatalf("unexpected uncaught throw stack: %q", err.Error())
		}
	}
}

func TestRuntimeRunSource_CatchCanReadErrorStack(t *testing.T) {
	src := `
let message = ""
let stack = ""

fn boom() {
  throw "boom"
}

try {
  boom()
} catch err {
  message = err.message
  stack = err.stack
}

if message != "boom" {
  panic("unexpected caught throw message")
}
if stack == "" {
  panic("expected non-empty error stack")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected catch stack run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_RethrowPreservesOriginalStack(t *testing.T) {
	src := `
let stack = ""

fn inner() {
  throw "boom"
}

fn outer() {
  try {
    inner()
  } catch err {
    throw err
  }
}

try {
  outer()
} catch err {
  stack = err.stack
}

if stack == "" {
  panic("expected rethrow stack")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected rethrow stack run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryFinallyRunsOnNormalPath(t *testing.T) {
	src := `
let out = ""

try {
  out = out + "try"
} finally {
  out = out + "/finally"
}

if out != "try/finally" {
  panic("unexpected try/finally normal result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected try/finally normal run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryCatchFinallyRunsCatchThenFinally(t *testing.T) {
	src := `
let out = ""

try {
  throw "boom"
} catch err {
  out = out + err.message
} finally {
  out = out + "/finally"
}

if out != "boom/finally" {
  panic("unexpected try/catch/finally result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected try/catch/finally run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_TryFinallyRethrowsAfterFinally(t *testing.T) {
	src := `
let out = ""

try {
  try {
    throw "boom"
  } finally {
    out = "finally"
  }
} catch err {
  out = out + "/" + err.message
}

if out != "finally/boom" {
  panic("unexpected try/finally rethrow result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected try/finally rethrow run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ReturnRunsFinally(t *testing.T) {
	src := `
fn demo() {
  let out = ""
  try {
    out = "try"
    return out
  } finally {
    out = out + "/finally"
    if out != "try/finally" {
      panic("finally should observe pre-return state")
    }
  }
}

if demo() != "try" {
  panic("unexpected return result after finally")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected return through finally run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_FinallyThrowOverridesReturn(t *testing.T) {
	src := `
fn demo() {
  try {
    return "ok"
  } finally {
    throw "finally boom"
  }
}

try {
  demo()
  panic("expected finally throw to override return")
} catch err {
  if err.message != "finally boom" {
    panic("unexpected finally override message")
  }
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected finally override run to succeed, got error: %v", err)
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
