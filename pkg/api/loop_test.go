package api

import "testing"

func TestRuntimeRunSource_ForLoopBreakContinue(t *testing.T) {
	src := `
let i = 0
let sum = 0

for i < 10 {
  i = i + 1

  if i == 3 {
    continue
  }

  if i == 8 {
    break
  }

  sum = sum + i
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected for loop run to succeed, got error: %v", err)
	}
}

func TestRuntimeRunSource_InfiniteForWithBreak(t *testing.T) {
	src := `
let i = 0

for {
  i = i + 1
  if i == 2 {
    break
  }
}
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected infinite for with break to succeed, got error: %v", err)
	}
}

func TestRuntimeCheckSource_RejectsBreakOutsideLoop(t *testing.T) {
	src := `break`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatalf("expected break outside loop to fail check")
	}
}

func TestRuntimeCheckSource_RejectsContinueOutsideLoop(t *testing.T) {
	src := `continue`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) == 0 {
		t.Fatalf("expected continue outside loop to fail check")
	}
}
