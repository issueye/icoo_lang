package api

import (
	"testing"
)

func TestLogicalAndSimple(t *testing.T) {
	src := `
if true && true {
  println("ok")
} else {
  panic("expected ok")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("&& simple failed: %v", err)
	}
}

func TestLogicalAndFalse(t *testing.T) {
	src := `
if true && false {
  panic("should not reach")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("&& false failed: %v", err)
	}
}

func TestLogicalOrSimple(t *testing.T) {
	src := `
if false || true {
  println("ok")
} else {
  panic("expected ok")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("|| simple failed: %v", err)
	}
}

func TestLogicalOrFalse(t *testing.T) {
	src := `
if false || false {
  panic("should not reach")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("|| false failed: %v", err)
	}
}

func TestLogicalAndShortCircuit(t *testing.T) {
	src := `
let sideEffect = false
fn setFlag() {
  sideEffect = true
  return true
}
let result = false && setFlag()
if sideEffect {
  panic("&& should short-circuit, sideEffect should be false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("&& short-circuit failed: %v", err)
	}
}

func TestLogicalOrShortCircuit(t *testing.T) {
	src := `
let sideEffect = false
fn setFlag() {
  sideEffect = true
  return true
}
let result = true || setFlag()
if sideEffect {
  panic("|| should short-circuit, sideEffect should be false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("|| short-circuit failed: %v", err)
	}
}

func TestLogicalAndReturnValue(t *testing.T) {
	src := `
let a = 1 && 2
let b = 0 && 3
let c = "" && "hello"
let d = "x" && "y"
if a != 2 {
  panic("1 && 2 should be 2")
}
if b != 0 {
  panic("0 && 3 should be 0")
}
if c != "" {
  panic("empty && hello should be empty")
}
if d != "y" {
  panic("x && y should be y")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("&& return value failed: %v", err)
	}
}

func TestLogicalOrReturnValue(t *testing.T) {
	src := `
let a = 1 || 2
let b = 0 || 3
let c = "" || "default"
if a != 1 {
  panic("1 || 2 should be 1")
}
if b != 3 {
  panic("0 || 3 should be 3")
}
if c != "default" {
  panic("empty || default should be default")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("|| return value failed: %v", err)
	}
}

func TestLogicalMixed(t *testing.T) {
	src := `
let a = true && false || true
let b = false || true && false
if !a {
  panic("T && F || T should be true")
}
if b {
  panic("F || T && F should be false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("mixed logical failed: %v", err)
	}
}

func TestLogicalInIfCondition(t *testing.T) {
	src := `
let x = 5
let result = ""
if x > 0 && x < 10 {
  result = "yes"
} else {
  result = "no"
}
if result != "yes" {
  panic("expected yes")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("logical in if failed: %v", err)
	}
}

func TestLogicalComplex(t *testing.T) {
	src := `
let a = 1
let b = 2
let c = 3
let r1 = a > 0 && b > 0 && c > 0
let r2 = a > 10 || b > 10 || c > 10
let r3 = a > 10 || b > 1 && c < 2
if !r1 {
  panic("all positive should be true")
}
if r2 {
  panic("none > 10 should be false")
}
if r3 {
  panic("a>10 || b>1 && c<2 should be false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("complex logical failed: %v", err)
	}
}

func TestLogicalWithClass(t *testing.T) {
	src := `
class Validator {
  init(min, max) {
    this.min = min
    this.max = max
  }
  check(n) {
    return n >= this.min && n <= this.max
  }
}

let v = Validator(1, 10)
if !v.check(5) {
  panic("5 should be valid")
}
if v.check(0) {
  panic("0 should not be valid")
}
if v.check(11) {
  panic("11 should not be valid")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("logical with class failed: %v", err)
	}
}

func TestLogicalShortCircuitDoesNotLeakStackValue(t *testing.T) {
	src := `
let rows = [
  {name: "", value: null},
  {name: "target", value: "ok"}
]

let found = false
let i = 0
for i < len(rows) {
  let row = rows[i]
  if row.name == "target" && row.value == "ok" {
    found = true
  }
  i = i + 1
}

if found != true {
  panic("short-circuit result should not leak past block cleanup")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("logical short-circuit stack cleanup failed: %v", err)
	}
}
