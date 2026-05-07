package api

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestExamplesRun(t *testing.T) {
	root := filepath.Join("..", "..", "examples")

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "lib" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".ic" {
			return nil
		}
		slashPath := filepath.ToSlash(path)
		if slashPath == "../../examples/proxy/app.ic" || slashPath == "../../examples/proxy/mock_upstream.ic" {
			return nil
		}

		absPath, _ := filepath.Abs(path)
		t.Run(strings.TrimSuffix(filepath.Base(path), ".ic"), func(t *testing.T) {
			rt := NewRuntime()
			setProjectRootForPath(rt, absPath)
			if _, err := rt.RunFile(path); err != nil {
				t.Fatalf("run example %s: %v", path, err)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
}

type projectFile struct {
	Project struct {
		RootAlias string `toml:"root_alias"`
	} `toml:"project"`
}

func findProjectRoot(start string) (string, string, error) {
	current := start
	for {
		configPath := filepath.Join(current, "project.toml")
		data, err := os.ReadFile(configPath)
		if err == nil {
			var pf projectFile
			if err := toml.Unmarshal(data, &pf); err != nil {
				return "", "", err
			}
			return current, pf.Project.RootAlias, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", "", nil
		}
		current = parent
	}
}

func setProjectRootForPath(rt *Runtime, absPath string) {
	dir := filepath.Dir(absPath)
	root, alias, err := findProjectRoot(dir)
	if err != nil || root == "" {
		return
	}
	rt.SetProjectRoot(root, alias)
}