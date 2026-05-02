package api

import "testing"

func TestRuntimeRunSource_ArrayNativeMethods(t *testing.T) {
	src := `
let factor = 3
let arr = [1, 2, 3, 4]

let mapped = arr.map(fn(value, index, source) {
  if source != arr {
    panic("map should receive the original array")
  }
  return value * factor + index
})

if len(mapped) != 4 {
  panic("unexpected map result length")
}
if mapped[0] != 3 || mapped[1] != 7 || mapped[2] != 11 || mapped[3] != 15 {
  panic("unexpected map result values")
}

let filtered = arr.filter(fn(value, index, source) {
  if source != arr {
    panic("filter should receive the original array")
  }
  return value % 2 == 0 && index >= 1
})

if len(filtered) != 2 || filtered[0] != 2 || filtered[1] != 4 {
  panic("unexpected filter result")
}

let found = arr.find(fn(value, index, source) {
  if source != arr {
    panic("find should receive the original array")
  }
  return value == 3 && index == 2
})
if found != 3 {
  panic("unexpected find result")
}

let missing = arr.find(fn(value) {
  return value == 99
})
if missing != null {
  panic("missing find should return null")
}

let foundIndex = arr.findIndex(fn(value) {
  return value == 4
})
if foundIndex != 3 {
  panic("unexpected findIndex result")
}

let missingIndex = arr.findIndex(fn(value) {
  return value == 99
})
if missingIndex != -1 {
  panic("missing findIndex should return -1")
}

if !arr.some(fn(value) { return value == 2 }) {
  panic("some should return true when a match exists")
}
if arr.some(fn(value) { return value == 99 }) {
  panic("some should return false when no match exists")
}
if !arr.every(fn(value) { return value >= 1 }) {
  panic("every should return true when all items match")
}
if arr.every(fn(value) { return value < 4 }) {
  panic("every should return false when an item does not match")
}

let sum = 0
let forEachResult = arr.forEach(fn(value, index, source) {
  if source != arr {
    panic("forEach should receive the original array")
  }
  sum = sum + value + index
})

if forEachResult != null {
  panic("forEach should return null")
}
if sum != 16 {
  panic("forEach callback should be able to mutate outer scope")
}

let reduced = arr.reduce(fn(acc, value, index, source) {
  if source != arr {
    panic("reduce should receive the original array")
  }
  return acc + value + index
}, 10)

if reduced != 26 {
  panic("unexpected reduce result with initial value")
}

let multiplied = [2, 3, 4].reduce(fn(acc, value) {
  return acc * value
})
if multiplied != 24 {
  panic("unexpected reduce result without initial value")
}

if !arr.includes(3) {
  panic("includes should find existing value")
}
if arr.includes(99) {
  panic("includes should return false for missing value")
}
if arr.indexOf(3) != 2 {
  panic("unexpected indexOf result")
}
if arr.indexOf(99) != -1 {
  panic("missing indexOf should return -1")
}

let caughtReduceError = false
try {
  [].reduce(fn(acc, value) {
    return acc + value
  })
} catch err {
  caughtReduceError = true
}

if !caughtReduceError {
  panic("reduce without initial value on empty array should throw")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected array native methods run to succeed, got error: %v", err)
	}
}
