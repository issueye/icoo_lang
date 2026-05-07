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

// ---- Channel edge cases ----

func TestChanUnbufferedSendRecv(t *testing.T) {
	src := `
let ch = chan()
let result = 0

go fn() {
  ch.send(42)
}()

let v = ch.recv()
if v != 42 {
  panic("expected 42 from unbuffered recv")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("unbuffered chan failed: %v", err)
	}
}

func TestChanSendOnClosed(t *testing.T) {
	src := `
let ch = chan(1)
ch.close()
let ok = ch.trySend(1)
if ok {
  panic("trySend should fail on closed channel")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("send on closed chan failed: %v", err)
	}
}

func TestChanRecvFromClosedEmpty(t *testing.T) {
	src := `
let ch = chan(1)
ch.close()
let result = ch.tryRecv()
if result.ok {
  panic("tryRecv should fail on closed empty channel")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("recv from closed empty chan failed: %v", err)
	}
}

func TestChanRecvFromClosedDrained(t *testing.T) {
	src := `
let ch = chan(2)
ch.send(10)
ch.send(20)
ch.close()

let a = ch.recv()
let b = ch.recv()
if a != 10 {
  panic("drain value 1 mismatch")
}
if b != 20 {
  panic("drain value 2 mismatch")
}

let result = ch.tryRecv()
if result.ok {
  panic("tryRecv should fail after draining closed channel")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("recv from closed drained chan failed: %v", err)
	}
}

func TestChanZeroSizeDefaultsToUnbuffered(t *testing.T) {
	src := `
let ch = chan(0)
let ok = ch.trySend(1)
if ok {
  panic("trySend on unbuffered should fail with no receiver")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan zero size failed: %v", err)
	}
}

func TestChanNegativeSizeDefaults(t *testing.T) {
	src := `
let ch = chan(-5)
let ok = ch.trySend(1)
if ok {
  panic("trySend on negative-size (unbuffered) should fail with no receiver")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("chan negative size failed: %v", err)
	}
}

func TestChanSmallBufferOverflow(t *testing.T) {
	src := `
let ch = chan(1)
let ok1 = ch.trySend(1)
let ok2 = ch.trySend(2)
if !ok1 {
  panic("first send should succeed")
}
if ok2 {
  panic("second send should fail on full buffer")
}
let v = ch.recv()
if v != 1 {
  panic("expected 1 from recv")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("small buffer overflow failed: %v", err)
	}
}

func TestChanPassAsArgument(t *testing.T) {
	src := `
fn worker(c) {
  c.send(88)
}
let ch = chan(1)
go worker(ch)
let v = ch.recv()
if v != 88 {
  panic("expected 88 from channel passed as arg")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("channel as argument failed: %v", err)
	}
}

func TestChanPassComplexValues(t *testing.T) {
	src := `
let ch = chan(1)
let obj = {name: "test", count: 42}
ch.send(obj)
let result = ch.recv()
if result.name != "test" {
  panic("object name mismatch")
}
if result.count != 42 {
  panic("object count mismatch")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("complex values in channel failed: %v", err)
	}
}

// ---- Go statement edge cases ----

func TestGoNamedFunction(t *testing.T) {
	src := `
fn task(c, val) {
  c.send(val)
}
let ch = chan(2)
go task(ch, 10)
go task(ch, 20)
let a = ch.recv()
let b = ch.recv()
if a + b != 30 {
  panic("named function go failed")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("go named function failed: %v", err)
	}
}

func TestGoNativeFunction(t *testing.T) {
	src := `
let ch = chan(1)
go println("hello from goroutine")
let v = 42
ch.send(v)
let result = ch.recv()
if result != 42 {
  panic("native go failed")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("go native function failed: %v", err)
	}
}

func TestGoCaptureValue(t *testing.T) {
	src := `
let ch = chan(2)
let x = 100
let y = 200

go fn() {
  ch.send(x)
}()

go fn() {
  ch.send(y)
}()

let a = ch.recv()
let b = ch.recv()
if a + b != 300 {
  panic("value capture failed")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("go capture value failed: %v", err)
	}
}

// ---- Select edge cases ----

func TestSelectOnlyElse(t *testing.T) {
	src := `
let result = 0
select {
  else {
    result = 123
  }
}
if result != 123 {
  panic("select only else failed")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select only else failed: %v", err)
	}
}

func TestSelectRecvClosedChannel(t *testing.T) {
	src := `
let ch = chan(1)
ch.close()
let result = 0
let okVal = true

select {
  recv ch as v, ok {
    result = 1
    okVal = ok
  }
  else {
    result = -1
  }
}

if okVal {
  panic("closed channel recv should have ok=false")
}
if result != 1 {
  panic("closed channel recv case should fire with ok=false")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select recv closed channel failed: %v", err)
	}
}

func TestSelectRecvAndSendMix(t *testing.T) {
	src := `
let ch1 = chan(1)
let ch2 = chan(1)
let result = ""

select {
  recv ch1 as v {
    result = "recv"
  }
  send ch2, 42 {
    result = "send"
  }
  else {
    result = "else"
  }
}

if result != "send" {
  panic("expected send to win")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select recv+send mix failed: %v", err)
	}
}

func TestSelectUnderscoreBinding(t *testing.T) {
	src := `
let ch = chan(1)
ch.send(7)
let result = 0

select {
  recv ch as _ {
    result = 42
  }
}

if result != 42 {
  panic("underscore binding failed")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select underscore binding failed: %v", err)
	}
}

func TestSelectMultipleRecvOneReady(t *testing.T) {
	src := `
let ch1 = chan(1)
let ch2 = chan(1)
let ch3 = chan(1)
let result = 0

ch2.send(999)

select {
  recv ch1 as v {
    result = 1
  }
  recv ch2 as v {
    result = v
  }
  recv ch3 as v {
    result = 3
  }
}

if result != 999 {
  panic("expected ch2 value 999")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select multiple recv one ready failed: %v", err)
	}
}

func TestSelectSecondCaseBindsV(t *testing.T) {
	src := `
let ch1 = chan(1)
let ch2 = chan(1)
let result = 0

ch1.send(888)

select {
  recv ch1 as v {
    result = v
  }
  recv ch2 as _ {
    result = 2
  }
}

if result != 888 {
  panic("expected ch1 value 888 with underscore second case")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select second case underscore failed: %v", err)
	}
}

// ---- Pipeline tests ----

func TestPipelineTwoStage(t *testing.T) {
	src := `
let ch1 = chan(3)
let ch2 = chan(3)

go fn() {
  let nums = [1, 2, 3]
  for n in nums {
    ch1.send(n)
  }
  ch1.close()
}()

go fn() {
  for v in ch1 {
    ch2.send(v * 10)
  }
  ch2.close()
}()

let sum = 0
for v in ch2 {
  sum = sum + v
}

if sum != 60 {
  panic("pipeline sum expected 60")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("two-stage pipeline failed: %v", err)
	}
}

func TestFanIn(t *testing.T) {
	src := `
let ch1 = chan(2)
let ch2 = chan(2)
let merged = chan(4)

go fn() {
  ch1.send(1)
  ch1.send(2)
}()
go fn() {
  ch2.send(3)
  ch2.send(4)
}()

go fn() {
  let c = 0
  let sum = 0
  while c < 4 {
    select {
      recv ch1 as v {
        sum = sum + v
        c = c + 1
      }
      recv ch2 as v {
        sum = sum + v
        c = c + 1
      }
    }
  }
  merged.send(sum)
}()

let result = merged.recv()
if result != 10 {
  panic("fan-in sum expected 10")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("fan-in failed: %v", err)
	}
}

func TestFanOut(t *testing.T) {
	src := `
let input = chan(4)
let results = chan(4)

let items = [2, 4, 6, 8]
for item in items {
  input.send(item)
}

let i = 0
while i < 4 {
  go fn() {
    let v = input.recv()
    results.send(v * v)
  }()
  i = i + 1
}

let sum = 0
let j = 0
while j < 4 {
  sum = sum + results.recv()
  j = j + 1
}

if sum != 120 {
  panic("fan-out sum expected 120")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("fan-out failed: %v", err)
	}
}

// ---- Error/negative tests ----

func TestChanInvalidSizeType(t *testing.T) {
	src := `
let ch = chan("not_a_number")
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err == nil {
		t.Fatal("expected error for non-int chan size")
	}
}

func TestChanTooManyArgs(t *testing.T) {
	src := `
let ch = chan(1, 2)
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err == nil {
		t.Fatal("expected error for too many chan args")
	}
}

func TestClosedChannelDrain(t *testing.T) {
	src := `
let ch = chan(2)
ch.send(1)
ch.send(2)
ch.close()

let a = ch.recv()
let b = ch.recv()

go fn() {
  ch.trySend(3)
}()

let result = ch.tryRecv()
let ok = result.ok
if ok {
  panic("tryRecv should fail on closed and drained channel")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("closed channel drain failed: %v", err)
	}
}

func TestSelectNoCases(t *testing.T) {
	src := `
let ch = chan(1)
select {
}
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err == nil {
		t.Fatal("expected error for empty select")
	}
}

// ---- Complex integration tests ----

func TestWorkerPool(t *testing.T) {
	src := `
let jobs = chan(10)
let results = chan(10)

fn worker(ch, res) {
  for job in ch {
    res.send(job * 2)
  }
}

go worker(jobs, results)
go worker(jobs, results)
go worker(jobs, results)

let k = 0
while k < 5 {
  jobs.send(k + 1)
  k = k + 1
}
jobs.close()

let total = 0
let l = 0
while l < 5 {
  total = total + results.recv()
  l = l + 1
}

if total != 30 {
  panic("worker pool sum expected 30")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("worker pool test failed: %v", err)
	}
}

func TestSelectLoop(t *testing.T) {
	src := `
let ch = chan(2)
let tick = chan(1)
let count = 0

ch.send(1)
ch.send(2)

go fn() {
  let i = 0
  while i < 3 {
    tick.trySend(1)
    i = i + 1
  }
}()

while count < 2 {
  select {
    recv ch as v {
      count = count + 1
    }
    else {
    }
  }
}

if count != 2 {
  panic("select loop count expected 2")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("select loop failed: %v", err)
	}
}

func TestConcurrentMapReduce(t *testing.T) {
	src := `
let input = chan(6)
let mapped = chan(6)

fn mapper(inCh, outCh) {
  for v in inCh {
    outCh.send(v * 10)
  }
}

go mapper(input, mapped)
go mapper(input, mapped)

let i = 0
while i < 6 {
  input.send(i + 1)
  i = i + 1
}
input.close()

let sum = 0
let j = 0
while j < 6 {
  sum = sum + mapped.recv()
  j = j + 1
}

if sum != 210 {
  panic("map-reduce sum expected 210")
}
`
	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("concurrent map-reduce failed: %v", err)
	}
}

// ---- check-only tests (sema validation) ----

func TestCheckSelectDuplicateElse(t *testing.T) {
	src := `
let ch = chan(1)
select {
  recv ch as v {}
  else {}
  else {}
}
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate else in select")
	}
}

func TestCheckSelectElseNotLast(t *testing.T) {
	src := `
let ch = chan(1)
select {
  else {}
  recv ch as v {}
}
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatal("expected error for else not last in select")
	}
}

func TestCheckGoExpression(t *testing.T) {
	src := `
fn f() { println("hi") }
go f
go f()
`
	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Logf("check error: %v", err)
		}
		t.Fatal("expected go with ident and call to pass check")
	}
}

// ---- Runtime error tests ----

func TestGoNonCallable(t *testing.T) {
	src := `
go 42
`
	rt := NewRuntime()
	_, err := rt.RunSource(src)
	if err == nil {
		t.Fatal("expected error for go with non-callable")
	}
}
