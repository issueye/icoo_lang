package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"icoo_lang/pkg/api"
)

func TestRunBundleBundlesFileImports(t *testing.T) {
	root := t.TempDir()
	entryPath := filepath.Join(root, "main.ic")
	libPath := filepath.Join(root, "lib", "math.ic")

	if err := os.MkdirAll(filepath.Dir(libPath), 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(libPath, []byte("export fn answer() {\n  return 42\n}\n"), 0o644); err != nil {
		t.Fatalf("write lib file: %v", err)
	}
	entrySource := strings.Join([]string{
		"import \"./lib/math.ic\" as math",
		"",
		"if math.answer() != 42 {",
		"  panic(\"unexpected answer\")",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(entrySource), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	bundlePath := filepath.Join(root, "app.icb")
	if err := runBundle([]string{entryPath, bundlePath}); err != nil {
		t.Fatalf("expected bundle to succeed, got: %v", err)
	}

	rt := api.NewRuntime()
	if _, err := rt.RunBundleFile(bundlePath); err != nil {
		t.Fatalf("expected bundle to run, got: %v", err)
	}
}

func TestRunBundleBundlesProjectRootAlias(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if err := runInit([]string{root, "--root-alias", "app"}); err != nil {
		t.Fatalf("expected init to succeed, got: %v", err)
	}

	entryPath := filepath.Join(root, "src", "main.ic")
	libPath := filepath.Join(root, "src", "lib", "message.ic")
	if err := os.MkdirAll(filepath.Dir(libPath), 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(libPath, []byte("export fn text() {\n  return \"hello\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write lib file: %v", err)
	}
	entrySource := strings.Join([]string{
		"import \"app/src/lib/message.ic\" as message",
		"",
		"fn main() {",
		"  if message.text() != \"hello\" {",
		"    panic(\"unexpected message\")",
		"  }",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(entrySource), 0o644); err != nil {
		t.Fatalf("write entry file: %v", err)
	}

	bundlePath := filepath.Join(filepath.Dir(root), "demo.icb")
	if err := runBundle([]string{root, bundlePath}); err != nil {
		t.Fatalf("expected project bundle to succeed, got: %v", err)
	}

	rt := api.NewRuntime()
	if _, err := rt.RunBundleFile(bundlePath); err != nil {
		t.Fatalf("expected project bundle to run, got: %v", err)
	}
}

func TestAppendBundleToExecutableRoundTrip(t *testing.T) {
	root := t.TempDir()
	stubPath := filepath.Join(root, "stub.bin")
	outputPath := filepath.Join(root, "app.bin")
	payload := []byte(`{"version":1,"entry":"main.ic","modules":{"main.ic":"fn main() {}"}}`)

	if err := os.WriteFile(stubPath, []byte("stub-binary"), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	if err := appendBundleToExecutable(stubPath, outputPath, payload); err != nil {
		t.Fatalf("append bundle to executable: %v", err)
	}

	embedded, err := readEmbeddedBundle(outputPath)
	if err != nil {
		t.Fatalf("read embedded bundle: %v", err)
	}
	if !bytes.Equal(embedded, payload) {
		t.Fatalf("expected embedded payload %q, got %q", string(payload), string(embedded))
	}
}

func TestLoadArchiveForInspectBundleAndExecutable(t *testing.T) {
	root := t.TempDir()
	archive := &api.BundleArchive{
		Version: api.BundleVersion,
		Entry:   "main.ic",
		Modules: map[string]string{"main.ic": "fn main() {}\n"},
	}
	bundleData, err := jsonMarshalForTest(archive)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	bundlePath := filepath.Join(root, "app.icb")
	if err := os.WriteFile(bundlePath, bundleData, 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	loadedBundle, kind, err := loadArchiveForInspect(bundlePath)
	if err != nil {
		t.Fatalf("inspect bundle: %v", err)
	}
	if kind != "bundle" || loadedBundle.Entry != archive.Entry {
		t.Fatalf("unexpected bundle inspect result: kind=%q entry=%q", kind, loadedBundle.Entry)
	}

	stubPath := filepath.Join(root, "stub.bin")
	exePath := filepath.Join(root, "app.bin")
	if err := os.WriteFile(stubPath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	if err := appendBundleToExecutable(stubPath, exePath, bundleData); err != nil {
		t.Fatalf("append bundle: %v", err)
	}
	loadedExe, kind, err := loadArchiveForInspect(exePath)
	if err != nil {
		t.Fatalf("inspect executable: %v", err)
	}
	if kind != "executable" || loadedExe.Entry != archive.Entry {
		t.Fatalf("unexpected executable inspect result: kind=%q entry=%q", kind, loadedExe.Entry)
	}
}

func TestPrintArchiveSummary(t *testing.T) {
	archive := &api.BundleArchive{
		Version:       api.BundleVersion,
		Entry:         "src/main.ic",
		EntryFunction: "main",
		RootAlias:     "app",
		ProjectRoot:   "src",
		Modules: map[string]string{
			"src/main.ic":      "fn main() {}\n",
			"src/lib/util.ic":  "export fn value() { return 1 }\n",
		},
	}

	output := captureStdoutForTest(t, func() {
		printArchiveSummary("demo.icb", "bundle", archive)
	})
	for _, expected := range []string{
		"type: bundle",
		"path: demo.icb",
		"entry: src/main.ic",
		"entry_function: main",
		"root_alias: app",
		"project_root: src",
		"module_count: 2",
		"  - src/lib/util.ic",
		"  - src/main.ic",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestRunExtractFromExecutable(t *testing.T) {
	root := t.TempDir()
	archive := &api.BundleArchive{
		Version: api.BundleVersion,
		Entry:   "main.ic",
		Modules: map[string]string{"main.ic": "fn main() {}\n"},
	}
	bundleData, err := jsonMarshalForTest(archive)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	stubPath := filepath.Join(root, "stub.bin")
	exePath := filepath.Join(root, "app.bin")
	if err := os.WriteFile(stubPath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	if err := appendBundleToExecutable(stubPath, exePath, bundleData); err != nil {
		t.Fatalf("append bundle: %v", err)
	}

	outputPath := filepath.Join(root, "app.icb")
	if err := runExtract([]string{exePath, outputPath}); err != nil {
		t.Fatalf("extract executable bundle: %v", err)
	}

	extracted, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read extracted bundle: %v", err)
	}
	if !bytes.Equal(extracted, bundleData) {
		t.Fatalf("expected extracted payload %q, got %q", string(bundleData), string(extracted))
	}
}

func TestResolveExtractOutputDefaults(t *testing.T) {
	root := t.TempDir()
	exePath := filepath.Join(root, "demo.exe")
	bundlePath := filepath.Join(root, "demo.icb")

	gotExe, err := resolveExtractOutput(exePath, "")
	if err != nil {
		t.Fatalf("resolve extract output for exe: %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(gotExe), strings.ToLower(filepath.Join(root, "demo.icb"))) {
		t.Fatalf("expected exe extract output to end with demo.icb, got %q", gotExe)
	}

	gotBundle, err := resolveExtractOutput(bundlePath, "")
	if err != nil {
		t.Fatalf("resolve extract output for bundle: %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(gotBundle), strings.ToLower(filepath.Join(root, "demo.extracted.icb"))) {
		t.Fatalf("expected bundle extract output to end with demo.extracted.icb, got %q", gotBundle)
	}
}

func TestParseBuildArgsSupportsMetadataFlags(t *testing.T) {
	opts, err := parseBuildArgs([]string{
		"demo",
		"out.exe",
		"--metadata", "build.json",
		"--icon", "app.png",
		"--version", "1.2.3",
		"--product-name", "Demo App",
	})
	if err != nil {
		t.Fatalf("parse build args: %v", err)
	}
	if opts.Target != "demo" || opts.Output != "out.exe" {
		t.Fatalf("unexpected positional args: %+v", opts)
	}
	if opts.MetadataPath != "build.json" || opts.IconPath != "app.png" || opts.Version != "1.2.3" || opts.ProductName != "Demo App" {
		t.Fatalf("unexpected metadata args: %+v", opts)
	}
}

func TestLoadBuildMetadataIntoAppliesJSONDefaults(t *testing.T) {
	root := t.TempDir()
	metaPath := filepath.Join(root, "build.json")
	iconPath := filepath.Join(root, "icon.png")

	if err := os.WriteFile(iconPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("write icon placeholder: %v", err)
	}
	meta := `{
  "icon": "icon.png",
  "version": "2.3.4",
  "product_name": "Demo Product",
  "file_description": "Demo Description",
  "company_name": "OpenAI",
  "copyright": "Copyright 2026",
  "internal_name": "demo-app"
}`
	if err := os.WriteFile(metaPath, []byte(meta), 0o644); err != nil {
		t.Fatalf("write metadata json: %v", err)
	}

	opts := buildOptions{MetadataPath: metaPath, ProductName: "CLI Override"}
	if err := loadBuildMetadataInto(&opts); err != nil {
		t.Fatalf("load build metadata: %v", err)
	}

	if opts.IconPath != iconPath {
		t.Fatalf("expected resolved icon path %q, got %q", iconPath, opts.IconPath)
	}
	if opts.Version != "2.3.4" {
		t.Fatalf("expected version from json, got %q", opts.Version)
	}
	if opts.ProductName != "CLI Override" {
		t.Fatalf("expected cli override to win, got %q", opts.ProductName)
	}
	if opts.FileDescription != "Demo Description" || opts.CompanyName != "OpenAI" || opts.InternalName != "demo-app" {
		t.Fatalf("unexpected metadata merge result: %+v", opts)
	}
}

func TestApplyWindowsBuildMetadataWritesVersionInfo(t *testing.T) {
	root := t.TempDir()
	srcPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	tempStub := filepath.Join(root, "stub.exe")
	iconPath := filepath.Join(root, "app.png")

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 0x33, G: 0x99, B: 0xdd, A: 0xff})
		}
	}
	iconFile, err := os.Create(iconPath)
	if err != nil {
		t.Fatalf("create icon: %v", err)
	}
	if err := png.Encode(iconFile, img); err != nil {
		_ = iconFile.Close()
		t.Fatalf("encode icon: %v", err)
	}
	if err := iconFile.Close(); err != nil {
		t.Fatalf("close icon: %v", err)
	}

	opts := buildOptions{
		IconPath:    iconPath,
		Version:     "1.2.3",
		ProductName: "Demo Product",
	}
	outputPath := filepath.Join(root, "demo.exe")
	if err := applyWindowsBuildMetadata(srcPath, tempStub, outputPath, opts); err != nil {
		t.Fatalf("apply windows build metadata: %v", err)
	}

	exe, err := os.Open(tempStub)
	if err != nil {
		t.Fatalf("open temp stub: %v", err)
	}
	defer exe.Close()

	rs, err := winres.LoadFromEXE(exe)
	if err != nil {
		t.Fatalf("load resources: %v", err)
	}
	if _, err := rs.GetIcon(winres.Name("APPICON")); err != nil {
		t.Fatalf("expected APPICON resource, got: %v", err)
	}

	viData := rs.Get(winres.RT_VERSION, winres.ID(1), winres.LCIDDefault)
	if len(viData) == 0 {
		t.Fatal("expected version resource")
	}
	vi, err := version.FromBytes(viData)
	if err != nil {
		t.Fatalf("parse version resource: %v", err)
	}
	table := vi.Table().GetMainTranslation()
	if table[version.ProductName] != "Demo Product" {
		t.Fatalf("expected product name %q, got %q", "Demo Product", table[version.ProductName])
	}
	if table[version.ProductVersion] != "1.2.3" {
		t.Fatalf("expected product version %q, got %q", "1.2.3", table[version.ProductVersion])
	}
}

func jsonMarshalForTest(archive *api.BundleArchive) ([]byte, error) {
	return json.Marshal(archive)
}

func captureStdoutForTest(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(data)
}
