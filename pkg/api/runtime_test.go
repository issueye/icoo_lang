package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestRuntimeCheckSource_RejectsInvalidObjectFieldWithoutHanging(t *testing.T) {
	src := `
let value = {
  1: 2
}
`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected invalid object field to report an error")
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

func TestRuntimeInvokeGlobal_CallsProjectEntryFunction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.ic")
	marker := filepath.Join(dir, "marker.txt")
	if err := os.WriteFile(path, []byte(`import std.io.fs as fs

fn main() {
  fs.writeFile("`+filepath.ToSlash(marker)+`", "ok")
}
`), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(path); err != nil {
		t.Fatalf("expected run file to succeed, got: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expected entry function not to run during RunFile, stat err=%v", err)
	}
	if _, err := rt.InvokeGlobal("main"); err != nil {
		t.Fatalf("expected InvokeGlobal to succeed, got: %v", err)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected marker file after InvokeGlobal, got: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("expected marker contents ok, got %q", string(data))
	}
}

func TestRuntimeInvokeGlobal_RejectsMissingFunction(t *testing.T) {
	rt := NewRuntime()
	if _, err := rt.RunSource(`let value = 1`); err != nil {
		t.Fatalf("expected source run to succeed, got: %v", err)
	}
	if _, err := rt.InvokeGlobal("main"); err == nil {
		t.Fatal("expected missing global to return error")
	}
}

func TestRuntimeInvokeGlobal_RejectsNonCallableGlobal(t *testing.T) {
	rt := NewRuntime()
	if _, err := rt.RunSource(`let main = 1`); err != nil {
		t.Fatalf("expected source run to succeed, got: %v", err)
	}
	if _, err := rt.InvokeGlobal("main"); err == nil {
		t.Fatal("expected non-callable global to return error")
	} else if !strings.Contains(err.Error(), "not callable") {
		t.Fatalf("expected not callable error, got: %v", err)
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
		if err.Error() != "boom\n  at __module_init__ (unknown:1)" {
			t.Fatalf("unexpected uncaught throw stack: %q", err.Error())
		}
	}
}

func TestRuntimeRunSource_CatchCanReadErrorStack(t *testing.T) {
	src := `
let message = ""
let stack = ""
let frameCount = 0
let topFunction = ""
let causeMessage = ""
let causeType = ""

fn boom() {
  throw error("boom", error("inner"))
}

try {
  boom()
} catch err {
  message = err.message
  stack = err.stack
  frameCount = len(err.frames)
  topFunction = err.frames[0].function
  causeMessage = err.cause.message
  causeType = typeOf(err.cause)
}

if message != "boom" {
  panic("unexpected caught throw message")
}
if stack == "" {
  panic("expected non-empty error stack")
}
if frameCount == 0 {
  panic("expected structured frames")
}
if topFunction != "boom" {
  panic("unexpected top frame function")
}
if causeMessage != "inner" {
  panic("unexpected cause message")
}
if causeType != "error" {
  panic("unexpected cause type")
}
if stack == message {
  panic("expected chained stack output")
}
if stack == causeMessage {
  panic("expected stack to include throwing frame")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected catch stack run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ErrorWithoutCauseReturnsNullCause(t *testing.T) {
	src := `
let isNull = false

try {
  throw error("solo")
} catch err {
  isNull = err.cause == null
}

if isNull != true {
  panic("expected null cause")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected error without cause run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ErrorNormalizesNonErrorCause(t *testing.T) {
	src := `
let causeMessage = ""

try {
  throw error("outer", 123)
} catch err {
  causeMessage = err.cause.message
}

if causeMessage != "123" {
  panic("unexpected normalized cause message")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected normalized cause run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_RethrowPreservesOriginalStack(t *testing.T) {
	src := `
let stack = ""
let causeMessage = ""

fn inner() {
  throw error("boom", error("inner cause"))
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
  causeMessage = err.cause.message
}

if stack == "" {
  panic("expected rethrow stack")
}
if causeMessage != "inner cause" {
  panic("expected preserved rethrow cause")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected rethrow stack run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunFile_UncaughtErrorIncludesSourceLines(t *testing.T) {
	src := `
fn inner() {
  throw error("outer", error("inner"))
}

fn outer() {
  inner()
}

outer()
`

	dir := t.TempDir()
	path := filepath.Join(dir, "stack_lines.ic")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(path); err == nil {
		t.Fatalf("expected uncaught runtime error")
	} else {
		msg := err.Error()
		if !strings.Contains(msg, "outer") {
			t.Fatalf("expected outer message, got: %q", msg)
		}
		if !strings.Contains(msg, "Caused by: inner") {
			t.Fatalf("expected caused-by message, got: %q", msg)
		}
		if !strings.Contains(msg, "outer\n  at inner ("+path+":3)") {
			t.Fatalf("expected top segment before cause, got: %q", msg)
		}
		if !strings.Contains(msg, "Caused by: inner") {
			t.Fatalf("expected caused-by message, got: %q", msg)
		}
		if !strings.Contains(msg, "at inner ("+path+":3)") {
			t.Fatalf("expected inner frame with source line, got: %q", msg)
		}
		if !strings.Contains(msg, "at outer ("+path+":7)") {
			t.Fatalf("expected outer frame with source line, got: %q", msg)
		}
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
let causeMessage = ""

try {
  try {
    throw error("boom", error("inner"))
  } finally {
    out = "finally"
  }
} catch err {
  out = out + "/" + err.message
  causeMessage = err.cause.message
}

if out != "finally/boom" {
  panic("unexpected try/finally rethrow result")
}
if causeMessage != "inner" {
  panic("expected cause to survive finally rethrow")
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
    throw error("finally boom", error("root cause"))
  }
}

try {
  demo()
  panic("expected finally throw to override return")
} catch err {
  if err.message != "finally boom" {
    panic("unexpected finally override message")
  }
  if err.cause.message != "root cause" {
    panic("unexpected finally override cause")
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

func TestRuntimeStatsAndShutdown(t *testing.T) {
	rt := NewRuntime()
	if err := rt.ConfigureGoPool(2, 3); err != nil {
		t.Fatalf("configure go pool failed: %v", err)
	}

	src := `
let ch = chan(1)
go fn() {
  ch.send(7)
}()
if ch.recv() != 7 {
  panic("expected goroutine result")
}
`
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("run source failed: %v", err)
	}

	stats := rt.Stats()
	if stats.NumCPU < 1 {
		t.Fatalf("expected cpu count, got %d", stats.NumCPU)
	}
	if stats.NumGoroutine < 1 {
		t.Fatalf("expected goroutine count, got %d", stats.NumGoroutine)
	}
	if stats.Memory.HeapAllocBytes == 0 {
		t.Fatal("expected heap allocation stats")
	}
	if stats.Pool.Workers != 2 {
		t.Fatalf("expected 2 pool workers, got %d", stats.Pool.Workers)
	}
	if stats.Pool.QueueCapacity != 3 {
		t.Fatalf("expected queue capacity 3, got %d", stats.Pool.QueueCapacity)
	}
	if stats.Pool.Submitted < 1 {
		t.Fatalf("expected submitted tasks, got %d", stats.Pool.Submitted)
	}

	afterGC := rt.CollectGarbage()
	if afterGC.Memory.NumGC < stats.Memory.NumGC {
		t.Fatalf("expected gc count to stay monotonic, before=%d after=%d", stats.Memory.NumGC, afterGC.Memory.NumGC)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}
