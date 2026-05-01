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
import std.net.http.client as http

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
		t.Fatalf("expected std.net.http import to succeed, got: %v", err)
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
import std.net.http.client as http

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
		t.Fatalf("expected std.net.http request to succeed, got: %v", err)
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
import std.net.http.client as http

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
		t.Fatalf("expected std.net.http download to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdHTTPClientModule(t *testing.T) {
	src := `
import std.net.http.client as http

let keys = ""
let count = 0
for key, value in http {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.net.http.client export kind")
  }
}

if keys != "deletedownloadgetgetJSONpostputrequestrequestJSON" {
  panic("unexpected std.net.http.client iteration order")
}
if count != 8 {
  panic("unexpected std.net.http.client export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.net.http.client iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_IteratesStdHTTPServerModule(t *testing.T) {
	src := `
import std.net.http.server as http

let keys = ""
let count = 0
for key, value in http {
  keys = keys + key
  count = count + 1
  if typeOf(value) != "native_function" {
    panic("unexpected std.net.http.server export kind")
  }
}

if keys != "listen" {
  panic("unexpected std.net.http.server iteration order")
}
if count != 1 {
  panic("unexpected std.net.http.server export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.net.http.server iteration to succeed, got: %v", err)
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
import std.net.http.client as http

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
		t.Fatalf("expected std.net.http shortcut methods to succeed, got: %v", err)
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
import std.net.http.client as http

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
		t.Fatalf("expected std.net.http JSON helpers to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPListen(t *testing.T) {
	src := `
import std.net.http.client as client
import std.net.http.server as server

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
		t.Fatalf("expected std.net.http listen to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdHTTPListenJSONResponse(t *testing.T) {
	src := `
import std.net.http.client as client
import std.net.http.server as server

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
		t.Fatalf("expected std.net.http listen JSON response to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_RejectsOldStdHTTPPaths(t *testing.T) {
	src := `
import std.http.client as http

http.get("http://127.0.0.1")
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected old std.http.client import to fail")
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
import std.net.http.client as client

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
import std.net.http.client as client

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
