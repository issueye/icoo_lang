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

func TestClassDefaultFieldDeclaration(t *testing.T) {
	src := `
class Person {
  name = ""
  age = 18
}

let p = Person()
if p.name != "" {
  panic("expected default name")
}
if p.age != 18 {
  panic("expected default age")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class default field declaration failed: %v", err)
	}
}

func TestClassDefaultFieldCanBeOverriddenInInit(t *testing.T) {
	src := `
class Person {
  name = "unknown"
  age = 0

  init(name) {
    this.name = name
    this.age = this.age + 20
  }
}

let p = Person("Ada")
if p.name != "Ada" {
  panic("expected init override name")
}
if p.age != 20 {
  panic("expected init to see default age")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class default field init override failed: %v", err)
	}
}

func TestClassDefaultFieldUsesDistinctMutableInstances(t *testing.T) {
	src := `
class Box {
  items = []
  meta = {
    count: 0
  }

  push(value) {
    this.items = this.items.append(value)
    this.meta.count = this.meta.count + 1
  }
}

let a = Box()
let b = Box()
a.push(1)

if len(a.items) != 1 {
  panic("expected first instance item")
}
if len(b.items) != 0 {
  panic("expected second instance isolated items")
}
if a.meta.count != 1 {
  panic("expected first instance meta count")
}
if b.meta.count != 0 {
  panic("expected second instance isolated meta count")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class mutable default field isolation failed: %v", err)
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

func TestClassInheritanceInheritsParentMethods(t *testing.T) {
	src := `
class Animal {
  init(name) {
    this.name = name
  }
  speak() {
    return "Animal:" + this.name
  }
}

class Dog <- Animal {
  wag() {
    return this.name
  }
}

let dog = Dog("Milo")
if dog.speak() != "Animal:Milo" {
  panic("expected inherited speak")
}
if dog.wag() != "Milo" {
  panic("expected child method")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class inheritance failed: %v", err)
	}
}

func TestClassSuperMethodAndInit(t *testing.T) {
	src := `
class Animal {
  init(name) {
    this.name = name
  }
  speak() {
    return "Animal:" + this.name
  }
}

class Dog <- Animal {
  init(name, breed) {
    super.init(name)
    this.breed = breed
  }
  speak() {
    return super.speak() + ":" + this.breed
  }
}

let dog = Dog("Milo", "corgi")
if dog.name != "Milo" {
  panic("expected super init to set name")
}
if dog.speak() != "Animal:Milo:corgi" {
  panic("expected super speak call")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class super failed: %v", err)
	}
}

func TestClassSuperInNestedClosure(t *testing.T) {
	src := `
class Base {
  greet() {
    return "base"
  }
}

class Child <- Base {
  greet() {
    let callSuper = fn() {
      return super.greet()
    }
    return callSuper() + "-child"
  }
}

let value = Child().greet()
if value != "base-child" {
  panic("expected nested super call")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("nested super failed: %v", err)
	}
}

func TestClassSuperRequiresSubclass(t *testing.T) {
	src := `
class Person {
  greet() {
    return super.greet()
  }
}
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected super usage error")
	}
}

func TestClassMethodDecorator(t *testing.T) {
	src := `
fn excited(target) {
  return fn() {
    return target() + "!"
  }
}

class Greeter {
  @excited
  greet() {
    return "hi"
  }
}

if Greeter().greet() != "hi!" {
  panic("expected decorated method")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class method decorator failed: %v", err)
	}
}

func TestClassMethodDecoratorOrder(t *testing.T) {
	src := `
fn prefix(text) {
  return fn(target) {
    return fn() {
      return text + target()
    }
  }
}

class Greeter {
  @prefix("A")
  @prefix("B")
  greet() {
    return "C"
  }
}

if Greeter().greet() != "ABC" {
  panic("expected method decorator order")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class method decorator order failed: %v", err)
	}
}

func TestClassInitDecorator(t *testing.T) {
	src := `
fn withFlag(target) {
  return fn(name) {
    let obj = target(name)
    obj.decorated = true
    return obj
  }
}

class Person {
  @withFlag
  init(name) {
    this.name = name
  }
}

let p = Person("Ada")
if p.name != "Ada" {
  panic("expected init to run")
}
if !p.decorated {
  panic("expected init decorator to run")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class init decorator failed: %v", err)
	}
}
