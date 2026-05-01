package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
}

type resolvedProject struct {
	Root          string
	ConfigPath    string
	EntryPath     string
	EntryDisplay  string
	EntryFunction string
}

func runInit(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: icoo init [dir]")
	}

	target := "."
	if len(args) == 1 {
		target = args[0]
	}

	root, err := filepath.Abs(target)
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

	entry, err := normalizeProjectEntry(defaultProjectEntry)
	if err != nil {
		return err
	}
	entryPath, err := resolveProjectEntryPath(root, entry)
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
		Entry:         entry,
		EntryFunction: defaultProjectEntryFunction,
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
		"fn main() {",
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

func runCheckPath(path string) error {
	resolved, err := resolveRunTarget(path)
	if err != nil {
		return err
	}

	rt := api.NewRuntime()
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
	resolved, err := resolveRunTarget(path)
	if err != nil {
		return err
	}

	rt := api.NewRuntime()
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
		return resolvedProject{EntryPath: entryPath, EntryDisplay: path}, nil
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

	return resolvedProject{
		Root:          absRoot,
		ConfigPath:    configPath,
		EntryPath:     entryPath,
		EntryDisplay:  entryPath,
		EntryFunction: entryFn,
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
