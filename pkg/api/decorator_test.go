package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecoratorWrapsFunction(t *testing.T) {
	src := `
fn excited(target) {
  return fn() {
    return target() + "!"
  }
}

@excited
fn greet() {
  return "hi"
}

if greet() != "hi!" {
  panic("expected decorated function result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("decorated function failed: %v", err)
	}
}

func TestDecoratorOrderAndArguments(t *testing.T) {
	src := `
fn prefix(text) {
  return fn(target) {
    return fn() {
      return text + target()
    }
  }
}

@prefix("A")
@prefix("B")
fn greet() {
  return "C"
}

if greet() != "ABC" {
  panic("expected decorator application order")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("decorator order failed: %v", err)
	}
}

func TestDecoratorWorksForLocalFunction(t *testing.T) {
	src := `
fn wrap(target) {
  return fn() {
    return target() + " local"
  }
}

fn outer() {
  @wrap
  fn greet() {
    return "ok"
  }
  return greet()
}

if outer() != "ok local" {
  panic("expected local decorator")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("local decorator failed: %v", err)
	}
}

func TestDecoratorWrapsClass(t *testing.T) {
	src := `
fn tag(target) {
  return fn(value) {
    let obj = target(value)
    obj.decorated = true
    return obj
  }
}

@tag
class Box {
  init(value) {
    this.value = value
  }
}

let box = Box(7)
if box.value != 7 {
  panic("expected class init through decorator")
}
if !box.decorated {
  panic("expected decorated marker")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("class decorator failed: %v", err)
	}
}

func TestDecoratorExportedFunction(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "decorated.ic")
	mainPath := filepath.Join(dir, "main.ic")

	modSrc := `fn suffix(text) {
  return fn(target) {
    return fn() {
      return target() + text
    }
  }
}

export @suffix("!")
fn greet() {
  return "hello"
}
`
	if err := os.WriteFile(modPath, []byte(modSrc), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	mainSrc := `import "./decorated.ic" as decorated

if decorated.greet() != "hello!" {
  panic("expected exported decorated function")
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("exported decorator failed: %v", err)
	}
}
