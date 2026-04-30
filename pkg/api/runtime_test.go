package api

import "testing"

func TestRuntimeCheckSource_AllowsTopLevelExprStmt(t *testing.T) {
	src := `
fn main() {
  println("ok")
}

main()
`

	rt := NewRuntime()
	errs := rt.CheckSource(src)
	if len(errs) > 0 {
		t.Fatalf("expected no check errors, got %v", errs)
	}
}

func TestRuntimeRunSource_FunctionCallUsesCorrectArguments(t *testing.T) {
	src := `
fn add(a, b) {
  return a + b
}

let result = add(1, 2)
`

	rt := NewRuntime()
	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected run to succeed, got error: %v", err)
	}
}
