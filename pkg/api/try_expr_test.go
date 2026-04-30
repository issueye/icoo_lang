package api

import (
	"testing"
)

func TestTryExprSimple(t *testing.T) {
	src := `
fn safe() {
  return 42
}
let result = safe()?
if result != 42 {
  panic("expected 42")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr simple failed: %v", err)
	}
}

func TestTryExprPropagatesError(t *testing.T) {
	src := `
fn inner() {
  return error("something failed")
}

fn outer() {
  let val = inner()?
  panic("should not reach")
}

let err = outer()
if err.message != "something failed" {
  panic("expected error propagation")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr propagate error failed: %v", err)
	}
}

func TestTryExprNested(t *testing.T) {
	src := `
fn deep() {
  return error("deep error")
}

fn middle() {
  return deep()?
}

fn outer() {
  return middle()?
}

let err = outer()
if err.message != "deep error" {
  panic("expected deep error")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr nested failed: %v", err)
	}
}

func TestTryExprWithNormalValue(t *testing.T) {
	src := `
fn getOrDefault(val, defaultVal) {
  let result = val?
  return result
}

fn maybeError() {
  return 100
}

let x = getOrDefault(maybeError(), 0)
if x != 100 {
  panic("expected 100")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr with normal value failed: %v", err)
	}
}

func TestTryExprWithChannel(t *testing.T) {
	src := `
fn recvOrError(ch) {
  return ch.recv()?
}

let ch = chan(1)
ch.send(99)
let result = recvOrError(ch)
if result != 99 {
  panic("expected 99")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr with channel failed: %v", err)
	}
}

func TestTryExprChained(t *testing.T) {
	src := `
fn step1(v) {
  if v < 0 {
    return error("negative")
  }
  return v * 2
}

fn step2(v) {
  return v + 10
}

fn pipeline(input) {
  let a = step1(input)?
  let b = step2(a)
  return b
}

let ok = pipeline(5)
if ok != 20 {
  panic("expected 20")
}

let err = pipeline(-1)
if typeOf(err) != "error" {
  panic("expected error")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr chained failed: %v", err)
	}
}

func TestTryExprWithClass(t *testing.T) {
	src := `
class Validator {
  init() {
    this.threshold = 10
  }
  check(v) {
    if v < this.threshold {
      return error("too small")
    }
    return v
  }
}

fn process(v) {
  let validator = Validator()
  return validator.check(v)?
}

let ok = process(15)
if ok != 15 {
  panic("expected 15")
}

let err = process(5)
if typeOf(err) != "error" {
  panic("expected error")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr with class failed: %v", err)
	}
}

func TestTryExprNotErrorType(t *testing.T) {
	src := `
fn getString() {
  return "hello"
}

let result = getString()?
if result != "hello" {
  panic("expected hello")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("try expr with string failed: %v", err)
	}
}
