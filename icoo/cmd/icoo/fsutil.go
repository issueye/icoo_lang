package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "" || parent == "." {
		return nil
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	return nil
}
