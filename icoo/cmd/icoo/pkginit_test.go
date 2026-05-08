package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"icoo_lang/pkg/api"
)

func TestRunInitPackageCreatesScaffold(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hello_pkg")

	if err := runInitPackage([]string{root}); err != nil {
		t.Fatalf("expected init-pkg to succeed, got: %v", err)
	}

	for _, path := range []string{
		filepath.Join(root, packageConfigFileName),
		filepath.Join(root, packageBuildScriptFileName),
		filepath.Join(root, "lib.ic"),
		filepath.Join(root, "src", "main.ic"),
		filepath.Join(root, "examples"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffold path to exist %q, got: %v", path, err)
		}
	}

	manifest, err := loadPackageManifest(root)
	if err != nil {
		t.Fatalf("load package manifest: %v", err)
	}
	if manifest.Name != "hello_pkg" {
		t.Fatalf("expected default package name %q, got %q", "hello_pkg", manifest.Name)
	}
	if manifest.Export != defaultPackageExport || manifest.Entry != defaultPackageEntry || manifest.RootAlias != "@" {
		t.Fatalf("unexpected package manifest defaults: %+v", manifest)
	}

	libData, err := os.ReadFile(filepath.Join(root, "lib.ic"))
	if err != nil {
		t.Fatalf("read lib.ic: %v", err)
	}
	if !strings.Contains(string(libData), `import "@/src/main.ic" as mainModule`) {
		t.Fatalf("expected lib.ic to import package entry, got:\n%s", string(libData))
	}

	buildData, err := os.ReadFile(filepath.Join(root, packageBuildScriptFileName))
	if err != nil {
		t.Fatalf("read build.ps1: %v", err)
	}
	if !strings.Contains(string(buildData), "package $packageRoot $Output") {
		t.Fatalf("expected build.ps1 to package current module, got:\n%s", string(buildData))
	}
}

func TestRunInitPackageSupportsFlags(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_pkg")

	if err := runInitPackage([]string{
		root,
		"--name", "acme/demo",
		"--version", "1.2.3",
		"--entry", "src/bootstrap.ic",
		"--entry-fn", "bootstrap",
		"--export", "lib/api.ic",
		"--root-alias", "app",
	}); err != nil {
		t.Fatalf("expected init-pkg with flags to succeed, got: %v", err)
	}

	manifest, err := loadPackageManifest(root)
	if err != nil {
		t.Fatalf("load package manifest: %v", err)
	}
	if manifest.Name != "acme/demo" || manifest.Version != "1.2.3" || manifest.Entry != "src/bootstrap.ic" || manifest.Export != "lib/api.ic" || manifest.RootAlias != "app" {
		t.Fatalf("unexpected package manifest: %+v", manifest)
	}

	libData, err := os.ReadFile(filepath.Join(root, "lib", "api.ic"))
	if err != nil {
		t.Fatalf("read custom export file: %v", err)
	}
	if !strings.Contains(string(libData), `import "app/src/bootstrap.ic" as mainModule`) {
		t.Fatalf("expected export file to use custom root alias, got:\n%s", string(libData))
	}
}

func TestRunPackageDirectoryLoadsPkgTomlDefaults(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo_pkg")
	outputPath := filepath.Join(t.TempDir(), "demo_pkg.icpkg")

	if err := runInitPackage([]string{root, "--name", "acme/demo"}); err != nil {
		t.Fatalf("expected init-pkg to succeed, got: %v", err)
	}
	if err := runPackage([]string{root, outputPath}); err != nil {
		t.Fatalf("expected package dir to succeed, got: %v", err)
	}

	archive, err := api.LoadBundleFile(outputPath)
	if err != nil {
		t.Fatalf("load packaged archive: %v", err)
	}
	if archive.PackageName != "acme/demo" || archive.PackageVersion != defaultPackageVersion || archive.Export != defaultPackageExport {
		t.Fatalf("unexpected packaged archive metadata: %+v", archive)
	}

	rt := api.NewRuntime()
	defer func() {
		_ = rt.Close()
	}()
	if _, err := rt.RunBundleFile(outputPath); err != nil {
		t.Fatalf("expected generated package archive to run, got: %v", err)
	}
}
