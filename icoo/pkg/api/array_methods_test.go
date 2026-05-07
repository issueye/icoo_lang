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

let appended = arr.append(5)
if len(appended) != 5 || appended[4] != 5 {
  panic("append should add one item to the end")
}
if len(arr) != 4 {
  panic("append should not mutate original array")
}

let concatenated = arr.concat([5, 6])
if len(concatenated) != 6 {
  panic("concat should produce merged array")
}
if concatenated[4] != 5 || concatenated[5] != 6 {
  panic("concat should preserve right-hand array order")
}
if len(arr) != 4 {
  panic("concat should not mutate original array")
}

let concatError = false
try {
  arr.concat("bad")
} catch err {
  concatError = true
}
if !concatError {
  panic("concat should reject non-array input")
}

let prepended = arr.prepend(0)
if len(prepended) != 5 || prepended[0] != 0 || prepended[1] != 1 {
  panic("prepend should add one item to the front")
}
if len(arr) != 4 {
  panic("prepend should not mutate original array")
}

let slicedAll = arr.slice()
if len(slicedAll) != 4 || slicedAll[0] != 1 || slicedAll[3] != 4 {
  panic("slice without args should copy whole array")
}

let slicedTail = arr.slice(1)
if len(slicedTail) != 3 || slicedTail[0] != 2 || slicedTail[2] != 4 {
  panic("slice(start) should return tail")
}

let slicedRange = arr.slice(1, 3)
if len(slicedRange) != 2 || slicedRange[0] != 2 || slicedRange[1] != 3 {
  panic("slice(start, end) should return half-open range")
}

let slicedNegative = arr.slice(-2)
if len(slicedNegative) != 2 || slicedNegative[0] != 3 || slicedNegative[1] != 4 {
  panic("slice should support negative start index")
}

let slicedClamped = arr.slice(-99, 99)
if len(slicedClamped) != 4 {
  panic("slice should clamp indexes into array bounds")
}

let sliceError = false
try {
  arr.slice("bad")
} catch err {
  sliceError = true
}
if !sliceError {
  panic("slice should reject non-int start")
}

let flatMapped = arr.flatMap(fn(value) {
  return [value, value * 10]
})
if len(flatMapped) != 8 || flatMapped[0] != 1 || flatMapped[1] != 10 || flatMapped[6] != 4 || flatMapped[7] != 40 {
  panic("flatMap should flatten one array level")
}

let flatMappedScalars = arr.flatMap(fn(value) {
  return value * 2
})
if len(flatMappedScalars) != 4 || flatMappedScalars[0] != 2 || flatMappedScalars[3] != 8 {
  panic("flatMap should keep scalar callback results")
}

let taken = arr.take(2)
if len(taken) != 2 || taken[0] != 1 || taken[1] != 2 {
  panic("take should keep first n items")
}

let takeClamped = arr.take(99)
if len(takeClamped) != 4 {
  panic("take should clamp count to array length")
}

let takeNegative = arr.take(-5)
if len(takeNegative) != 0 {
  panic("take should clamp negative count to zero")
}

let takeError = false
try {
  arr.take("bad")
} catch err {
  takeError = true
}
if !takeError {
  panic("take should reject non-int count")
}

let dropped = arr.drop(2)
if len(dropped) != 2 || dropped[0] != 3 || dropped[1] != 4 {
  panic("drop should skip first n items")
}

let dropClamped = arr.drop(99)
if len(dropClamped) != 0 {
  panic("drop should clamp count to array length")
}

let dropNegative = arr.drop(-5)
if len(dropNegative) != 4 {
  panic("drop should clamp negative count to zero")
}

let dropError = false
try {
  arr.drop("bad")
} catch err {
  dropError = true
}
if !dropError {
  panic("drop should reject non-int count")
}

if arr.first() != 1 {
  panic("first should return first item")
}
if arr.last() != 4 {
  panic("last should return last item")
}
if [].first() != null {
  panic("first on empty array should return null")
}
if [].last() != null {
  panic("last on empty array should return null")
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
