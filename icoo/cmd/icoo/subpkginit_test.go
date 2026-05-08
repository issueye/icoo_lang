package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSubpackageName(t *testing.T) {
	got := buildSubpackageName("issueye/agent", filepath.Join("pkg", "config"))
	if got != "issueye/agent/pkg/config" {
		t.Fatalf("expected derived subpackage name %q, got %q", "issueye/agent/pkg/config", got)
	}
}

func TestRunInitSubpackageCreatesNestedScaffold(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	target := filepath.Join(root, "pkg", "config")

	if err := runInitSubpackage([]string{target, "--parent", "issueye/agent"}); err != nil {
		t.Fatalf("expected init-subpkg to succeed, got: %v", err)
	}

	manifest, err := loadPackageManifest(target)
	if err != nil {
		t.Fatalf("load subpackage manifest: %v", err)
	}
	if manifest.Name != "issueye/agent/pkg/config" {
		t.Fatalf("expected derived subpackage name, got %q", manifest.Name)
	}

	libData, err := os.ReadFile(filepath.Join(target, "lib.ic"))
	if err != nil {
		t.Fatalf("read subpackage lib.ic: %v", err)
	}
	if !strings.Contains(string(libData), `import "@/src/main.ic" as mainModule`) {
		t.Fatalf("expected subpackage lib.ic to import default entry, got:\n%s", string(libData))
	}
	if _, err := os.Stat(filepath.Join(target, packageBuildScriptFileName)); err != nil {
		t.Fatalf("expected subpackage build script to exist, got: %v", err)
	}
}

func TestParseInitSubpackageArgsRejectsMissingParent(t *testing.T) {
	if _, err := parseInitSubpackageArgs([]string{"pkg/config"}); err == nil {
		t.Fatal("expected init-subpkg to require --parent")
	}
}
