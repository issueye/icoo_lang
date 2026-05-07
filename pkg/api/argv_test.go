package api

import (
	"testing"
)

func TestRuntimeRunSource_BuiltinArgvUsesScriptArgs(t *testing.T) {
	rt := NewRuntime()
	rt.SetScriptArgs([]string{"--workspace", "demo", "--task", "inspect"})

	src := `
let args = argv()
if len(args) != 4 {
  panic("expected argv length")
}
if args[0] != "--workspace" || args[1] != "demo" {
  panic("unexpected argv prefix")
}
if args[2] != "--task" || args[3] != "inspect" {
  panic("unexpected argv tail")
}
`

	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected builtin argv to succeed, got: %v", err)
	}
}
