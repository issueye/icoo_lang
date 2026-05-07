package api

import "testing"

func TestRuntimeRunSource_StdCLIParsesRequiredAliasAndUnknownArgs(t *testing.T) {
	rt := NewRuntime()
	rt.SetScriptArgs([]string{"--root", "demo", "--mystery", "x", "tail"})

	src := `
import std.sys.cli as cli

let app = cli.create({
  name: "demo",
  allowUnknownArgs: true
})

app.flagString({
  name: "workspace",
  aliases: ["root"],
  required: true
})

let result = app.run()
if result.flags.workspace != "demo" {
  panic("expected alias to populate workspace")
}
if len(result.unknown) != 3 {
  panic("expected passthrough unknown args")
}
if result.unknown[0] != "--mystery" || result.unknown[1] != "x" || result.unknown[2] != "tail" {
  panic("unexpected unknown args payload")
}
`

	if _, err := rt.RunSource(src); err != nil {
		t.Fatalf("expected std.sys.cli alias/passthrough to succeed, got: %v", err)
	}
}

func TestRuntimeRunSource_StdCLIRejectsMissingRequiredFlag(t *testing.T) {
	rt := NewRuntime()
	rt.SetScriptArgs([]string{})

	src := `
import std.sys.cli as cli

let app = cli.create({
  name: "demo"
})

app.flagString({
  name: "workspace",
  required: true
})

app.run()
`

	if _, err := rt.RunSource(src); err == nil {
		t.Fatal("expected std.sys.cli to reject missing required flag")
	}
}
