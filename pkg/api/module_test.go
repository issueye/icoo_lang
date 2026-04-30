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

if keys != "decodeencode" {
  panic("unexpected std.json iteration order")
}
if count != 2 {
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
	filePath := filepath.Join(dir, "note.txt")
	src := `
import std.fs as fs

if fs.exists("` + filePath + `") {
  panic("file should not exist before write")
}
fs.writeFile("` + filePath + `", "hello")
if !fs.exists("` + filePath + `") {
  panic("file should exist after write")
}
if fs.readFile("` + filePath + `") != "hello" {
  panic("unexpected file contents")
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

if keys != "existsreadFilewriteFile" {
  panic("unexpected std.fs iteration order")
}
if count != 3 {
  panic("unexpected std.fs export count")
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.fs iteration to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdFSRejectsNonStringArgs(t *testing.T) {
	src := `
import std.fs as fs

fs.exists(1)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.fs.exists to reject non-string argument")
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
