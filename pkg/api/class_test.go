package api

import (
	"testing"
)

func TestClassBasicInit(t *testing.T) {
	src := `
class Person {
  init(name, age) {
    this.name = name
    this.age = age
  }
}

let p = Person("Alice", 30)
if p.name != "Alice" {
  panic("expected Alice")
}
if p.age != 30 {
  panic("expected age 30")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class basic init failed: %v", err)
	}
}

func TestClassMethod(t *testing.T) {
	src := `
class Person {
  init(name) {
    this.name = name
  }
  greet() {
    return "Hello " + this.name
  }
}

let p = Person("Bob")
let msg = p.greet()
if msg != "Hello Bob" {
  panic("expected Hello Bob")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class method failed: %v", err)
	}
}

func TestClassMultipleMethods(t *testing.T) {
	src := `
class Counter {
  init() {
    this.value = 0
  }
  inc() {
    this.value = this.value + 1
    return this.value
  }
  dec() {
    this.value = this.value - 1
    return this.value
  }
}

let c = Counter()
if c.inc() != 1 {
  panic("expected 1")
}
if c.inc() != 2 {
  panic("expected 2")
}
if c.dec() != 1 {
  panic("expected 1")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class multiple methods failed: %v", err)
	}
}

func TestClassWithoutInit(t *testing.T) {
	src := `
class Empty {
  getValue() {
    return 42
  }
}

let e = Empty()
if e.getValue() != 42 {
  panic("expected 42")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class without init failed: %v", err)
	}
}

func TestClassMultipleInstances(t *testing.T) {
	src := `
class Point {
  init(x, y) {
    this.x = x
    this.y = y
  }
  sum() {
    return this.x + this.y
  }
}

let p1 = Point(1, 2)
let p2 = Point(10, 20)
if p1.sum() != 3 {
  panic("expected 3")
}
if p2.sum() != 30 {
  panic("expected 30")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class multiple instances failed: %v", err)
	}
}

func TestClassMethodWithParams(t *testing.T) {
	src := `
class Calculator {
  init(base) {
    this.base = base
  }
  add(n) {
    return this.base + n
  }
  multiply(n) {
    return this.base * n
  }
}

let calc = Calculator(10)
if calc.add(5) != 15 {
  panic("expected 15")
}
if calc.multiply(3) != 30 {
  panic("expected 30")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class method with params failed: %v", err)
	}
}

func TestClassWithGo(t *testing.T) {
	src := `
class Worker {
  init(ch) {
    this.ch = ch
  }
  doWork(val) {
    go fn() {
      this.ch.send(val)
    }()
  }
}

let ch = chan(1)
let w = Worker(ch)
w.doWork(42)
let result = ch.recv()
if result != 42 {
  panic("expected 42")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class with go failed: %v", err)
	}
}

func TestClassWithSelect(t *testing.T) {
	src := `
class Selector {
  init(ch) {
    this.ch = ch
  }
  tryRead() {
    let result = 0
    select {
      recv this.ch as v {
        result = v
      }
      else {
        result = -1
      }
    }
    return result
  }
}

let ch = chan(1)
ch.send(7)
let s = Selector(ch)
let val = s.tryRead()
if val != 7 {
  panic("expected 7")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class with select failed: %v", err)
	}
}

func TestClassExport(t *testing.T) {
	t.Skip("export test requires file-based testing")
}

func TestClassCheckDuplicateClass(t *testing.T) {
	src := `
class Foo {
  init() {}
}
class Foo {
  init() {}
}
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected duplicate class error")
	}
}
