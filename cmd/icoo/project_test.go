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
	text := string(src)
	if !strings.Contains(text, "fn main() {") {
		t.Fatal("entry template should define the default main function")
	}
	if strings.Count(text, "main()") != 1 {
		t.Fatal("entry template should not manually call the entry function")
	}
}

func TestRunInitSupportsEntryFlags(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")

	if err := runInit([]string{root, "--entry", "app/start.ic", "--entry-fn", "bootstrap"}); err != nil {
		t.Fatalf("expected init with flags to succeed, got: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected project to load, got: %v", err)
	}
	if project.EntryFunction != "bootstrap" {
		t.Fatalf("expected bootstrap entry function, got %q", project.EntryFunction)
	}
	if !strings.HasSuffix(filepath.ToSlash(project.EntryPath), "app/start.ic") {
		t.Fatalf("expected custom entry path, got %q", project.EntryPath)
	}
	data, err := os.ReadFile(project.EntryPath)
	if err != nil {
		t.Fatalf("read custom entry file: %v", err)
	}
	if !strings.Contains(string(data), "fn bootstrap() {") {
		t.Fatalf("expected custom entry function in template, got %q", string(data))
	}
}

func TestRunInitSupportsRootAlias(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")

	if err := runInit([]string{root, "--root-alias", "app"}); err != nil {
		t.Fatalf("expected init with root alias to succeed, got: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected project to load, got: %v", err)
	}
	if project.RootAlias != "app" {
		t.Fatalf("expected root alias %q, got %q", "app", project.RootAlias)
	}

	configData, err := os.ReadFile(filepath.Join(root, projectConfigFileName))
	if err != nil {
		t.Fatalf("read project config: %v", err)
	}
	if !strings.Contains(string(configData), "root_alias") || !strings.Contains(string(configData), "app") {
		t.Fatalf("expected root_alias in project.toml, got %q", string(configData))
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

func TestParseInitArgsRejectsInvalidFlags(t *testing.T) {
	if _, err := parseInitArgs([]string{"--entry-fn", ""}); err == nil {
		t.Fatal("expected empty entry function to be rejected")
	}
	if _, err := parseInitArgs([]string{"--entry", "../main.ic"}); err == nil {
		t.Fatal("expected entry outside project root to be rejected")
	}
}

func TestParseInitArgsRejectsInvalidRootAlias(t *testing.T) {
	cases := [][]string{
		{"--root-alias", "std"},
		{"--root-alias", "app/root"},
		{"--root-alias", "1app"},
	}
	for _, args := range cases {
		if _, err := parseInitArgs(args); err == nil {
			t.Fatalf("expected invalid root alias to be rejected for args %v", args)
		}
	}
}

func TestParseInitArgsAcceptsAtRootAlias(t *testing.T) {
	opts, err := parseInitArgs([]string{"--root-alias", "@"})
	if err != nil {
		t.Fatalf("expected @ root alias to be accepted, got: %v", err)
	}
	if opts.RootAlias != "@" {
		t.Fatalf("expected root alias %q, got %q", "@", opts.RootAlias)
	}
}

func TestLoadProjectReadsAtRootAlias(t *testing.T) {
	root := t.TempDir()
	entryPath := filepath.Join(root, "src", "main.ic")
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}
	config := "[project]\nname = \"demo\"\nentry = \"src/main.ic\"\nroot_alias = \"@\"\n"
	if err := os.WriteFile(filepath.Join(root, projectConfigFileName), []byte(config), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected loadProject to succeed, got: %v", err)
	}
	if project.RootAlias != "@" {
		t.Fatalf("expected root alias %q, got %q", "@", project.RootAlias)
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

func TestLoadProjectReadsRootAlias(t *testing.T) {
	root := t.TempDir()
	entryPath := filepath.Join(root, "src", "main.ic")
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}
	config := "[project]\nname = \"demo\"\nentry = \"src/main.ic\"\nroot_alias = \"app\"\n"
	if err := os.WriteFile(filepath.Join(root, projectConfigFileName), []byte(config), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	project, err := loadProject(root)
	if err != nil {
		t.Fatalf("expected loadProject to succeed, got: %v", err)
	}
	if project.RootAlias != "app" {
		t.Fatalf("expected root alias %q, got %q", "app", project.RootAlias)
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

func TestResolveRunTargetLoadsProjectContextForFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := runInit([]string{root, "--root-alias", "app"}); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	libPath := filepath.Join(root, "lib", "helper.ic")
	if err := os.MkdirAll(filepath.Dir(libPath), 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(libPath, []byte("export fn greet() { return \"hi\" }\n"), 0o644); err != nil {
		t.Fatalf("write lib file: %v", err)
	}

	entryPath := filepath.Join(root, "src", "main.ic")
	source := strings.Join([]string{
		"import \"app/lib/helper.ic\" as helper",
		"",
		"helper.greet()",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	resolved, err := resolveRunTarget(entryPath)
	if err != nil {
		t.Fatalf("expected resolveRunTarget for file to succeed, got: %v", err)
	}
	if filepath.Clean(resolved.Root) != filepath.Clean(root) {
		t.Fatalf("expected root %q, got %q", root, resolved.Root)
	}
	if resolved.RootAlias != "app" {
		t.Fatalf("expected root alias %q, got %q", "app", resolved.RootAlias)
	}
	if resolved.EntryFunction != "" {
		t.Fatalf("expected direct file runs to skip project entry function, got %q", resolved.EntryFunction)
	}
}

func TestRunProjectPathUsesProjectContextForFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := runInit([]string{root, "--root-alias", "app"}); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	libPath := filepath.Join(root, "lib", "helper.ic")
	if err := os.MkdirAll(filepath.Dir(libPath), 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(libPath, []byte("export fn greet() { return \"hi\" }\n"), 0o644); err != nil {
		t.Fatalf("write lib file: %v", err)
	}

	entryPath := filepath.Join(root, "src", "main.ic")
	source := strings.Join([]string{
		"import \"app/lib/helper.ic\" as helper",
		"",
		"helper.greet()",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	if err := runProjectPath(entryPath, nil); err != nil {
		t.Fatalf("expected runProjectPath to use project context for file, got: %v", err)
	}
}
