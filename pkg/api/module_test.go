package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeRunFile_ImportsExportedModule(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "math.ic")
	mainPath := filepath.Join(dir, "main.ic")

	if err := os.WriteFile(modPath, []byte(`export const version = "icoo"
export fn add(a, b) {
  return a + b
}
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(`import "./math.ic" as math

let total = math.add(1, 2)
let name = math.version
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("expected import/export run to succeed, got: %v", err)
	}
}

func TestRuntimeRunFile_IteratesModuleExports(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "math.ic")
	mainPath := filepath.Join(dir, "main.ic")

	if err := os.WriteFile(modPath, []byte(`export const version = "icoo"
export const answer = 42
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(`import "./math.ic" as math

let keys = ""
let score = 0
for key, value in math {
  keys = keys + key
  if key == "answer" {
    score = score + value
  }
}

if keys != "answerversion" {
  panic("unexpected module iteration order")
}
if score != 42 {
  panic("unexpected module iteration values")
}
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("expected module iteration run to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdIOModule(t *testing.T) {
	src := `
import std.io as io

io.print("a")
io.println("b")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.io import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdIOModule(t *testing.T) {
	src := `
import std.io as io

let keys = ""
for key, value in io {
  keys = keys + key
  if typeOf(value) != "native_function" {
    panic("unexpected std.io export kind")
  }
}

if keys != "printprintln" {
  panic("unexpected std.io iteration order")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.io iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdTimeModule(t *testing.T) {
	src := `
import std.time as time

let before = time.now()
time.sleep(0)
let after = time.now()

if typeOf(before) != "int" {
  panic("std.time.now should return int")
}
if typeOf(after) != "int" {
  panic("std.time.now should return int")
}
if after < before {
  panic("time should not go backwards")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.time import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdTimeModule(t *testing.T) {
	src := `
import std.time as time

let keys = ""
for key, value in time {
  keys = keys + key
  if typeOf(value) != "native_function" {
    panic("unexpected std.time export kind")
  }
}

if keys != "nowsleep" {
  panic("unexpected std.time iteration order")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.time iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdTimeSleepRejectsNonInt(t *testing.T) {
	src := `
import std.time as time

time.sleep("1")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.time.sleep to reject non-int argument")
	}
}

func TestRuntimeRunSource_ImportsStdMathModule(t *testing.T) {
	src := `
import std.math as math

if math.abs(-3) != 3 {
  panic("unexpected abs int result")
}
if math.abs(-1.5) != 1.5 {
  panic("unexpected abs float result")
}
if math.max(4, 7) != 7 {
  panic("unexpected max result")
}
if math.min(4, 7) != 4 {
  panic("unexpected min result")
}
if math.floor(1.8) != 1 {
  panic("unexpected floor result")
}
if math.ceil(1.2) != 2 {
  panic("unexpected ceil result")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.math import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdMathModule(t *testing.T) {
	src := `
import std.math as math

let keys = ""
let count = 0
for key, value in math {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.math export kind")
  }
}

if keys != "absceilfloormaxmin" {
  panic("unexpected std.math iteration order")
}
if count != 5 {
  panic("unexpected std.math export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.math iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdMathRejectsNonNumericArgs(t *testing.T) {
	src := `
import std.math as math

math.abs("x")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.math.abs to reject non-numeric argument")
	}
}

func TestRuntimeCheckSource_ParsesImportExport(t *testing.T) {
	src := `
import "./math.ic" as math
export const answer = 42
`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) > 0 {
		t.Fatalf("expected no check errors, got %v", errs)
	}
}
