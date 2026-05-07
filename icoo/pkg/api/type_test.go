package api

import (
	"testing"
)

func TestTypeDeclSimple(t *testing.T) {
	src := `
type UserID = int
type Name = string
let id = 42
let name = "Alice"
println("ok")
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("type decl failed: %v", err)
	}
}

func TestTypeDeclDuplicate(t *testing.T) {
	src := `
type Foo = int
type Foo = string
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected duplicate type error")
	}
}

func TestTypeDeclExport(t *testing.T) {
	t.Skip("export requires file-based test")
}

func TestInterfaceDeclSimple(t *testing.T) {
	src := `
interface Writer {
  write(data string)
}

interface Reader {
  read() string
}

println("ok")
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("interface decl failed: %v", err)
	}
}

func TestInterfaceDeclDuplicate(t *testing.T) {
	src := `
interface Foo {
  bar()
}
interface Foo {
  baz()
}
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected duplicate interface error")
	}
}

func TestInterfaceDeclWithReturnType(t *testing.T) {
	src := `
interface Greeter {
  greet(name string) string
  getCount() int
}

println("ok")
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("interface with return type failed: %v", err)
	}
}

func TestSatisfiesTrue(t *testing.T) {
	src := `
interface Writer {
  write(data string)
}

let obj = {write: fn(s) {}}
let ok = satisfies(obj, Writer)
if !ok {
  panic("expected true")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("satisfies true failed: %v", err)
	}
}

func TestSatisfiesFalse(t *testing.T) {
	src := `
interface Writer {
  write(data string)
}

let obj = {say: fn(s) {}}
let ok = satisfies(obj, Writer)
if ok {
  panic("expected false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("satisfies false failed: %v", err)
	}
}

func TestSatisfiesNonObject(t *testing.T) {
	src := `
interface Writer {
  write(data string)
}

let ok = satisfies(42, Writer)
if ok {
  panic("expected false for non-object")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("satisfies non-object failed: %v", err)
	}
}

func TestSatisfiesMultipleMethods(t *testing.T) {
	src := `
interface ReadWriter {
  read() string
  write(data string)
}

let rw = {
  read: fn() { return "hello" },
  write: fn(s) {}
}
if !satisfies(rw, ReadWriter) {
  panic("expected true for both methods")
}

let rOnly = {
  read: fn() { return "hello" }
}
if satisfies(rOnly, ReadWriter) {
  panic("expected false for missing write")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("satisfies multiple methods failed: %v", err)
	}
}

func TestTypeAndInterfaceInRepl(t *testing.T) {
	// Simulates REPL usage across multiple runs
	rt := NewRuntime()

	// Define interface
	_, err := rt.RunReplLine("interface Greeter { greet(name string) string }")
	if err != nil {
		t.Fatalf("define interface: %v", err)
	}

	// Define type
	_, err = rt.RunReplLine("type UserID = int")
	if err != nil {
		t.Fatalf("define type: %v", err)
	}

	// Use them
	_, err = rt.RunReplLine("let obj = {greet: fn(n) { return n }}")
	if err != nil {
		t.Fatalf("create object: %v", err)
	}

	result, err := rt.RunReplLine("satisfies(obj, Greeter)")
	if err != nil {
		t.Fatalf("check satisfies: %v", err)
	}
	if result.String() != "true" {
		t.Fatalf("expected true, got %s", result.String())
	}
}
