package api

import "testing"

func TestRuntimeRunSource_ForLoopBreakContinue(t *testing.T) {
	src := `
let i = 0
let sum = 0

for i < 10 {
  i = i + 1

  if i == 3 {
    continue
  }

  if i == 8 {
    break
  }

  sum = sum + i
}

if sum != 25 {
  panic("unexpected for-loop sum")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected for loop run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_InfiniteForWithBreak(t *testing.T) {
	src := `
let i = 0

for {
  i = i + 1
  if i == 2 {
    break
  }
}

if i != 2 {
  panic("unexpected infinite-for counter")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected infinite for with break to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInArray(t *testing.T) {
	src := `
let arr = [1, 2, 3, 4]
let sum = 0

for item in arr {
  sum = sum + item
}

if sum != 10 {
  panic("unexpected for-in sum")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected for-in loop run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInArrayWithContinue(t *testing.T) {
	src := `
let arr = [1, 2, 3, 4]
let sum = 0

for item in arr {
  if item == 2 {
    continue
  }
  sum = sum + item
}

if sum != 8 {
  panic("unexpected for-in continue sum")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected for-in continue run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInString(t *testing.T) {
	src := `
let out = ""

for ch in "你好a" {
  out = out + ch
}

if out != "你好a" {
  panic("unexpected string iteration")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected string for-in to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInObjectKeys(t *testing.T) {
	src := `
let obj = {b: 2, a: 1, c: 3}
let keys = ""

for key in obj {
  keys = keys + key
}

if keys != "abc" {
  panic("unexpected object iteration order")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected object for-in to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ArrayIteratorProtocol(t *testing.T) {
	src := `
let iter = [4, 5].iter()
let first = iter.next()
let second = iter.next()
let third = iter.next()

if first.done {
  panic("first step should not be done")
}
if first.value != 4 {
  panic("unexpected first value")
}
if second.done {
  panic("second step should not be done")
}
if second.value != 5 {
  panic("unexpected second value")
}
if !third.done {
  panic("third step should be done")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected explicit iterator protocol to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_StringIteratorProtocol(t *testing.T) {
	src := `
let iter = "ab".iter()
let first = iter.next()
let second = iter.next()
let third = iter.next()

if first.done {
  panic("first string step should not be done")
}
if first.value != "a" {
  panic("unexpected first string step")
}
if second.done {
  panic("second string step should not be done")
}
if second.value != "b" {
  panic("unexpected second string step")
}
if !third.done {
  panic("third string step should be done")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected string iterator protocol to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ObjectIteratorProtocol(t *testing.T) {
	src := `
let iter = {z: 1, x: 2}.iter()
let first = iter.next()
let second = iter.next()
let third = iter.next()

if first.done {
  panic("first object step should not be done")
}
if first.value != "x" {
  panic("unexpected first object key")
}
if second.done {
  panic("second object step should not be done")
}
if second.value != "z" {
  panic("unexpected second object key")
}
if !third.done {
  panic("third object step should be done")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected object iterator protocol to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_DirectIteratorForIn(t *testing.T) {
	src := `
let iter = [1, 2, 3].iter()
let sum = 0

for item in iter {
  sum = sum + item
}

if sum != 6 {
  panic("unexpected direct iterator for-in")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected direct iterator for-in to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_MixedIteratorKinds(t *testing.T) {
	src := `
let text = "ab"
let out = ""

for ch in text {
  out = out + ch
}

let obj = {b: 2, a: 1}
let keys = ""

for key in obj {
  keys = keys + key
}

if out != "ab" {
  panic("unexpected mixed string iteration")
}
if keys != "ab" {
  panic("unexpected mixed object iteration")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected mixed iterator kinds to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_MatchLiteralAndWildcard(t *testing.T) {
	src := `
let x = 2
let y = 0

match x {
  1 {
    y = 10
  }
  2 {
    y = 20
  }
  _ {
    y = 30
  }
}

if y != 20 {
  panic("unexpected match result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected match run to succeed, got error: %v", err)
	}
}

func TestRuntimeCheckSource_RejectsWildcardBeforeLastMatchArm(t *testing.T) {
	src := `
match 1 {
  _ {
  }
  1 {
  }
}
`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatalf("expected invalid wildcard arm order to fail check")
	}
}

func TestRuntimeCheckSource_RejectsBreakOutsideLoop(t *testing.T) {
	src := `break`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatalf("expected break outside loop to fail check")
	}
}

func TestRuntimeCheckSource_RejectsContinueOutsideLoop(t *testing.T) {
	src := `continue`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatalf("expected continue outside loop to fail check")
	}
}
