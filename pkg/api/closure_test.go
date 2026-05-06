package api

import (
	"testing"
)

// ---- Basic closure capture ----

func TestClosureCaptureSimple(t *testing.T) {
	src := `
fn outer() {
  let x = 10
  let inner = fn() {
    return x
  }
  return inner()
}
let result = outer()
if result != 10 {
  panic("expected 10")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("simple capture failed: %v", err)
	}
}

func TestClosureCaptureModifyUpvalue(t *testing.T) {
	src := `
fn outer() {
  let x = 10
  let inner = fn() {
    x = x + 5
  }
  inner()
  return x
}
let result = outer()
if result != 15 {
  panic("expected 15")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("modify upvalue failed: %v", err)
	}
}

func TestClosureCaptureMultipleUpvalues(t *testing.T) {
	src := `
fn outer() {
  let a = 1
  let b = 2
  let inner = fn() {
    a = a + b
  }
  inner()
  inner()
  return a
}
let result = outer()
if result != 5 {
  panic("expected 5")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("multiple upvalues failed: %v", err)
	}
}

func TestClosureCaptureInLoop(t *testing.T) {
	src := `
fn makeCounter() {
  let count = 0
  return fn() {
    count = count + 1
    return count
  }
}

let counter = makeCounter()
let a = counter()
let b = counter()
let c = counter()
if a != 1 {
  panic("expected a=1")
}
if b != 2 {
  panic("expected b=2")
}
if c != 3 {
  panic("expected c=3")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("closure in loop failed: %v", err)
	}
}

func TestClosureCaptureParameter(t *testing.T) {
	src := `
fn makeAdder(n) {
  return fn(x) {
    return x + n
  }
}

let add5 = makeAdder(5)
let add10 = makeAdder(10)
if add5(3) != 8 {
  panic("expected 8")
}
if add10(3) != 13 {
  panic("expected 13")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("capture parameter failed: %v", err)
	}
}

func TestClosureCaptureDeepNested(t *testing.T) {
	src := `
fn outer() {
  let x = 1
  let middle = fn() {
    let inner = fn() {
      x = x * 2
    }
    inner()
  }
  middle()
  middle()
  return x
}
let result = outer()
if result != 4 {
  panic("expected 4")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("deep nested capture failed: %v", err)
	}
}

func TestClosureCaptureWithGo(t *testing.T) {
	src := `
fn startWorker(ch, val) {
  go fn() {
    ch.send(val)
  }()
}

let ch = chan(1)
startWorker(ch, 42)
let v = ch.recv()
if v != 42 {
  panic("expected 42")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("closure capture with go failed: %v", err)
	}
}

func TestClosureCaptureWithSelect(t *testing.T) {
	src := `
fn process(ch, multiplier) {
  select {
    recv ch as v {
      return v * multiplier
    }
  }
  return 0
}

let ch = chan(1)
ch.send(10)
let result = process(ch, 3)
if result != 30 {
  panic("expected 30")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("closure capture with select failed: %v", err)
	}
}

func TestClosureCaptureInGoBody(t *testing.T) {
	src := `
let ch = chan(2)
let prefix = "hello"

go fn() {
  ch.send(prefix)
}()

let msg = ch.recv()
if msg != "hello" {
  panic("expected 'hello'")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("closure capture in go body failed: %v", err)
	}
}

func TestClosureCaptureTwoLevels(t *testing.T) {
	src := `
fn factory(base) {
  return fn(inc) {
    return base + inc
  }
}

let add10 = factory(10)
if add10(5) != 15 {
  panic("expected 15")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("two-level capture failed: %v", err)
	}
}

// ---- Error cases ----

func TestClosureCaptureFunctionBodyTopLevelLocal(t *testing.T) {
	src := `
fn factory() {
  let prefix = "hi"
  fn inner(name) {
    return prefix + ", " + name
  }
  return inner
}

let greet = factory()
let result = greet("icoo")
if result != "hi, icoo" {
  panic("expected function body top-level local capture")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("function body top-level local capture failed: %v", err)
	}
}

func TestClosureCaptureFunctionBodyLocalRecursion(t *testing.T) {
	src := `
fn factory() {
  fn walk(n) {
    if n <= 0 {
      return 0
    }
    return n + walk(n - 1)
  }

  return walk(4)
}

if factory() != 10 {
  panic("expected recursive local function to work")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("function body local recursion failed: %v", err)
	}
}

func TestClosureCaptureNonExistent(t *testing.T) {
	src := `
fn outer() {
  let inner = fn() {
    return x
  }
  return inner()
}
outer()
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err == nil {
		t.Fatal("expected error for undefined capture 'x'")
	}
}
