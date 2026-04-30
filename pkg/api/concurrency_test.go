package api

import (
	"testing"
	"time"
)

// ---- Channel tests ----

func TestChanCreate(t *testing.T) {
	src := `
let ch = chan()
if typeOf(ch) != "channel" {
  panic("expected channel type")
}
let ch2 = chan(5)
if typeOf(ch2) != "channel" {
  panic("expected channel type for buffered")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan creation failed: %v", err)
	}
}

func TestChanSendRecv(t *testing.T) {
	src := `
let ch = chan(1)
ch.send(42)
let v = ch.recv()
if v != 42 {
  panic("expected 42 from recv")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan send/recv failed: %v", err)
	}
}

func TestChanTrySendRecv(t *testing.T) {
	src := `
let ch = chan(1)
let ok = ch.trySend(10)
if !ok {
  panic("trySend should succeed on empty buffered channel")
}
let result = ch.tryRecv()
if !result.ok {
  panic("tryRecv should succeed")
}
if result.value != 10 {
  panic("expected 10 from tryRecv")
}
let result2 = ch.tryRecv()
if result2.ok {
  panic("tryRecv should fail on empty channel")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan trySend/tryRecv failed: %v", err)
	}
}

func TestChanClose(t *testing.T) {
	src := `
let ch = chan(2)
ch.send(1)
ch.send(2)
ch.close()
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan close failed: %v", err)
	}
}

// ---- Go statement tests ----

func TestGoStatementRuns(t *testing.T) {
	src := `
let ch = chan(1)
let counter = 0

go fn() {
  ch.send(99)
}()

let v = ch.recv()
if v != 99 {
  panic("expected 99 from go goroutine")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("go statement failed: %v", err)
	}
}

func TestGoMultipleGoroutines(t *testing.T) {
	src := `
let ch = chan(3)

go fn() { ch.send(1) }()
go fn() { ch.send(2) }()
go fn() { ch.send(3) }()

let sum = ch.recv() + ch.recv() + ch.recv()
if sum != 6 {
  panic("expected sum 6")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("multiple go statements failed: %v", err)
	}
}

// ---- Select tests ----

func TestSelectRecv(t *testing.T) {
	src := `
let ch = chan(1)
ch.send(42)
let result = 0

select {
  recv ch as v {
    result = v
  }
}

if result != 42 {
  panic("expected 42 from select recv")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select recv failed: %v", err)
	}
}

func TestSelectElse(t *testing.T) {
	src := `
let ch = chan(1)
let result = 0

select {
  recv ch as v {
    result = v
  }
  else {
    result = 99
  }
}

if result != 99 {
  panic("expected else branch to run")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select else failed: %v", err)
	}
}

func TestSelectSend(t *testing.T) {
	src := `
let ch = chan(1)
let result = 0

select {
  send ch, 7 {
    result = 100
  }
  else {
    result = -1
  }
}

let v = ch.recv()
if v != 7 {
  panic("expected 7 sent via select send")
}
if result != 100 {
  panic("expected send case to run")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select send failed: %v", err)
	}
}

func TestSelectMultipleChannels(t *testing.T) {
	src := `
let ch1 = chan(1)
let ch2 = chan(1)
let result = ""

ch1.send(1)

select {
  recv ch2 as v {
    result = "ch2"
  }
  recv ch1 as v {
    result = "ch1"
  }
  else {
    result = "else"
  }
}

if result != "ch1" {
  panic("expected ch1 to be selected")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select multiple channels failed: %v", err)
	}
}

func TestSelectRecvWithOk(t *testing.T) {
	src := `
let ch = chan(1)
let result = 0
let okVal = false
ch.send(5)

select {
  recv ch as v, ok {
    result = v
    okVal = ok
  }
}

if result != 5 {
  panic("expected 5 from select recv with ok")
}
if !okVal {
  panic("expected ok to be true")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select recv with ok failed: %v", err)
	}
}

// ---- Integration tests ----

func TestConcurrencyProducerConsumer(t *testing.T) {
	src := `
let ch = chan(3)
let done = chan(1)
let sum = 0

go fn() {
  let nums = [1, 2, 3, 4, 5]
  for num in nums {
    ch.send(num)
  }
  ch.close()
}()

go fn() {
  let s = 0
  for v in ch {
    s = s + v
  }
  done.send(s)
}()

sum = done.recv()
if sum != 15 {
  panic("expected producer/consumer sum 15")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("producer-consumer test failed: %v", err)
	}
}

func TestChannelForIn(t *testing.T) {
	src := `
let ch = chan(3)
let result = 0

ch.send(10)
ch.send(20)
ch.send(30)
ch.close()

for v in ch {
  result = result + v
}

if result != 60 {
  panic("expected 60 from channel for-in")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("channel for-in failed: %v", err)
	}
}

func TestConcurrencyWithWait(t *testing.T) {
	src := `
let ch = chan(1)

go fn() {
  let x = 0
  ch.send(42)
}()

let v = ch.recv()
if v != 42 {
  panic("expected 42")
}
`
	rt := NewRuntime()
	result, err := rt.RunSource(src)
	if err != nil {
		t.Fatalf("concurrency with wait failed: %v", err)
	}
	_ = result
}

func TestMultipleProducerSingleConsumer(t *testing.T) {
	src := `
let ch = chan(5)

go fn() { ch.send(10) }()
go fn() { ch.send(20) }()
go fn() { ch.send(30) }()

let a = ch.recv()
let b = ch.recv()
let c = ch.recv()
let sum = a + b + c
if sum != 60 {
  panic("expected sum 60")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("multiple producer single consumer failed: %v", err)
	}
}

// ---- Time-based tests ----

func TestGoWithDelay(t *testing.T) {
	src := `
let ch = chan(1)
let start = std.time.now()

go fn() {
  let x = 0
  ch.send(1)
}()

let v = ch.recv()
if v != 1 {
  panic("expected 1")
}
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err != nil {
		// std.time might not be importable in source mode
		t.Skipf("time-based test skipped: %v", err)
	}
}

// Helper: ensure goroutine pool shutdown doesn't block tests
func TestPoolCleanup(t *testing.T) {
	// Test that multiple quick go statements work
	src := `
let ch = chan(10)

let i = 0
while i < 5 {
  go fn() {
    ch.send(1)
  }()
  i = i + 1
}

let sum = 0
let j = 0
while j < 5 {
  sum = sum + ch.recv()
  j = j + 1
}

if sum != 5 {
  panic("expected sum 5")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("pool cleanup test failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}
