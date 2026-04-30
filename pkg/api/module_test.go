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
