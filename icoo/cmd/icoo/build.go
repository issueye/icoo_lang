package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"icoo_lang/pkg/api"
)

const embeddedBundleMagic = "ICOO_EMBEDDED_BUNDLE_V1"

type buildOptions struct {
	Target          string
	Output          string
	MetadataPath    string
	IconPath        string
	Version         string
	ProductName     string
	FileDescription string
	CompanyName     string
	Copyright       string
	InternalName    string
}

type buildMetadataFile struct {
	IconPath        string `json:"icon"`
	Version         string `json:"version"`
	ProductName     string `json:"product_name"`
	FileDescription string `json:"file_description"`
	CompanyName     string `json:"company_name"`
	Copyright       string `json:"copyright"`
	InternalName    string `json:"internal_name"`
}

func runBuild(args []string) error {
	opts, err := parseBuildArgs(args)
	if err != nil {
		return err
	}
	if err := loadBuildMetadataInto(&opts); err != nil {
		return err
	}

	archive, _, err := buildArchive(buildArchiveOptions{
		Target:     opts.Target,
		ArchiveExt: bundleFileExt,
		Kind:       api.ArchiveKindApplication,
	})
	if err != nil {
		return err
	}
	bundleData, err := json.Marshal(archive)
	if err != nil {
		return fmt.Errorf("encode bundle: %w", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	outputPath, err := resolveBuildOutput(opts.Target, opts.Output)
	if err != nil {
		return err
	}
	stubPath, cleanup, err := prepareBuildStub(selfPath, outputPath, opts)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}
	if err := appendBundleToExecutable(stubPath, outputPath, bundleData); err != nil {
		return err
	}

	fmt.Printf("built executable: %s\n", outputPath)
	return nil
}

func parseBuildArgs(args []string) (buildOptions, error) {
	opts := buildOptions{}
	positionals := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--metadata" || arg == "--meta":
			i++
			if i >= len(args) {
				return buildOptions{}, fmt.Errorf("usage: icoo build <file|dir> [output] [--metadata path] [--icon path] [--version value] [--product-name value]")
			}
			opts.MetadataPath = args[i]
		case strings.HasPrefix(arg, "--metadata="):
			opts.MetadataPath = strings.TrimPrefix(arg, "--metadata=")
		case strings.HasPrefix(arg, "--meta="):
			opts.MetadataPath = strings.TrimPrefix(arg, "--meta=")
		case arg == "--icon":
			i++
			if i >= len(args) {
				return buildOptions{}, fmt.Errorf("usage: icoo build <file|dir> [output] [--metadata path] [--icon path] [--version value] [--product-name value]")
			}
			opts.IconPath = args[i]
		case strings.HasPrefix(arg, "--icon="):
			opts.IconPath = strings.TrimPrefix(arg, "--icon=")
		case arg == "--version":
			i++
			if i >= len(args) {
				return buildOptions{}, fmt.Errorf("usage: icoo build <file|dir> [output] [--metadata path] [--icon path] [--version value] [--product-name value]")
			}
			opts.Version = args[i]
		case strings.HasPrefix(arg, "--version="):
			opts.Version = strings.TrimPrefix(arg, "--version=")
		case arg == "--product-name":
			i++
			if i >= len(args) {
				return buildOptions{}, fmt.Errorf("usage: icoo build <file|dir> [output] [--metadata path] [--icon path] [--version value] [--product-name value]")
			}
			opts.ProductName = args[i]
		case strings.HasPrefix(arg, "--product-name="):
			opts.ProductName = strings.TrimPrefix(arg, "--product-name=")
		case strings.HasPrefix(arg, "--"):
			return buildOptions{}, fmt.Errorf("unknown option: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) < 1 || len(positionals) > 2 {
		return buildOptions{}, fmt.Errorf("usage: icoo build <file|dir> [output] [--metadata path] [--icon path] [--version value] [--product-name value]")
	}
	opts.Target = positionals[0]
	if len(positionals) == 2 {
		opts.Output = positionals[1]
	}
	opts.MetadataPath = strings.TrimSpace(opts.MetadataPath)
	opts.IconPath = strings.TrimSpace(opts.IconPath)
	opts.Version = strings.TrimSpace(opts.Version)
	opts.ProductName = strings.TrimSpace(opts.ProductName)
	opts.FileDescription = strings.TrimSpace(opts.FileDescription)
	opts.CompanyName = strings.TrimSpace(opts.CompanyName)
	opts.Copyright = strings.TrimSpace(opts.Copyright)
	opts.InternalName = strings.TrimSpace(opts.InternalName)
	return opts, nil
}

func loadBuildMetadataInto(opts *buildOptions) error {
	if opts == nil || opts.MetadataPath == "" {
		return nil
	}

	absPath, err := filepath.Abs(opts.MetadataPath)
	if err != nil {
		return fmt.Errorf("resolve metadata file: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read metadata file: %w", err)
	}

	var meta buildMetadataFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse metadata file: %w", err)
	}

	if opts.IconPath == "" {
		opts.IconPath = strings.TrimSpace(meta.IconPath)
		if opts.IconPath != "" && !filepath.IsAbs(opts.IconPath) {
			opts.IconPath = filepath.Join(filepath.Dir(absPath), opts.IconPath)
		}
	}
	if opts.Version == "" {
		opts.Version = strings.TrimSpace(meta.Version)
	}
	if opts.ProductName == "" {
		opts.ProductName = strings.TrimSpace(meta.ProductName)
	}
	if opts.FileDescription == "" {
		opts.FileDescription = strings.TrimSpace(meta.FileDescription)
	}
	if opts.CompanyName == "" {
		opts.CompanyName = strings.TrimSpace(meta.CompanyName)
	}
	if opts.Copyright == "" {
		opts.Copyright = strings.TrimSpace(meta.Copyright)
	}
	if opts.InternalName == "" {
		opts.InternalName = strings.TrimSpace(meta.InternalName)
	}
	return nil
}

func resolveBuildOutput(target string, output string) (string, error) {
	if strings.TrimSpace(output) != "" {
		absPath, err := filepath.Abs(output)
		if err != nil {
			return "", fmt.Errorf("resolve build output: %w", err)
		}
		if runtime.GOOS == "windows" && filepath.Ext(absPath) == "" {
			absPath += ".exe"
		}
		return absPath, nil
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("stat build target: %w", err)
	}

	name := ""
	if info.IsDir() {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return "", fmt.Errorf("resolve build target: %w", err)
		}
		name = filepath.Base(absTarget)
		return filepath.Join(filepath.Dir(absTarget), executableName(name)), nil
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve build target: %w", err)
	}
	name = strings.TrimSuffix(filepath.Base(absTarget), filepath.Ext(absTarget))
	return filepath.Join(filepath.Dir(absTarget), executableName(name)), nil
}

func executableName(base string) string {
	if runtime.GOOS == "windows" && filepath.Ext(base) == "" {
		return base + ".exe"
	}
	return base
}

func prepareBuildStub(selfPath string, outputPath string, opts buildOptions) (string, func(), error) {
	if !requiresNativeMetadata(opts) {
		return selfPath, nil, nil
	}
	if runtime.GOOS != "windows" {
		return "", nil, fmt.Errorf("--icon, --version, and --product-name are currently supported on Windows only")
	}

	tempFile, err := os.CreateTemp(filepath.Dir(outputPath), "icoo-build-stub-*.exe")
	if err != nil {
		return "", nil, fmt.Errorf("create temp launcher: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", nil, fmt.Errorf("close temp launcher: %w", err)
	}

	if err := applyWindowsBuildMetadata(selfPath, tempPath, outputPath, opts); err != nil {
		_ = os.Remove(tempPath)
		return "", nil, err
	}
	return tempPath, func() { _ = os.Remove(tempPath) }, nil
}

func requiresNativeMetadata(opts buildOptions) bool {
	return opts.IconPath != "" ||
		opts.Version != "" ||
		opts.ProductName != "" ||
		opts.FileDescription != "" ||
		opts.CompanyName != "" ||
		opts.Copyright != "" ||
		opts.InternalName != ""
}

func applyWindowsBuildMetadata(srcPath string, tempPath string, outputPath string, opts buildOptions) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open build stub: %w", err)
	}
	defer srcFile.Close()

	rs, err := winres.LoadFromEXE(srcFile)
	if err != nil {
		if err == winres.ErrNoResources {
			rs = &winres.ResourceSet{}
		} else {
			return fmt.Errorf("load executable resources: %w", err)
		}
	}

	if opts.IconPath != "" {
		icon, err := loadBuildIcon(opts.IconPath)
		if err != nil {
			return err
		}
		rs.SetIcon(winres.Name("APPICON"), icon)
	}

	if opts.Version != "" || opts.ProductName != "" || opts.FileDescription != "" || opts.CompanyName != "" || opts.Copyright != "" || opts.InternalName != "" {
		vi, err := buildVersionInfo(opts, outputPath)
		if err != nil {
			return err
		}
		rs.SetVersionInfo(*vi)
	}

	if _, err := srcFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind build stub: %w", err)
	}
	outFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create resource-patched stub: %w", err)
	}
	defer outFile.Close()

	if err := rs.WriteToEXE(outFile, srcFile, winres.ForceCheckSum()); err != nil {
		return fmt.Errorf("write executable resources: %w", err)
	}
	return nil
}

func loadBuildIcon(path string) (*winres.Icon, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open icon: %w", err)
	}
	defer file.Close()

	if strings.EqualFold(filepath.Ext(path), ".ico") {
		icon, err := winres.LoadICO(file)
		if err != nil {
			return nil, fmt.Errorf("load ico: %w", err)
		}
		return icon, nil
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode icon image: %w", err)
	}
	icon, err := winres.NewIconFromResizedImage(img, nil)
	if err != nil {
		return nil, fmt.Errorf("build icon from image: %w", err)
	}
	return icon, nil
}

func buildVersionInfo(opts buildOptions, outputPath string) (*version.Info, error) {
	vi := &version.Info{}
	versionText := opts.Version
	if versionText == "" {
		versionText = "0.0.0.0"
	}
	vi.SetFileVersion(versionText)
	vi.SetProductVersion(versionText)

	productName := opts.ProductName
	if productName == "" {
		productName = strings.TrimSuffix(filepath.Base(outputPath), filepath.Ext(outputPath))
	}
	fileDescription := opts.FileDescription
	if fileDescription == "" {
		fileDescription = productName
	}
	internalName := opts.InternalName
	if internalName == "" {
		internalName = productName
	}
	originalFilename := filepath.Base(outputPath)

	for key, value := range map[string]string{
		version.ProductName:      productName,
		version.FileDescription:  fileDescription,
		version.InternalName:     internalName,
		version.OriginalFilename: originalFilename,
		version.ProductVersion:   versionText,
		version.FileVersion:      versionText,
	} {
		if err := vi.Set(version.LangDefault, key, value); err != nil {
			return nil, fmt.Errorf("set version info %s: %w", key, err)
		}
	}
	if opts.CompanyName != "" {
		if err := vi.Set(version.LangDefault, version.CompanyName, opts.CompanyName); err != nil {
			return nil, fmt.Errorf("set version info %s: %w", version.CompanyName, err)
		}
	}
	if opts.Copyright != "" {
		if err := vi.Set(version.LangDefault, version.LegalCopyright, opts.Copyright); err != nil {
			return nil, fmt.Errorf("set version info %s: %w", version.LegalCopyright, err)
		}
	}
	return vi, nil
}

func appendBundleToExecutable(stubPath string, outputPath string, bundleData []byte) error {
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	in, err := os.Open(stubPath)
	if err != nil {
		return fmt.Errorf("open executable stub: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create output executable: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy executable stub: %w", err)
	}
	if _, err := out.Write(bundleData); err != nil {
		return fmt.Errorf("append bundle payload: %w", err)
	}

	footer := make([]byte, len(embeddedBundleMagic)+8)
	copy(footer, []byte(embeddedBundleMagic))
	binary.LittleEndian.PutUint64(footer[len(embeddedBundleMagic):], uint64(len(bundleData)))
	if _, err := out.Write(footer); err != nil {
		return fmt.Errorf("append bundle footer: %w", err)
	}
	return nil
}

func readEmbeddedBundle(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open executable: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat executable: %w", err)
	}
	footerSize := int64(len(embeddedBundleMagic) + 8)
	if info.Size() < footerSize {
		return nil, nil
	}

	if _, err := file.Seek(-footerSize, io.SeekEnd); err != nil {
		return nil, fmt.Errorf("seek bundle footer: %w", err)
	}
	footer := make([]byte, footerSize)
	if _, err := io.ReadFull(file, footer); err != nil {
		return nil, fmt.Errorf("read bundle footer: %w", err)
	}
	if string(footer[:len(embeddedBundleMagic)]) != embeddedBundleMagic {
		return nil, nil
	}

	bundleSize := int64(binary.LittleEndian.Uint64(footer[len(embeddedBundleMagic):]))
	if bundleSize <= 0 || bundleSize > info.Size()-footerSize {
		return nil, fmt.Errorf("invalid embedded bundle size")
	}

	if _, err := file.Seek(-(footerSize + bundleSize), io.SeekEnd); err != nil {
		return nil, fmt.Errorf("seek bundle payload: %w", err)
	}
	data := make([]byte, bundleSize)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, fmt.Errorf("read bundle payload: %w", err)
	}
	return data, nil
}

func runEmbeddedBundleIfPresent() (bool, error) {
	execPath, err := os.Executable()
	if err != nil {
		return false, err
	}
	data, err := readEmbeddedBundle(execPath)
	if err != nil {
		return false, err
	}
	if len(data) == 0 {
		return false, nil
	}

	archive, err := api.LoadBundle(data)
	if err != nil {
		return true, err
	}
	rt := api.NewRuntime()
	defer func() {
		_ = rt.Close()
	}()
	// 嵌入式可执行文件需要把宿主进程参数透传给 bundle 内的 argv()/CLI 框架。
	rt.SetScriptArgs(os.Args[1:])
	_, err = rt.RunBundleArchive(execPath, archive)
	return true, err
}
