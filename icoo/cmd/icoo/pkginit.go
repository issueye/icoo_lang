package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	defaultPackageVersion       = "0.1.0"
	defaultPackageEntry         = "src/main.ic"
	defaultPackageEntryFunction = "main"
	defaultPackageExport        = "lib.ic"
	packageConfigFileName       = "pkg.toml"
)

type packageFile struct {
	Package packageConfig `toml:"package"`
}

type packageConfig struct {
	Name          string `toml:"name"`
	Version       string `toml:"version"`
	Entry         string `toml:"entry"`
	EntryFunction string `toml:"entry_function"`
	Export        string `toml:"export"`
	RootAlias     string `toml:"root_alias"`
}

type initPackageOptions struct {
	Target        string
	Name          string
	Version       string
	Entry         string
	EntryFunction string
	Export        string
	RootAlias     string
}

func runInitPackage(args []string) error {
	opts, err := parseInitPackageArgs(args)
	if err != nil {
		return err
	}

	root, err := filepath.Abs(opts.Target)
	if err != nil {
		return fmt.Errorf("resolve package dir: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create package dir: %w", err)
	}

	configPath := filepath.Join(root, packageConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("package already exists: %s", configPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat package config: %w", err)
	}

	entryPath, err := resolveProjectEntryPath(root, opts.Entry)
	if err != nil {
		return err
	}
	exportPath, err := resolveProjectEntryPath(root, opts.Export)
	if err != nil {
		return err
	}
	for _, path := range []string{entryPath, exportPath} {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("package scaffold file already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat package scaffold file: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return fmt.Errorf("create entry dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(exportPath), 0o755); err != nil {
		return fmt.Errorf("create export dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "examples"), 0o755); err != nil {
		return fmt.Errorf("create examples dir: %w", err)
	}

	cfg := packageFile{Package: packageConfig{
		Name:          opts.Name,
		Version:       opts.Version,
		Entry:         opts.Entry,
		EntryFunction: opts.EntryFunction,
		Export:        opts.Export,
		RootAlias:     opts.RootAlias,
	}}
	encoded, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode package config: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write package config: %w", err)
	}

	entrySource := strings.Join([]string{
		"export fn hello(name) {",
		"  return \"hello \" + name",
		"}",
		"",
		fmt.Sprintf("fn %s() {", opts.EntryFunction),
		"  return hello(\"icoo\")",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(entrySource), 0o644); err != nil {
		return fmt.Errorf("write package entry file: %w", err)
	}

	return writeInitPackageExport(root, exportPath, opts)
}

func writeInitPackageExport(root string, exportPath string, opts initPackageOptions) error {
	entryImport := opts.RootAlias + "/" + strings.TrimPrefix(opts.Entry, "/")
	if opts.RootAlias == "@" {
		entryImport = "@/" + strings.TrimPrefix(opts.Entry, "/")
	}
	exportSource := strings.Join([]string{
		fmt.Sprintf("import %q as mainModule", entryImport),
		"",
		"export {",
		"  hello: mainModule.hello",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(exportPath, []byte(exportSource), 0o644); err != nil {
		return fmt.Errorf("write package export file: %w", err)
	}

	fmt.Printf("initialized package: %s\n", root)
	return nil
}

func parseInitPackageArgs(args []string) (initPackageOptions, error) {
	opts := initPackageOptions{
		Target:        ".",
		Version:       defaultPackageVersion,
		Entry:         defaultPackageEntry,
		EntryFunction: defaultPackageEntryFunction,
		Export:        defaultPackageExport,
		RootAlias:     "@",
	}

	positionals := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--name":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.Name = args[i]
		case strings.HasPrefix(arg, "--name="):
			opts.Name = strings.TrimPrefix(arg, "--name=")
		case arg == "--version":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.Version = args[i]
		case strings.HasPrefix(arg, "--version="):
			opts.Version = strings.TrimPrefix(arg, "--version=")
		case arg == "--entry":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.Entry = args[i]
		case strings.HasPrefix(arg, "--entry="):
			opts.Entry = strings.TrimPrefix(arg, "--entry=")
		case arg == "--entry-fn":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.EntryFunction = args[i]
		case strings.HasPrefix(arg, "--entry-fn="):
			opts.EntryFunction = strings.TrimPrefix(arg, "--entry-fn=")
		case arg == "--export":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.Export = args[i]
		case strings.HasPrefix(arg, "--export="):
			opts.Export = strings.TrimPrefix(arg, "--export=")
		case arg == "--root-alias":
			i++
			if i >= len(args) {
				return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
			}
			opts.RootAlias = args[i]
		case strings.HasPrefix(arg, "--root-alias="):
			opts.RootAlias = strings.TrimPrefix(arg, "--root-alias=")
		case strings.HasPrefix(arg, "--"):
			return initPackageOptions{}, fmt.Errorf("unknown option: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) > 1 {
		return initPackageOptions{}, errors.New("usage: icoo init-pkg [dir] [--name value] [--version value] [--entry path] [--entry-fn name] [--export path] [--root-alias name]")
	}
	if len(positionals) == 1 {
		opts.Target = positionals[0]
	}

	if strings.TrimSpace(opts.Name) == "" {
		opts.Name = filepath.Base(opts.Target)
		if opts.Name == "." || opts.Name == string(filepath.Separator) || opts.Name == "" {
			opts.Name = "icoo_pkg"
		}
	}
	entry, err := normalizeProjectEntry(opts.Entry)
	if err != nil {
		return initPackageOptions{}, err
	}
	exportPath, err := normalizeProjectEntry(opts.Export)
	if err != nil {
		return initPackageOptions{}, err
	}
	entryFn := strings.TrimSpace(opts.EntryFunction)
	if entryFn == "" {
		return initPackageOptions{}, errors.New("package entry function is required")
	}
	if strings.ContainsAny(entryFn, " \t\r\n") {
		return initPackageOptions{}, fmt.Errorf("package entry function must not contain whitespace: %s", entryFn)
	}
	rootAlias, err := normalizeProjectRootAlias(opts.RootAlias)
	if err != nil {
		return initPackageOptions{}, err
	}
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Version = strings.TrimSpace(opts.Version)
	opts.Entry = entry
	opts.EntryFunction = entryFn
	opts.Export = exportPath
	opts.RootAlias = rootAlias
	return opts, nil
}

func tryLoadPackageTarget(path string) (resolvedProject, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return resolvedProject{}, false, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return resolvedProject{}, false, nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return resolvedProject{}, false, fmt.Errorf("resolve package dir: %w", err)
	}
	configPath := filepath.Join(absPath, packageConfigFileName)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return resolvedProject{}, false, nil
	} else if err != nil {
		return resolvedProject{}, false, fmt.Errorf("stat package config: %w", err)
	}
	resolved, err := loadPackageProject(absPath)
	return resolved, true, err
}

func loadPackageProject(root string) (resolvedProject, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return resolvedProject{}, fmt.Errorf("resolve package dir: %w", err)
	}
	manifest, err := loadPackageManifest(absRoot)
	if err != nil {
		return resolvedProject{}, err
	}
	entry, err := normalizeProjectEntry(manifest.Entry)
	if err != nil {
		return resolvedProject{}, err
	}
	entryPath, err := resolveProjectEntryPath(absRoot, entry)
	if err != nil {
		return resolvedProject{}, err
	}
	if _, err := os.Stat(entryPath); err != nil {
		return resolvedProject{}, fmt.Errorf("stat package entry file: %w", err)
	}
	entryFn := strings.TrimSpace(manifest.EntryFunction)
	if entryFn == "" {
		entryFn = defaultPackageEntryFunction
	}
	rootAlias, err := normalizeProjectRootAlias(manifest.RootAlias)
	if err != nil {
		return resolvedProject{}, err
	}
	return resolvedProject{
		Root:          absRoot,
		ConfigPath:    filepath.Join(absRoot, packageConfigFileName),
		EntryPath:     entryPath,
		EntryDisplay:  entryPath,
		EntryFunction: entryFn,
		RootAlias:     rootAlias,
	}, nil
}

func loadPackageManifest(root string) (packageConfig, error) {
	data, err := os.ReadFile(filepath.Join(root, packageConfigFileName))
	if err != nil {
		return packageConfig{}, fmt.Errorf("read package config: %w", err)
	}
	var file packageFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return packageConfig{}, fmt.Errorf("parse package config: %w", err)
	}
	cfg := file.Package
	if strings.TrimSpace(cfg.Entry) == "" {
		cfg.Entry = defaultPackageEntry
	}
	if strings.TrimSpace(cfg.EntryFunction) == "" {
		cfg.EntryFunction = defaultPackageEntryFunction
	}
	if strings.TrimSpace(cfg.Export) == "" {
		cfg.Export = defaultPackageExport
	}
	if strings.TrimSpace(cfg.Version) == "" {
		cfg.Version = defaultPackageVersion
	}
	if strings.TrimSpace(cfg.RootAlias) == "" {
		cfg.RootAlias = "@"
	}
	return cfg, nil
}
