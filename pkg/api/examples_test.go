package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

		t.Run(strings.TrimSuffix(filepath.Base(path), ".ic"), func(t *testing.T) {
			rt := NewRuntime()
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
