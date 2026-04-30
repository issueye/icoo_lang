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

func TestRuntimeRunSource_BreakContinueAcrossTryDoNotLeakHandlers(t *testing.T) {
	src := `
let i = 0
let sum = 0

for i < 6 {
  i = i + 1

  try {
    if i == 2 {
      continue
    }
    if i == 5 {
      break
    }
    sum = sum + i
  } catch err {
    panic(err.message)
  }
}

if sum != 8 {
  panic("unexpected try loop sum")
}

len(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatalf("expected final uncaught runtime error after loop")
	}
}

func TestRuntimeRunSource_ThrowInsideLoopTryIsCaught(t *testing.T) {
	src := `
let i = 0
let sum = 0

for i < 4 {
  i = i + 1

  try {
    if i == 2 {
      throw "skip"
    }
    sum = sum + i
  } catch err {
    if err.message != "skip" {
      panic("unexpected loop throw message")
    }
  }
}

if sum != 8 {
  panic("unexpected loop throw sum")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected loop throw catch run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_BreakRunsFinally(t *testing.T) {
	src := `
let i = 0
let out = ""

for i < 4 {
  i = i + 1
  try {
    out = out + "t"
    if i == 2 {
      break
    }
  } finally {
    out = out + "f"
  }
}

if out != "tftf" {
  panic("unexpected break/finally trace")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected break through finally run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ContinueRunsFinally(t *testing.T) {
	src := `
let i = 0
let out = ""

for i < 3 {
  i = i + 1
  try {
    out = out + "t"
    if i < 3 {
      continue
    }
    out = out + "x"
  } finally {
    out = out + "f"
  }
}

if out != "tftftxf" {
  panic("unexpected continue/finally trace")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected continue through finally run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_FinallyLoopCleanupDoesNotLeakHandlers(t *testing.T) {
	src := `
let i = 0

for i < 2 {
  i = i + 1
  try {
    if i == 1 {
      continue
    }
  } finally {
    let marker = i
    if marker < 0 {
      panic("unreachable")
    }
  }
}

len(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatalf("expected final uncaught runtime error after finally loop")
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
let total = 0

for pair in obj {
  keys = keys + pair.key
  total = total + pair.value
}

if keys != "abc" {
  panic("unexpected object iteration order")
}
if total != 6 {
  panic("unexpected object iteration values")
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
if first.key != 0 {
  panic("unexpected first key")
}
if first.value != 4 {
  panic("unexpected first value")
}
if second.done {
  panic("second step should not be done")
}
if second.key != 1 {
  panic("unexpected second key")
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
if first.key != 0 {
  panic("unexpected first string key")
}
if first.value != "a" {
  panic("unexpected first string step")
}
if second.done {
  panic("second string step should not be done")
}
if second.key != 1 {
  panic("unexpected second string key")
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
if first.key != "x" {
  panic("unexpected first object key")
}
if first.value != 2 {
  panic("unexpected first object value")
}
if first.item.key != "x" {
  panic("unexpected first object item key")
}
if first.item.value != 2 {
  panic("unexpected first object item value")
}
if second.done {
  panic("second object step should not be done")
}
if second.key != "z" {
  panic("unexpected second object key")
}
if second.value != 1 {
  panic("unexpected second object value")
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

func TestRuntimeRunSource_ForInObjectKeyValueBindings(t *testing.T) {
	src := `
let obj = {b: 2, a: 1, c: 3}
let keys = ""
let total = 0

for key, value in obj {
  keys = keys + key
  total = total + value
}

if keys != "abc" {
  panic("unexpected object key bindings")
}
if total != 6 {
  panic("unexpected object value bindings")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected object key/value for-in to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInArrayKeyValueBindings(t *testing.T) {
	src := `
let arr = [4, 5, 6]
let idxSum = 0
let valueSum = 0

for idx, value in arr {
  idxSum = idxSum + idx
  valueSum = valueSum + value
}

if idxSum != 3 {
  panic("unexpected array index bindings")
}
if valueSum != 15 {
  panic("unexpected array value bindings")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected array key/value for-in to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ForInStringKeyValueBindings(t *testing.T) {
	src := `
let text = "ab"
let idxSum = 0
let out = ""

for idx, ch in text {
  idxSum = idxSum + idx
  out = out + ch
}

if idxSum != 1 {
  panic("unexpected string index bindings")
}
if out != "ab" {
  panic("unexpected string value bindings")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected string key/value for-in to succeed, got error: %v", err)
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

func TestRuntimeRunSource_DirectIteratorKeyValueBindings(t *testing.T) {
	src := `
let iter = [7, 8].iter()
let idxSum = 0
let valueSum = 0

for idx, value in iter {
  idxSum = idxSum + idx
  valueSum = valueSum + value
}

if idxSum != 1 {
  panic("unexpected direct iterator index bindings")
}
if valueSum != 15 {
  panic("unexpected direct iterator value bindings")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected direct iterator key/value for-in to succeed, got error: %v", err)
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

for pair in obj {
  keys = keys + pair.key
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

func TestRuntimeRunSource_CustomObjectIterOverride(t *testing.T) {
	src := `
let obj = {
  label: "fallback",
  iter: fn() {
    return ["x", "y"].iter()
  }
}

let out = ""
for item in obj {
  out = out + item
}

if out != "xy" {
  panic("custom iter should override default object iteration")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected custom object iter override to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_ModuleLikePairIterationShape(t *testing.T) {
	src := `
let obj = {answer: 42}
let iter = obj.iter()
let first = iter.next()

if first.done {
  panic("first pair step should not be done")
}
if first.key != "answer" {
  panic("unexpected pair key")
}
if first.value != 42 {
  panic("unexpected pair value")
}
if first.item.key != "answer" {
  panic("unexpected pair item key")
}
if first.item.value != 42 {
  panic("unexpected pair item value")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected pair iteration shape to succeed, got error: %v", err)
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
