package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runExtract(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: icoo extract <bundle|executable> [output]")
	}

	target := args[0]
	data, err := loadBundleDataForExtract(target)
	if err != nil {
		return err
	}
	outputPath, err := resolveExtractOutput(target, optionalArg(args, 1))
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write extracted bundle: %w", err)
	}

	fmt.Printf("extracted bundle: %s\n", outputPath)
	return nil
}

func loadBundleDataForExtract(path string) ([]byte, error) {
	if isArchivePath(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read bundle: %w", err)
		}
		if _, _, err := loadArchiveForInspect(path); err != nil {
			return nil, err
		}
		return data, nil
	}

	data, err := readEmbeddedBundle(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no embedded bundle found: %s", path)
	}
	if _, _, err := loadArchiveForInspect(path); err != nil {
		return nil, err
	}
	return data, nil
}

func resolveExtractOutput(target string, output string) (string, error) {
	if strings.TrimSpace(output) != "" {
		return filepath.Abs(output)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve extract target: %w", err)
	}
	if isArchivePath(absTarget) {
		base := strings.TrimSuffix(absTarget, filepath.Ext(absTarget))
		return base + ".extracted" + filepath.Ext(absTarget), nil
	}
	base := strings.TrimSuffix(absTarget, filepath.Ext(absTarget))
	return base + bundleFileExt, nil
}
