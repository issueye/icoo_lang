package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitCreatesProjectScaffold(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")

	if err := runInit([]string{root}); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	configPath := filepath.Join(root, projectConfigFileName)
	entryPath := filepath.Join(root, filepath.FromSlash(defaultProjectEntry))
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected project config to exist, got: %v", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		t.Fatalf("expected entry file to exist, got: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected project to load, got: %v", err)
	}
	if project.EntryFunction != defaultProjectEntryFunction {
		t.Fatalf("expected default entry function %q, got %q", defaultProjectEntryFunction, project.EntryFunction)
	}
	if filepath.Clean(project.EntryPath) != filepath.Clean(entryPath) {
		t.Fatalf("expected entry path %q, got %q", entryPath, project.EntryPath)
	}

	src, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read entry file: %v", err)
	}
	if strings.Contains(string(src), "main()") && !strings.Contains(string(src), "fn main()") {
		t.Fatal("entry template should define main without manually calling it")
	}
}

func TestRunInitRejectsExistingProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, projectConfigFileName), []byte("[project]\nentry=\"src/main.ic\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := runInit([]string{root}); err == nil {
		t.Fatal("expected init to reject existing project")
	}
}

func TestLoadProjectDefaultsEntryFunction(t *testing.T) {
	root := t.TempDir()
	entryPath := filepath.Join(root, "src", "main.ic")
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}
	config := "[project]\nname = \"demo\"\nentry = \"src/main.ic\"\n"
	if err := os.WriteFile(filepath.Join(root, projectConfigFileName), []byte(config), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected loadProject to succeed, got: %v", err)
	}
	if project.EntryFunction != defaultProjectEntryFunction {
		t.Fatalf("expected default entry function %q, got %q", defaultProjectEntryFunction, project.EntryFunction)
	}
}

func TestLoadProjectRejectsEntryOutsideRoot(t *testing.T) {
	root := t.TempDir()
	config := "[project]\nname = \"demo\"\nentry = \"../main.ic\"\nentry_function = \"main\"\n"
	if err := os.WriteFile(filepath.Join(root, projectConfigFileName), []byte(config), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	if _, err := loadProject(root); err == nil {
		t.Fatal("expected loadProject to reject entry outside root")
	}
}

func TestResolveRunTargetLoadsProjectDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := runInit([]string{root}); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	resolved, err := resolveRunTarget(root)
	if err != nil {
		t.Fatalf("expected resolveRunTarget to succeed, got: %v", err)
	}
	if resolved.EntryFunction != defaultProjectEntryFunction {
		t.Fatalf("expected entry function %q, got %q", defaultProjectEntryFunction, resolved.EntryFunction)
	}
	if !strings.HasSuffix(filepath.ToSlash(resolved.EntryPath), "src/main.ic") {
		t.Fatalf("expected entry path to end with src/main.ic, got %q", resolved.EntryPath)
	}
}
