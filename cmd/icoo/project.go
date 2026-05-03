package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"

	"icoo_lang/pkg/api"
)

const (
	defaultProjectEntry         = "src/main.ic"
	defaultProjectEntryFunction = "main"
	projectConfigFileName       = "project.toml"
)

type projectFile struct {
	Project projectConfig `toml:"project"`
}

type projectConfig struct {
	Name          string `toml:"name"`
	Entry         string `toml:"entry"`
	EntryFunction string `toml:"entry_function"`
	RootAlias     string `toml:"root_alias"`
}

type initOptions struct {
	Target        string
	Entry         string
	EntryFunction string
	RootAlias     string
}

type resolvedProject struct {
	Root          string
	ConfigPath    string
	EntryPath     string
	EntryDisplay  string
	EntryFunction string
	RootAlias     string
}

func runInit(args []string) error {
	opts, err := parseInitArgs(args)
	if err != nil {
		return err
	}

	root, err := filepath.Abs(opts.Target)
	if err != nil {
		return fmt.Errorf("resolve project dir: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}

	configPath := filepath.Join(root, projectConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("project already exists: %s", configPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat project config: %w", err)
	}

	entryPath, err := resolveProjectEntryPath(root, opts.Entry)
	if err != nil {
		return err
	}
	if _, err := os.Stat(entryPath); err == nil {
		return fmt.Errorf("entry file already exists: %s", entryPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat entry file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return fmt.Errorf("create entry dir: %w", err)
	}

	name := filepath.Base(root)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "icoo_app"
	}
	cfg := projectFile{Project: projectConfig{
		Name:          name,
		Entry:         opts.Entry,
		EntryFunction: opts.EntryFunction,
		RootAlias:     opts.RootAlias,
	}}
	encoded, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode project config: %w", err)
	}
	if err := os.WriteFile(configPath, encoded, 0o644); err != nil {
		return fmt.Errorf("write project config: %w", err)
	}

	source := strings.Join([]string{
		"import std.io as io",
		"",
		fmt.Sprintf("fn %s() {", opts.EntryFunction),
		"  io.println(\"Hello from icoo\")",
		"}",
		"",
	}, "\n")
	if err := os.WriteFile(entryPath, []byte(source), 0o644); err != nil {
		return fmt.Errorf("write entry file: %w", err)
	}

	fmt.Printf("initialized project: %s\n", root)
	return nil
}

func parseInitArgs(args []string) (initOptions, error) {
	opts := initOptions{
		Target:        ".",
		Entry:         defaultProjectEntry,
		EntryFunction: defaultProjectEntryFunction,
	}

	positionals := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--entry":
			i++
			if i >= len(args) {
				return initOptions{}, errors.New("usage: icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]")
			}
			opts.Entry = args[i]
		case strings.HasPrefix(arg, "--entry="):
			opts.Entry = strings.TrimPrefix(arg, "--entry=")
		case arg == "--entry-fn":
			i++
			if i >= len(args) {
				return initOptions{}, errors.New("usage: icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]")
			}
			opts.EntryFunction = args[i]
		case strings.HasPrefix(arg, "--entry-fn="):
			opts.EntryFunction = strings.TrimPrefix(arg, "--entry-fn=")
		case arg == "--root-alias":
			i++
			if i >= len(args) {
				return initOptions{}, errors.New("usage: icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]")
			}
			opts.RootAlias = args[i]
		case strings.HasPrefix(arg, "--root-alias="):
			opts.RootAlias = strings.TrimPrefix(arg, "--root-alias=")
		case strings.HasPrefix(arg, "--"):
			return initOptions{}, fmt.Errorf("unknown option: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) > 1 {
		return initOptions{}, errors.New("usage: icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]")
	}
	if len(positionals) == 1 {
		opts.Target = positionals[0]
	}

	entry, err := normalizeProjectEntry(opts.Entry)
	if err != nil {
		return initOptions{}, err
	}
	entryFn := strings.TrimSpace(opts.EntryFunction)
	if entryFn == "" {
		return initOptions{}, errors.New("project entry function is required")
	}
	if strings.ContainsAny(entryFn, " \t\r\n") {
		return initOptions{}, fmt.Errorf("project entry function must not contain whitespace: %s", entryFn)
	}
	rootAlias, err := normalizeProjectRootAlias(opts.RootAlias)
	if err != nil {
		return initOptions{}, err
	}

	opts.Entry = entry
	opts.EntryFunction = entryFn
	opts.RootAlias = rootAlias
	return opts, nil
}

func runCheckPath(path string) error {
	if strings.EqualFold(filepath.Ext(path), bundleFileExt) {
		rt := api.NewRuntime()
		defer func() {
			_ = rt.Close()
		}()
		errs := rt.CheckBundleFile(path)
		if len(errs) > 0 {
			for _, checkErr := range errs {
				fmt.Fprintln(os.Stderr, checkErr)
			}
			return errors.New("check failed")
		}
		fmt.Printf("ok: %s\n", path)
		return nil
	}

	resolved, err := resolveRunTarget(path)
	if err != nil {
		return err
	}

	rt := api.NewRuntime()
	defer func() {
		_ = rt.Close()
	}()
	rt.SetProjectRoot(resolved.Root, resolved.RootAlias)
	errs := rt.CheckFile(resolved.EntryPath)
	if len(errs) > 0 {
		for _, checkErr := range errs {
			fmt.Fprintln(os.Stderr, checkErr)
		}
		return errors.New("check failed")
	}

	fmt.Printf("ok: %s\n", resolved.EntryDisplay)
	return nil
}

func runProjectPath(path string) error {
	if strings.EqualFold(filepath.Ext(path), bundleFileExt) {
		rt := api.NewRuntime()
		defer func() {
			_ = rt.Close()
		}()
		_, err := rt.RunBundleFile(path)
		return err
	}

	resolved, err := resolveRunTarget(path)
	if err != nil {
		return err
	}

	rt := api.NewRuntime()
	defer func() {
		_ = rt.Close()
	}()
	rt.SetProjectRoot(resolved.Root, resolved.RootAlias)
	if _, err := rt.RunFile(resolved.EntryPath); err != nil {
		return err
	}
	if resolved.EntryFunction != "" {
		if _, err := rt.InvokeGlobal(resolved.EntryFunction); err != nil {
			return err
		}
	}
	return nil
}

func resolveRunTarget(path string) (resolvedProject, error) {
	info, err := os.Stat(path)
	if err != nil {
		return resolvedProject{}, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		entryPath, err := filepath.Abs(path)
		if err != nil {
			return resolvedProject{}, fmt.Errorf("resolve path: %w", err)
		}

		resolved := resolvedProject{
			EntryPath:    entryPath,
			EntryDisplay: path,
		}
		projectRoot, ok, err := findProjectRoot(filepath.Dir(entryPath))
		if err != nil {
			return resolvedProject{}, err
		}
		if ok {
			project, err := loadProject(projectRoot)
			if err != nil {
				return resolvedProject{}, err
			}
			resolved.Root = project.Root
			resolved.ConfigPath = project.ConfigPath
			resolved.RootAlias = project.RootAlias
		}
		return resolved, nil
	}
	return loadProject(path)
}

func loadProject(root string) (resolvedProject, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return resolvedProject{}, fmt.Errorf("resolve project dir: %w", err)
	}
	configPath := filepath.Join(absRoot, projectConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return resolvedProject{}, fmt.Errorf("read project config: %w", err)
	}

	var file projectFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return resolvedProject{}, fmt.Errorf("parse project config: %w", err)
	}
	entry, err := normalizeProjectEntry(file.Project.Entry)
	if err != nil {
		return resolvedProject{}, err
	}
	entryPath, err := resolveProjectEntryPath(absRoot, entry)
	if err != nil {
		return resolvedProject{}, err
	}
	if _, err := os.Stat(entryPath); err != nil {
		return resolvedProject{}, fmt.Errorf("stat entry file: %w", err)
	}

	entryFn := strings.TrimSpace(file.Project.EntryFunction)
	if entryFn == "" {
		entryFn = defaultProjectEntryFunction
	}
	rootAlias, err := normalizeProjectRootAlias(file.Project.RootAlias)
	if err != nil {
		return resolvedProject{}, err
	}

	return resolvedProject{
		Root:          absRoot,
		ConfigPath:    configPath,
		EntryPath:     entryPath,
		EntryDisplay:  entryPath,
		EntryFunction: entryFn,
		RootAlias:     rootAlias,
	}, nil
}

func normalizeProjectEntry(entry string) (string, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return "", errors.New("project entry is required")
	}
	entry = filepath.ToSlash(entry)
	if filepath.IsAbs(entry) {
		return "", fmt.Errorf("project entry must be relative: %s", entry)
	}
	cleaned := pathClean(entry)
	if cleaned == "." || cleaned == "" {
		return "", errors.New("project entry is required")
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("project entry must stay within project root: %s", entry)
	}
	return cleaned, nil
}

func normalizeProjectRootAlias(alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", nil
	}
	if alias == "std" {
		return "", errors.New("project root alias cannot be std")
	}
	if strings.ContainsAny(alias, "/\\. \t\r\n") {
		return "", fmt.Errorf("project root alias must be a single identifier: %s", alias)
	}
	for i, r := range alias {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return "", fmt.Errorf("project root alias must start with a letter or underscore: %s", alias)
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return "", fmt.Errorf("project root alias must be a single identifier: %s", alias)
		}
	}
	return alias, nil
}

func resolveProjectEntryPath(root, entry string) (string, error) {
	entryPath := filepath.Join(root, filepath.FromSlash(entry))
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve entry path: %w", err)
	}
	rel, err := filepath.Rel(root, absEntry)
	if err != nil {
		return "", fmt.Errorf("resolve entry relation: %w", err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("project entry must stay within project root: %s", entry)
	}
	return absEntry, nil
}

func pathClean(value string) string {
	parts := strings.Split(value, "/")
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			} else {
				stack = append(stack, "..")
			}
		default:
			stack = append(stack, part)
		}
	}
	if len(stack) == 0 {
		return "."
	}
	return strings.Join(stack, "/")
}

func findProjectRoot(start string) (string, bool, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false, fmt.Errorf("resolve project search path: %w", err)
	}

	for {
		configPath := filepath.Join(current, projectConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return current, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("stat project config: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}
