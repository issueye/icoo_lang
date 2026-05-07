package vm

import (
	"errors"
	"fmt"
	"testing"

	"icoo_lang/internal/runtime"
)

func TestErrorToValue_PreservesExistingRuntimeError(t *testing.T) {
	machine := New()
	existing := &runtime.ErrorValue{Message: "boom"}

	got := machine.errorToValue(existing)
	if got != existing {
		t.Fatalf("expected existing runtime error pointer to be preserved")
	}
}

func TestErrorToValue_ConvertsWrappedGoErrorChain(t *testing.T) {
	machine := New()
	root := errors.New("root")
	wrapped := fmt.Errorf("middle: %w", root)
	top := fmt.Errorf("outer: %w", wrapped)

	got := machine.errorToValue(top)
	if got.Message != "outer: middle: root" {
		t.Fatalf("unexpected top message: %q", got.Message)
	}
	if got.Cause == nil || got.Cause.Message != "middle: root" {
		t.Fatalf("unexpected middle cause: %#v", got.Cause)
	}
	if got.Cause.Cause == nil || got.Cause.Cause.Message != "root" {
		t.Fatalf("unexpected root cause: %#v", got.Cause)
	}
}

func TestErrorToValue_PreservesWrappedRuntimeErrorIdentity(t *testing.T) {
	machine := New()
	root := &runtime.ErrorValue{
		Message: "root",
		Stack: []runtime.StackFrame{{Function: "inner", File: "root.ic", Line: 3}},
	}

	got := machine.errorToValue(fmt.Errorf("outer: %w", root))
	if got != root {
		t.Fatalf("expected wrapped runtime error identity to be preserved")
	}
	if got.Error() != "root\n  at inner (root.ic:3)" {
		t.Fatalf("unexpected preserved runtime error formatting: %q", got.Error())
	}
}
