package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestRuntimeRunFile_ImportsProjectRootAliasModule(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "src", "main.ic")
	modPath := filepath.Join(dir, "utils", "math.ic")

	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(modPath), 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	if err := os.WriteFile(modPath, []byte(`export fn add(a, b) {
  return a + b
}
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte(`import "app/utils/math.ic" as math

let total = math.add(1, 2)
if total != 3 {
  panic("unexpected alias import result")
}
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	rt.SetProjectRoot(dir, "app")
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("expected root alias import to succeed, got: %v", err)
	}
}

func TestRuntimeRunFile_ProjectRootAliasRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "src", "main.ic")

	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte(`import "app/../outside.ic" as outside
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	rt.SetProjectRoot(dir, "app")
	if _, err := rt.RunFile(mainPath); err == nil {
		t.Fatal("expected root alias traversal import to fail")
	}
}

func TestRuntimeRunFile_RelativeImportStillWorksWithProjectRootAlias(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "src", "main.ic")
	modPath := filepath.Join(dir, "src", "math.ic")

	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.WriteFile(modPath, []byte(`export const answer = 42
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte(`import "./math.ic" as math

if math.answer != 42 {
  panic("unexpected relative import result")
}
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	rt.SetProjectRoot(dir, "app")
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("expected relative import to keep working, got: %v", err)
	}
}

func TestRuntimeRunFile_RootAliasRequiresConfiguration(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "src", "main.ic")

	if err := os.MkdirAll(filepath.Dir(mainPath), 0o755); err != nil {
		t.Fatalf("mkdir main dir: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte(`import "app/utils/math.ic" as math
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(mainPath); err == nil {
		t.Fatal("expected alias-like import without configuration to fail")
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

if keys != "CopycopyopenReaderopenWriterprintprintlnreadAll" {
  panic("unexpected std.io iteration order")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.io iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_CapturesImportedStdModuleInClosure(t *testing.T) {
	src := `
import std.object as object

const api = fn() {
  fn readName(value) {
    return object.get(value, "name", "missing")
  }

  return {
    readName: readName
  }
}()

if api.readName({name: "icoo"}) != "icoo" {
  panic("expected imported std module capture to work")
}
if api.readName({}) != "missing" {
  panic("expected imported std module fallback in closure")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected imported std module capture to succeed, got: %v", err)
	}
}

func TestRuntimeRunFile_CapturesImportedFileModuleInClosure(t *testing.T) {
	dir := t.TempDir()
	helperPath := filepath.Join(dir, "helper.ic")
	mainPath := filepath.Join(dir, "main.ic")

	if err := os.WriteFile(helperPath, []byte(`export fn tag(value) {
  return "[" + value + "]"
}
`), 0o644); err != nil {
		t.Fatalf("write helper module: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(`import "./helper.ic" as helper

const api = fn() {
  fn render(value) {
    return helper.tag(value)
  }

  return {
    render: render
  }
}()

if api.render("icoo") != "[icoo]" {
  panic("expected imported file module capture to work")
}
`), 0o644); err != nil {
		t.Fatalf("write main module: %v", err)
	}

	rt := NewRuntime()
	if _, err := rt.RunFile(mainPath); err != nil {
		t.Fatalf("expected imported file module capture to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdIOCopy(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.txt")
	dstPath := filepath.Join(dir, "dest.txt")

	if err := os.WriteFile(srcPath, []byte("hello copy"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	src := `
import std.io as io

let reader = io.openReader("` + srcPath + `")
let writer = io.openWriter("` + dstPath + `")
let copied = io.copy(writer, reader)
reader.close()
writer.close()

if copied != 10 {
  panic("unexpected copied byte count")
}

let resultReader = io.openReader("` + dstPath + `")
let text = io.readAll(resultReader)
resultReader.close()

if text != "hello copy" {
  panic("unexpected copied contents")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.io copy to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdObjectModule(t *testing.T) {
	src := `
import std.object as object

let source = {
  name: "icoo",
  nested: {
    ok: true
  }
}

if object.get(source, "name") != "icoo" {
  panic("expected object.get to return existing field")
}
if object.get(source, "missing") != null {
  panic("expected object.get to return null for missing field")
}
if object.get(source, "missing", "fallback") != "fallback" {
  panic("expected object.get fallback")
}
if object.has(source, "nested") != true {
  panic("expected object.has to find field")
}
if object.has(source, "missing") != false {
  panic("expected object.has to miss field")
}

let merged = object.merge(source, {
  port: 8080,
  name: "proxy"
})
if merged.name != "proxy" {
  panic("expected object.merge to override field")
}
if merged.port != 8080 {
  panic("expected object.merge to add field")
}
if source.name != "icoo" {
  panic("expected object.merge to keep source immutable")
}

let keys = object.keys(merged)
if len(keys) != 3 {
  panic("expected object.keys size")
}
if keys[0] != "name" || keys[1] != "nested" || keys[2] != "port" {
  panic("expected object.keys to be sorted")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.object import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ObjectLiteralSupportsStringKeys(t *testing.T) {
	src := `
let headers = {
  "X-Trace-Id": "trace-1",
  "Content-Type": "application/json"
}

if headers["X-Trace-Id"] != "trace-1" {
  panic("expected string-key object index")
}
if headers["Content-Type"] != "application/json" {
  panic("expected second string-key field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected string-key object literal to succeed, got: %v", err)
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

func TestRuntimeRunSource_ImportsStdDBModule(t *testing.T) {
	src := `
import std.db as db

let conn = db.sqlite(":memory:")
if !conn.ping() {
  panic("expected db ping success")
}
conn.close()
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.db import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdDBModule(t *testing.T) {
	src := `
import std.db as db

let keys = ""
let count = 0
for key, value in db {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.db export kind")
  }
}

if keys != "mysqlopenpgsqlpostgressqlite" {
  panic("unexpected std.db iteration order")
}
if count != 5 {
  panic("unexpected std.db export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.db iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdDBSQLite(t *testing.T) {
	src := `
import std.db as db

let conn = db.sqlite(":memory:")
let createResult = conn.exec("create table users(id integer primary key, name text, age integer, note text)")
if createResult.rowsAffected != 0 {
  panic("unexpected create rowsAffected")
}

let insertOne = conn.exec("insert into users(name, age, note) values (?, ?, ?)", ["alice", 18, null])
let insertTwo = conn.exec("insert into users(name, age, note) values (?, ?, ?)", ["bob", 20, "hi"])
if insertOne.rowsAffected != 1 || insertTwo.rowsAffected != 1 {
  panic("unexpected insert rowsAffected")
}
if typeOf(insertOne.lastInsertId) != "int" {
  panic("expected sqlite lastInsertId")
}

let rows = conn.query("select id, name, age, note from users order by id")
if typeOf(rows) != "array" {
  panic("query should return array")
}
if rows[0].name != "alice" || rows[0].age != 18 {
  panic("unexpected first row")
}
if rows[0].note != null {
  panic("unexpected null field")
}
if rows[1].name != "bob" || rows[1].note != "hi" {
  panic("unexpected second row")
}

let one = conn.queryOne("select name, age from users where name = ?", ["bob"])
if one.name != "bob" || one.age != 20 {
  panic("unexpected queryOne row")
}
let missing = conn.queryOne("select name from users where name = ?", ["missing"])
if missing != null {
  panic("missing queryOne should return null")
}

conn.close()
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.db sqlite to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdJSONModule(t *testing.T) {
	src := `
import std.json as json

let text = json.encode({name: "icoo", nums: [1, 2], ok: true, miss: null})
let value = json.decode(text)

if typeOf(text) != "string" {
  panic("json.encode should return string")
}
if value.name != "icoo" {
  panic("unexpected decoded object field")
}
if value.nums[0] != 1 {
  panic("unexpected decoded array item")
}
if value.ok != true {
  panic("unexpected decoded bool field")
}
if value.miss != null {
  panic("unexpected decoded null field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.json import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdJSONModule(t *testing.T) {
	src := `
import std.json as json

let keys = ""
let count = 0
for key, value in json {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.json export kind")
  }
}

if keys != "decodeencodefromFilesaveToFile" {
  panic("unexpected std.json iteration order")
}
if count != 4 {
  panic("unexpected std.json export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.json iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdJSONRejectsUnsupportedEncodeValue(t *testing.T) {
	src := `
import std.json as json

json.encode(fn() {})
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.json.encode to reject unsupported value")
	}
}

func TestRuntimeRunSource_StdJSONRejectsNonStringDecodeArg(t *testing.T) {
	src := `
import std.json as json

json.decode(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.json.decode to reject non-string argument")
	}
}

func TestRuntimeRunSource_ImportsStdFSModule(t *testing.T) {
	dir := t.TempDir()
	src := `
import std.fs as fs

let root = "` + dir + `"
let nested = fs.join(root, "nested")
let filePath = fs.join(nested, "note.txt")
let copyPath = fs.join(root, "copy.txt")
let renamedPath = fs.join(root, "renamed.txt")
let emptyDir = fs.join(root, "empty")

if fs.exists(filePath) {
  panic("file should not exist before write")
}
fs.mkdir(nested)
fs.mkdir(emptyDir)
fs.writeFile(filePath, "hello")
if !fs.exists(filePath) {
  panic("file should exist after write")
}
if fs.readFile(filePath) != "hello" {
  panic("unexpected file contents")
}
if fs.base(filePath) != "note.txt" {
  panic("unexpected base")
}
if fs.base(fs.dir(filePath)) != "nested" {
  panic("unexpected dir")
}

let entries = fs.readDir(root)
if typeOf(entries) != "array" {
  panic("readDir should return array")
}
if entries[0].name != "empty" {
  panic("unexpected first readDir entry")
}
if !entries[0].isDir {
  panic("expected empty entry to be directory")
}
if entries[1].name != "nested" {
  panic("unexpected second readDir entry")
}

let info = fs.stat(filePath)
if info.name != "note.txt" {
  panic("unexpected stat name")
}
if info.size != 5 {
  panic("unexpected stat size")
}
if !info.isFile {
  panic("expected file stat")
}
if info.isDir {
  panic("file should not be dir")
}
if typeOf(info.mode) != "string" {
  panic("stat mode should be string")
}
if typeOf(info.modTime) != "int" {
  panic("stat modTime should be int")
}

fs.copyFile(filePath, copyPath)
if fs.readFile(copyPath) != "hello" {
  panic("unexpected copied file contents")
}
fs.rename(copyPath, renamedPath)
if fs.exists(copyPath) {
  panic("copy path should not exist after rename")
}
if fs.readFile(renamedPath) != "hello" {
  panic("unexpected renamed file contents")
}
fs.remove(renamedPath)
if fs.exists(renamedPath) {
  panic("renamed file should be removed")
}
fs.remove(emptyDir)
if fs.exists(emptyDir) {
  panic("empty dir should be removed")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.fs import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdFSModule(t *testing.T) {
	src := `
import std.fs as fs

let keys = ""
let count = 0
for key, value in fs {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.fs export kind")
  }
}

if keys != "basecopyFiledirexistsjoinmkdirreadDirreadFileremoverenamestatwriteFile" {
  panic("unexpected std.fs iteration order")
}
if count != 12 {
  panic("unexpected std.fs export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.fs iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdFSRejectsNonStringArgs(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "exists", src: `
import std.fs as fs
fs.exists(1)
`},
		{name: "mkdir", src: `
import std.fs as fs
fs.mkdir(1)
`},
		{name: "rename", src: `
import std.fs as fs
fs.rename("a", 1)
`},
		{name: "copyFile", src: `
import std.fs as fs
fs.copyFile(1, "b")
`},
		{name: "stat", src: `
import std.fs as fs
fs.stat(1)
`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt := NewRuntime()
			if _, err := rt.RunSource(tc.src); err == nil {
				t.Fatalf("expected std.fs.%s to reject non-string argument", tc.name)
			}
		})
	}
}

func TestRuntimeRunSource_ImportsStdExecModule(t *testing.T) {
	src := `
import std.exec as exec

let result = exec.run("go", ["env", "GOOS"])
if !result.ok {
  panic("expected exec.run success")
}
if result.stdout == "" {
  panic("expected exec output")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.exec import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdExecModule(t *testing.T) {
	src := `
import std.exec as exec

let keys = ""
let count = 0
for key, value in exec {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.exec export kind")
  }
}

if keys != "run" {
  panic("unexpected std.exec iteration order")
}
if count != 1 {
  panic("unexpected std.exec export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.exec iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExecRejectsNonStringArrayArgs(t *testing.T) {
	src := `
import std.exec as exec

exec.run("go", [1])
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.exec.run to reject non-string array args")
	}
}

func TestRuntimeRunSource_ImportsStdOSModule(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	envKey := "ICOO_STD_OS_TEST"
	src := `
import std.fs as fs
import std.os as os

os.setEnv("` + envKey + `", "ok")
if os.getEnv("` + envKey + `") != "ok" {
  panic("expected env value")
}
if os.getEnv("ICOO_STD_OS_MISSING") != null {
  panic("missing env should be null")
}
if typeOf(os.args()) != "array" {
  panic("args should be array")
}
if typeOf(os.cwd()) != "string" {
  panic("cwd should be string")
}
if typeOf(os.tempDir()) != "string" {
  panic("tempDir should be string")
}
os.mkdirAll("` + nested + `")
if !fs.exists("` + nested + `") {
  panic("mkdirAll should create directories")
}
os.removeAll("` + dir + `")
if fs.exists("` + dir + `") {
  panic("removeAll should remove directory tree")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.os import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdOSModule(t *testing.T) {
	src := `
import std.os as os

let keys = ""
let count = 0
for key, value in os {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.os export kind")
  }
}

if keys != "argscwdgetEnvmkdirAllremoveremoveAllsetEnvtempDir" {
  panic("unexpected std.os iteration order")
}
if count != 8 {
  panic("unexpected std.os export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.os iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdHostModule(t *testing.T) {
	src := `
import std.host as host

if host.goos() != "` + runtime.GOOS + `" {
  panic("unexpected host goos")
}
if host.arch() != "` + runtime.GOARCH + `" {
  panic("unexpected host arch")
}
if host.hostname() == "" {
  panic("expected hostname")
}
if host.numCPU() < 1 {
  panic("expected cpu count")
}
if host.pid() < 1 {
  panic("expected pid")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.host import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdHostModule(t *testing.T) {
	src := `
import std.host as host

let keys = ""
let count = 0
for key, value in host {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.host export kind")
  }
}

if keys != "archgooshostnamenumCPUpid" {
  panic("unexpected std.host iteration order")
}
if count != 5 {
  panic("unexpected std.host export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.host iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdJSONFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	src := `
import std.fs as fs
import std.json as json

json.saveToFile("` + path + `", {name: "icoo", nums: [1, 2], ok: true})
if !fs.exists("` + path + `") {
  panic("expected json file to exist")
}
let value = json.fromFile("` + path + `")
if value.name != "icoo" {
  panic("unexpected json file object field")
}
if value.nums[1] != 2 {
  panic("unexpected json file array item")
}
if value.ok != true {
  panic("unexpected json file bool field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.json file round trip to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdYAMLModule(t *testing.T) {
	src := `
import std.yaml as yaml

let text = yaml.encode({name: "icoo", nums: [1, 2], ok: true})
let value = yaml.decode(text)

if typeOf(text) != "string" {
  panic("yaml.encode should return string")
}
if value.name != "icoo" {
  panic("unexpected decoded yaml object field")
}
if value.nums[0] != 1 {
  panic("unexpected decoded yaml array item")
}
if value.ok != true {
  panic("unexpected decoded yaml bool field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.yaml import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdYAMLModule(t *testing.T) {
	src := `
import std.yaml as yaml

let keys = ""
let count = 0
for key, value in yaml {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.yaml export kind")
  }
}

if keys != "decodeencodefromFilesaveToFile" {
  panic("unexpected std.yaml iteration order")
}
if count != 4 {
  panic("unexpected std.yaml export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.yaml iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdYAMLFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.yaml")
	src := `
import std.fs as fs
import std.yaml as yaml

yaml.saveToFile("` + path + `", {name: "icoo", nums: [1, 2], ok: true})
if !fs.exists("` + path + `") {
  panic("expected yaml file to exist")
}
let value = yaml.fromFile("` + path + `")
if value.name != "icoo" {
  panic("unexpected yaml file object field")
}
if value.nums[1] != 2 {
  panic("unexpected yaml file array item")
}
if value.ok != true {
  panic("unexpected yaml file bool field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.yaml file round trip to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdTOMLModule(t *testing.T) {
	src := `
import std.toml as toml

let text = toml.encode({name: "icoo", port: 8080, ok: true})
let value = toml.decode(text)

if typeOf(text) != "string" {
  panic("toml.encode should return string")
}
if value.name != "icoo" {
  panic("unexpected decoded toml object field")
}
if value.port != 8080 {
  panic("unexpected decoded toml number field")
}
if value.ok != true {
  panic("unexpected decoded toml bool field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.toml import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdTOMLModule(t *testing.T) {
	src := `
import std.toml as toml

let keys = ""
let count = 0
for key, value in toml {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.toml export kind")
  }
}

if keys != "decodeencodefromFilesaveToFile" {
  panic("unexpected std.toml iteration order")
}
if count != 4 {
  panic("unexpected std.toml export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.toml iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdTOMLFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.toml")
	src := `
import std.fs as fs
import std.toml as toml

toml.saveToFile("` + path + `", {name: "icoo", port: 8080, ok: true})
if !fs.exists("` + path + `") {
  panic("expected toml file to exist")
}
let value = toml.fromFile("` + path + `")
if value.name != "icoo" {
  panic("unexpected toml file object field")
}
if value.port != 8080 {
  panic("unexpected toml file number field")
}
if value.ok != true {
  panic("unexpected toml file bool field")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.toml file round trip to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdXMLModule(t *testing.T) {
	src := `
import std.xml as xml

let node = {
  name: "root",
  attrs: {id: "1"},
  children: [
    {name: "item", text: "hello"}
  ]
}
let text = xml.encode(node)
let value = xml.decode(text)

if typeOf(text) != "string" {
  panic("xml.encode should return string")
}
if value.name != "root" {
  panic("unexpected decoded xml root name")
}
if value.attrs.id != "1" {
  panic("unexpected decoded xml attr")
}
if value.children[0].name != "item" {
  panic("unexpected decoded xml child name")
}
if value.children[0].text != "hello" {
  panic("unexpected decoded xml child text")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.xml import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdXMLModule(t *testing.T) {
	src := `
import std.xml as xml

let keys = ""
let count = 0
for key, value in xml {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.xml export kind")
  }
}

if keys != "decodeencodefromFilesaveToFile" {
  panic("unexpected std.xml iteration order")
}
if count != 4 {
  panic("unexpected std.xml export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.xml iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdXMLFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.xml")
	src := `
import std.fs as fs
import std.xml as xml

xml.saveToFile("` + path + `", {
  name: "root",
  attrs: {id: "7"},
  children: [
    {name: "item", text: "hello"}
  ]
})
if !fs.exists("` + path + `") {
  panic("expected xml file to exist")
}
let value = xml.fromFile("` + path + `")
if value.name != "root" {
  panic("unexpected xml file root name")
}
if value.attrs.id != "7" {
  panic("unexpected xml file attr")
}
if value.children[0].text != "hello" {
  panic("unexpected xml file child text")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.xml file round trip to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdHTTPModule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	src := `
import std.http.client as http

let resp = http.get("` + server.URL + `")
if !resp.ok {
  panic("expected http.get success")
}
if resp.status != 200 {
  panic("expected 200 status")
}
if resp.body != "hello" {
  panic("unexpected response body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPRequestSupportsMethodHeadersAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Accept") != "text/plain" {
			http.Error(w, "bad header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	src := `
import std.http.client as http

let resp = http.request({
  url: "` + server.URL + `",
  method: "POST",
  headers: {Accept: "text/plain"},
  body: "payload",
  timeoutMs: 5000
})
if !resp.ok {
  panic("expected http.request success")
}
if resp.body != "payload" {
  panic("unexpected echoed request body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client request to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPDownload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "download.txt")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("download-body"))
	}))
	defer server.Close()

	src := `
import std.fs as fs
import std.http.client as http

let resp = http.download("` + server.URL + `", "` + path + `")
if !resp.ok {
  panic("expected http.download success")
}
if !fs.exists("` + path + `") {
  panic("expected downloaded file")
}
if fs.readFile("` + path + `") != "download-body" {
  panic("unexpected downloaded contents")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client download to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdHTTPClientModule(t *testing.T) {
	src := `
import std.http.client as http

let keys = ""
let count = 0
for key, value in http {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.http.client export kind")
  }
}

if keys != "deletedownloadgetgetJSONpostputrequestrequestJSON" {
  panic("unexpected std.http.client iteration order")
}
if count != 8 {
  panic("unexpected std.http.client export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdHTTPServerModule(t *testing.T) {
	src := `
import std.http.server as http

let keys = ""
let count = 0
for key, value in http {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.http.server export kind")
  }
}

if keys != "forwardlisten" {
  panic("unexpected std.http.server iteration order")
}
if count != 2 {
  panic("unexpected std.http.server export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerRequestHeaderHelpers(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let upstream = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return {
      json: {
        auth: req.header("Authorization"),
        hasTrace: req.hasHeader("X-Trace-Id"),
        missing: req.header("X-Missing")
      }
    }
  }
})

let resp = client.requestJSON({
  url: upstream.url + "/headers",
  method: "POST",
  headers: {
    Authorization: "Bearer demo",
    "X-Trace-Id": "trace-1"
  },
  json: {
    ok: true
  }
})

upstream.close()

if resp.json.auth != "Bearer demo" {
  panic("expected req.header to read Authorization")
}
if resp.json.hasTrace != true {
  panic("expected req.hasHeader to detect header")
}
if resp.json.missing != null {
  panic("expected missing req.header to return null")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server request header helpers to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerRequestJSON(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let upstream = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return {
      json: {
        name: req.json.name,
        count: req.json.count
      }
    }
  }
})

let resp = client.requestJSON({
  url: upstream.url + "/json",
  method: "POST",
  json: {
    name: "icoo",
    count: 2
  }
})

upstream.close()

if resp.json.name != "icoo" {
  panic("expected req.json.name")
}
if resp.json.count != 2 {
  panic("expected req.json.count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server request json to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerResponseHandleWrite(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let upstream = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req, res) {
    res.setHeader("X-Reply", req.header("X-Trace-Id"))
    res.status(202)
    res.write(req.method + ":" + req.path)
  }
})

let resp = client.request({
  url: upstream.url + "/direct",
  method: "POST",
  headers: {
    "X-Trace-Id": "trace-7"
  },
  body: "ignored"
})

upstream.close()

if resp.status != 202 {
  panic("expected direct response status")
}
if resp.body != "POST:/direct" {
  panic("expected direct response body")
}
if resp.header("X-Reply") != "trace-7" {
  panic("expected direct response header")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server response handle write to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerResponseHandleJSON(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let upstream = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req, res) {
    res.status(201)
    res.json({
      name: req.json.name,
      ok: true
    })
  }
})

let resp = client.requestJSON({
  url: upstream.url + "/json",
  method: "POST",
  json: {
    name: "icoo"
  }
})

upstream.close()

if resp.status != 201 {
  panic("expected direct json status")
}
if resp.json.name != "icoo" {
  panic("expected direct json body name")
}
if resp.json.ok != true {
  panic("expected direct json body flag")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server response handle json to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerResponseHandleStatusWithReturnedValue(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let upstream = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req, res) {
    res.setHeader("X-Mode", "hybrid")
    res.status(203)
    return {
      json: {
        path: req.path
      }
    }
  }
})

let resp = client.getJSON(upstream.url + "/hybrid")
upstream.close()

if resp.status != 203 {
  panic("expected hybrid response status")
}
if resp.header("X-Mode") != "hybrid" {
  panic("expected hybrid response header")
}
if resp.json.path != "/hybrid" {
  panic("expected hybrid response body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server response handle hybrid path to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerResponseHandleProxyPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/proxy" || r.URL.Query().Get("name") != "icoo" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "text/plain" {
			http.Error(w, "bad header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "payload" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Upstream", "seen")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	src := `
import std.http.client as client
import std.http.server as server

let gateway = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req, res) {
    res.proxy(req, {
      url: "` + upstream.URL + `" + req.path + "?name=" + req.query.name,
      timeoutMs: 5000
    })
  }
})

let resp = client.request({
  url: gateway.url + "/proxy?name=icoo",
  method: "POST",
  headers: {
    Accept: "text/plain"
  },
  body: "payload"
})
gateway.close()

if resp.status != 202 {
  panic("expected proxied response status")
}
if resp.body != "proxied" {
  panic("expected proxied response body")
}
if resp.header("X-Upstream") != "seen" {
  panic("expected proxied response header")
}
if resp.header("Connection") != null {
  panic("expected hop-by-hop response header filtered")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server response handle proxy pass-through to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPServerResponseHandleProxyOverridesRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/override" || r.URL.Query().Get("ok") != "1" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer proxy" {
			http.Error(w, "bad auth header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "changed" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	src := `
import std.http.client as client
import std.http.server as server

let gateway = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req, res) {
    res.setHeader("X-Gateway", "icoo")
    res.proxy(req, {
      url: "` + upstream.URL + `/override?ok=1",
      method: "PUT",
      headers: {
        Authorization: "Bearer proxy"
      },
      body: "changed",
      timeoutMs: 5000
    })
  }
})

let resp = client.requestJSON({
  url: gateway.url + "/ignored",
  method: "POST",
  body: "original"
})
gateway.close()

if resp.status != 201 {
  panic("expected override proxied response status")
}
if resp.header("X-Gateway") != "icoo" {
  panic("expected local response header to survive proxy")
}
if resp.json.ok != true {
  panic("expected proxied json body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server response handle proxy overrides to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPShortcutMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write([]byte("post:" + string(body)))
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write([]byte("put:" + string(body)))
		case http.MethodDelete:
			_, _ = w.Write([]byte("delete"))
		default:
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	src := `
import std.http.client as http

let postResp = http.post("` + server.URL + `", "A")
let putResp = http.put("` + server.URL + `", "B")
let deleteResp = http.delete("` + server.URL + `")

if postResp.body != "post:A" {
  panic("unexpected post response")
}
if putResp.body != "put:B" {
  panic("unexpected put response")
}
if deleteResp.body != "delete" {
  panic("unexpected delete response")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client shortcut methods to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPJSONHelpers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"ok":true,"value":7}`))
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("X-Seen-Accept", r.Header.Get("Accept"))
			w.Header().Set("X-Seen-Content-Type", r.Header.Get("Content-Type"))
			_, _ = w.Write(body)
		default:
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	src := `
import std.http.client as http

let getResp = http.getJSON("` + server.URL + `")
if getResp.json.ok != true {
  panic("unexpected getJSON bool")
}
if getResp.json.value != 7 {
  panic("unexpected getJSON number")
}

let postResp = http.requestJSON({
  url: "` + server.URL + `",
  method: "POST",
  json: {name: "icoo", count: 2}
})
if postResp.json.name != "icoo" {
  panic("unexpected requestJSON object field")
}
if postResp.json.count != 2 {
  panic("unexpected requestJSON number field")
}
if postResp.headers["X-Seen-Accept"] != "application/json" {
  panic("expected requestJSON accept header")
}
if postResp.headers["X-Seen-Content-Type"] != "application/json" {
  panic("expected requestJSON content type header")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.client JSON helpers to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPListen(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let serverHandle = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return {
      status: 201,
      body: req.method + ":" + req.path + ":" + req.query.name
    }
  }
})

let resp = client.get(serverHandle.url + "/hello?name=icoo")
serverHandle.close()

if resp.status != 201 {
  panic("unexpected http.listen status")
}
if resp.body != "GET:/hello:icoo" {
  panic("unexpected http.listen body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server listen to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPListenJSONResponse(t *testing.T) {
	src := `
import std.http.client as client
import std.http.server as server

let serverHandle = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return {
      status: 200,
      json: {path: req.path, ok: true}
    }
  }
})

let resp = client.getJSON(serverHandle.url + "/json")
serverHandle.close()

if resp.json.path != "/json" {
  panic("unexpected http.listen json path")
}
if resp.json.ok != true {
  panic("unexpected http.listen json bool")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server listen JSON response to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPForwardPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/forward" || r.URL.Query().Get("name") != "icoo" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "text/plain" {
			http.Error(w, "bad header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "payload" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Upstream", "seen")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("forwarded"))
	}))
	defer upstream.Close()

	src := `
import std.http.client as client
import std.http.server as server

let serverHandle = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return server.forward(req, {
      url: "` + upstream.URL + `" + req.path + "?name=" + req.query.name,
      timeoutMs: 5000
    })
  }
})

let resp = client.request({
  url: serverHandle.url + "/forward?name=icoo",
  method: "POST",
  headers: {Accept: "text/plain"},
  body: "payload"
})
serverHandle.close()

if resp.status != 202 {
  panic("unexpected forward status")
}
if resp.body != "forwarded" {
  panic("unexpected forward body")
}
if resp.headers["X-Upstream"] != "seen" {
  panic("unexpected forward response header")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server forward pass-through to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPForwardOverridesRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/override" || r.URL.Query().Get("ok") != "1" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "application/json" {
			http.Error(w, "bad header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "changed" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("overridden"))
	}))
	defer upstream.Close()

	src := `
import std.http.client as client
import std.http.server as server

let serverHandle = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return server.forward(req, {
      url: "` + upstream.URL + `/override?ok=1",
      method: "PUT",
      headers: {Accept: "application/json"},
      body: "changed",
      timeoutMs: 5000
    })
  }
})

let resp = client.request({
  url: serverHandle.url + "/ignored",
  method: "POST",
  headers: {Accept: "text/plain"},
  body: "original"
})
serverHandle.close()

if resp.status != 200 {
  panic("unexpected override forward status")
}
if resp.body != "overridden" {
  panic("unexpected override forward body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server forward overrides to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPForwardFiltersHopByHopHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Connection") != "" {
			http.Error(w, "connection header forwarded", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Upgrade") != "" {
			http.Error(w, "upgrade header forwarded", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "text/plain" {
			http.Error(w, "regular header missing", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("filtered"))
	}))
	defer upstream.Close()

	src := `
import std.http.client as client
import std.http.server as server

let serverHandle = server.listen({
  addr: "127.0.0.1:0",
  handler: fn(req) {
    return server.forward(req, {
      url: "` + upstream.URL + `/headers",
      timeoutMs: 5000
    })
  }
})

let resp = client.request({
  url: serverHandle.url + "/headers",
  headers: {Accept: "text/plain", Connection: "close", Upgrade: "websocket"}
})
serverHandle.close()

if resp.status != 200 {
  panic("unexpected filtered forward status")
}
if resp.body != "filtered" {
  panic("unexpected filtered forward body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.http.server forward header filtering to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_RejectsOldStdNetHTTPPaths(t *testing.T) {
	src := `
import std.net.http.client as http

http.get("http://127.0.0.1")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected old std.net.http.client import to fail")
	}
}

func TestRuntimeRunSource_IteratesStdNetWebSocketModules(t *testing.T) {
	src := `
import std.net.websocket.client as client
import std.net.websocket.server as server

let clientKeys = ""
let clientCount = 0
for key, value in client {
  clientKeys = clientKeys + key
  clientCount = clientCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected websocket client export kind")
  }
}
let serverKeys = ""
let serverCount = 0
for key, value in server {
  serverKeys = serverKeys + key
  serverCount = serverCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected websocket server export kind")
  }
}
if clientKeys != "connect" || clientCount != 1 {
  panic("unexpected websocket client exports")
}
if serverKeys != "listen" || serverCount != 1 {
  panic("unexpected websocket server exports")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected websocket module iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdNetWebSocketEcho(t *testing.T) {
	src := `
import std.net.websocket.client as client
import std.net.websocket.server as server

let srv = server.listen({
  addr: "127.0.0.1:0",
  path: "/ws",
  handler: fn(conn, req) {
    let msg = conn.read()
    conn.write("echo:" + msg)
    conn.close()
  }
})
let conn = client.connect({url: srv.url, timeoutMs: 5000})
conn.write("hello")
let reply = conn.read()
conn.close()
srv.close()
if reply != "echo:hello" {
  panic("unexpected websocket echo")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected websocket echo to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdNetSSEModules(t *testing.T) {
	src := `
import std.net.sse.client as client
import std.net.sse.server as server

let clientKeys = ""
let clientCount = 0
for key, value in client {
  clientKeys = clientKeys + key
  clientCount = clientCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected sse client export kind")
  }
}
let serverKeys = ""
let serverCount = 0
for key, value in server {
  serverKeys = serverKeys + key
  serverCount = serverCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected sse server export kind")
  }
}
if clientKeys != "connect" || clientCount != 1 {
  panic("unexpected sse client exports")
}
if serverKeys != "listen" || serverCount != 1 {
  panic("unexpected sse server exports")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected sse module iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdNetSSEEvent(t *testing.T) {
	src := `
import std.net.sse.client as client
import std.net.sse.server as server

let srv = server.listen({
  addr: "127.0.0.1:0",
  path: "/events",
  handler: fn(stream, req) {
    stream.send({event: "message", data: "hello", id: "1"})
    stream.close()
  }
})
let stream = client.connect({url: srv.url, timeoutMs: 5000})
let event = stream.read()
stream.close()
srv.close()
if event.event != "message" {
  panic("unexpected sse event name")
}
if event.data != "hello" {
  panic("unexpected sse data")
}
if event.id != "1" {
  panic("unexpected sse id")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected sse event to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdNetSocketModules(t *testing.T) {
	src := `
import std.net.socket.client as client
import std.net.socket.server as server

let clientKeys = ""
let clientCount = 0
for key, value in client {
  clientKeys = clientKeys + key
  clientCount = clientCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected socket client export kind")
  }
}
let serverKeys = ""
let serverCount = 0
for key, value in server {
  serverKeys = serverKeys + key
  serverCount = serverCount + 1
  if typeOf(value) != "native_function" {
    panic("unexpected socket server export kind")
  }
}
if clientKeys != "connectTCPdialUDP" || clientCount != 2 {
  panic("unexpected socket client exports")
}
if serverKeys != "listenTCPlistenUDP" || serverCount != 2 {
  panic("unexpected socket server exports")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected socket module iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdNetSocketTCPEcho(t *testing.T) {
	src := `
import std.net.socket.client as client
import std.net.socket.server as server

let srv = server.listenTCP({
  addr: "127.0.0.1:0",
  handler: fn(conn) {
    let msg = conn.read(1024)
    conn.write("echo:" + msg)
    conn.close()
  }
})
let conn = client.connectTCP({addr: srv.addr, timeoutMs: 5000})
conn.write("hello")
let reply = conn.read(1024)
conn.close()
srv.close()
if reply != "echo:hello" {
  panic("unexpected tcp echo")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected tcp echo to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdNetSocketUDPEcho(t *testing.T) {
	src := `
import std.net.socket.client as client
import std.net.socket.server as server

let srv = server.listenUDP({
  addr: "127.0.0.1:0",
  handler: fn(packet) {
    packet.reply("echo:" + packet.data)
  }
})
let conn = client.dialUDP({addr: srv.addr})
conn.write("hello")
let reply = conn.read(1024)
conn.close()
srv.close()
if reply != "echo:hello" {
  panic("unexpected udp echo")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected udp echo to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdExpressModule(t *testing.T) {
	src := `
import std.express as express

let keys = ""
let count = 0
for key, value in express {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.express export kind")
  }
}

if keys != "createjsonnewnextredirecttext" {
  panic("unexpected std.express iteration order")
}
if count != 6 {
  panic("unexpected std.express export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.express iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressRoutes(t *testing.T) {
	src := `
import std.express as express
import std.http.client as client

let app = express.create()
app.get("/hello", fn(req) {
  return express.text(201, req.method + ":" + req.path + ":" + req.query.name)
})
app.post("/json", fn(req) {
  return express.json({name: req.json.name, count: req.json.count})
})

let server = app.listen({addr: "127.0.0.1:0"})
let getResp = client.get(server.url + "/hello?name=icoo")
let postResp = client.requestJSON({
  url: server.url + "/json",
  method: "POST",
  json: {name: "icoo", count: 2}
})
server.close()

if getResp.status != 201 {
  panic("unexpected express get status")
}
if getResp.body != "GET:/hello:icoo" {
  panic("unexpected express get body")
}
if postResp.json.name != "icoo" {
  panic("unexpected express json name")
}
if postResp.json.count != 2 {
  panic("unexpected express json count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.express routes to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressMiddleware(t *testing.T) {
	src := `
import std.express as express
import std.http.client as client

let app = express.new()
app.use(fn(req, next) {
  req.mark = "global"
  return next()
})
app.use("/api", fn(req) {
  req.scope = "api"
  return express.next()
})
app.get("/api/hello", fn(req) {
  return express.text(req.mark + ":" + req.scope + ":" + req.path)
})
app.get("/stop", fn(req) {
  return express.text(202, "stopped")
})
app.use("/stop", fn(req) {
  panic("middleware after response should not run")
})

let server = app.listen({addr: "127.0.0.1:0"})
let resp = client.get(server.url + "/api/hello")
let stopResp = client.get(server.url + "/stop")
server.close()

if resp.status != 200 {
  panic("unexpected express middleware status")
}
if resp.body != "global:api:/api/hello" {
  panic("unexpected express middleware body")
}
if stopResp.status != 202 {
  panic("unexpected express terminal middleware status")
}
if stopResp.body != "stopped" {
  panic("unexpected express terminal middleware body")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.express middleware to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressRequestHeaderHelpers(t *testing.T) {
	src := `
import std.express as express
import std.http.client as client

let app = express.create()
app.post("/inspect", fn(req) {
  return express.json({
    auth: req.header("Authorization"),
    hasTrace: req.hasHeader("X-Trace-Id"),
    missing: req.header("X-Missing")
  })
})

let server = app.listen({addr: "127.0.0.1:0"})
let resp = client.requestJSON({
  url: server.url + "/inspect",
  method: "POST",
  headers: {
    Authorization: "Bearer express",
    "X-Trace-Id": "trace-2"
  },
  json: {
    ok: true
  }
})

server.close()

if resp.json.auth != "Bearer express" {
  panic("expected express req.header to read Authorization")
}
if resp.json.hasTrace != true {
  panic("expected express req.hasHeader to detect header")
}
if resp.json.missing != null {
  panic("expected express missing req.header to return null")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected express request header helpers to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressResponseHandleWrite(t *testing.T) {
	src := `
import std.express as express
import std.http.client as client

let app = express.create()
app.get("/direct", fn(req, res) {
  res.setHeader("X-Mode", "direct")
  res.status(202)
  res.write(req.method + ":" + req.path)
})

let server = app.listen({addr: "127.0.0.1:0"})
let resp = client.get(server.url + "/direct")
server.close()

if resp.status != 202 {
  panic("unexpected express direct status")
}
if resp.body != "GET:/direct" {
  panic("unexpected express direct body")
}
if resp.header("X-Mode") != "direct" {
  panic("unexpected express direct header")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected express response handle write to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressResponseHandleProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/proxy" || r.URL.Query().Get("name") != "icoo" {
			http.Error(w, "bad url", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "text/plain" {
			http.Error(w, "bad header", http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "payload" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Upstream", "seen")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("express-proxied"))
	}))
	defer upstream.Close()

	src := `
import std.express as express
import std.http.client as client

let app = express.create()
app.post("/proxy", fn(req, res) {
  res.setHeader("X-Gateway", "icoo")
  res.proxy(req, {
    url: "` + upstream.URL + `" + req.path + "?name=" + req.query.name,
    timeoutMs: 5000
  })
})

let server = app.listen({addr: "127.0.0.1:0"})
let resp = client.request({
  url: server.url + "/proxy?name=icoo",
  method: "POST",
  headers: {
    Accept: "text/plain"
  },
  body: "payload"
})
server.close()

if resp.status != 202 {
  panic("unexpected express proxied status")
}
if resp.body != "express-proxied" {
  panic("unexpected express proxied body")
}
if resp.header("X-Upstream") != "seen" {
  panic("unexpected express proxied upstream header")
}
if resp.header("X-Gateway") != "icoo" {
  panic("unexpected express proxied local header")
}
if resp.header("Connection") != null {
  panic("expected express hop-by-hop header filtered")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected express response handle proxy to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdExpressRootRouteMatchesExactly(t *testing.T) {
	src := `
import std.express as express
import std.http.client as client

let app = express.create()
app.get("/", fn(req) {
  return express.text("root")
})
app.get("/admin/routes", fn(req) {
  return express.json({ok: true, route: req.path})
})

let server = app.listen({addr: "127.0.0.1:0"})
let rootResp = client.get(server.url + "/")
let adminResp = client.getJSON(server.url + "/admin/routes")
server.close()

if rootResp.body != "root" {
  panic("unexpected express root body")
}
if adminResp.json.route != "/admin/routes" {
  panic("root route should not shadow specific routes")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected express root route match to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_ImportsStdCryptoModule(t *testing.T) {
	src := `
import std.crypto as crypto

if crypto.sha256("abc") != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
  panic("unexpected sha256 result")
}
if crypto.sha512("abc") != "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f" {
  panic("unexpected sha512 result")
}
if crypto.hmacSHA256("key", "abc") != "9c196e32dc0175f86f4b1cb89289d6619de6bee699e4c378e68309ed97a1a6ab" {
  panic("unexpected hmacSHA256 result")
}
if crypto.base64Decode(crypto.base64Encode("hello")) != "hello" {
  panic("unexpected base64 round trip")
}
if crypto.hexDecode(crypto.hexEncode("hello")) != "hello" {
  panic("unexpected hex round trip")
}
if typeOf(crypto.randomBytes(16)) != "string" {
  panic("randomBytes should return string")
}
let encrypted = crypto.aesGCMEncrypt("1234567890123456", "secret")
if crypto.aesGCMDecrypt("1234567890123456", encrypted.nonce, encrypted.ciphertext) != "secret" {
  panic("unexpected aes-gcm round trip")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.crypto import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdCryptoModule(t *testing.T) {
	src := `
import std.crypto as crypto

let keys = ""
let count = 0
for key, value in crypto {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.crypto export kind")
  }
}

if keys != "aesGCMDecryptaesGCMEncryptbase64Decodebase64EncodehexDecodehexEncodehmacSHA256hmacSHA512randomBytessha256sha512" {
  panic("unexpected std.crypto iteration order")
}
if count != 11 {
  panic("unexpected std.crypto export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.crypto iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdCryptoRejectsInvalidArgs(t *testing.T) {
	src := `
import std.crypto as crypto

crypto.aesGCMEncrypt("bad", "secret")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.crypto.aesGCMEncrypt to reject invalid key")
	}
}

func TestRuntimeRunSource_ImportsStdUUIDModule(t *testing.T) {
	src := `
import std.uuid as uuid

let first = uuid.v4()
let second = uuid.v4()
if typeOf(first) != "string" {
  panic("uuid.v4 should return string")
}
if !uuid.isValid(first) {
  panic("uuid.v4 should return valid uuid")
}
if first == second {
  panic("uuid.v4 should generate unique values")
}
if uuid.isValid("not-a-uuid") {
  panic("invalid uuid should be rejected")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.uuid import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdUUIDModule(t *testing.T) {
	src := `
import std.uuid as uuid

let keys = ""
let count = 0
for key, value in uuid {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.uuid export kind")
  }
}

if keys != "isValidv4" {
  panic("unexpected std.uuid iteration order")
}
if count != 2 {
  panic("unexpected std.uuid export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.uuid iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdUUIDRejectsNonStringArgs(t *testing.T) {
	src := `
import std.uuid as uuid

uuid.isValid(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.uuid.isValid to reject non-string argument")
	}
}

func TestRuntimeRunSource_ImportsStdCompressModule(t *testing.T) {
	src := `
import std.compress as compress

let gzipText = compress.gzipCompress("hello compression")
if typeOf(gzipText) != "string" {
  panic("gzipCompress should return string")
}
if compress.gzipDecompress(gzipText) != "hello compression" {
  panic("unexpected gzip round trip")
}
let zlibText = compress.zlibCompress("hello compression")
if typeOf(zlibText) != "string" {
  panic("zlibCompress should return string")
}
if compress.zlibDecompress(zlibText) != "hello compression" {
  panic("unexpected zlib round trip")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.compress import to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdCompressModule(t *testing.T) {
	src := `
import std.compress as compress

let keys = ""
let count = 0
for key, value in compress {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.compress export kind")
  }
}

if keys != "gzipCompressgzipDecompresszlibCompresszlibDecompress" {
  panic("unexpected std.compress iteration order")
}
if count != 4 {
  panic("unexpected std.compress export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.compress iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdCompressRejectsInvalidInput(t *testing.T) {
	src := `
import std.compress as compress

compress.gzipDecompress("not-base64")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.compress.gzipDecompress to reject invalid input")
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
