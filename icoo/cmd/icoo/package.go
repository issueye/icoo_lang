package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"icoo_lang/pkg/api"
)

type packageOptions struct {
	Target     string
	Output     string
	Name       string
	Version    string
	ExportPath string
}

func runPackage(args []string) error {
	opts, err := parsePackageArgs(args)
	if err != nil {
		return err
	}
	if err := loadPackageConfigInto(&opts); err != nil {
		return err
	}

	archive, outputPath, err := buildArchive(buildArchiveOptions{
		Target:         opts.Target,
		Output:         opts.Output,
		ArchiveExt:     packageFileExt,
		Kind:           api.ArchiveKindPackage,
		PackageName:    opts.Name,
		PackageVersion: opts.Version,
		ExportPath:     opts.ExportPath,
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("encode package: %w", err)
	}
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write package: %w", err)
	}

	fmt.Printf("packaged project: %s\n", outputPath)
	return nil
}

func parsePackageArgs(args []string) (packageOptions, error) {
	opts := packageOptions{}
	positionals := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--name":
			i++
			if i >= len(args) {
				return packageOptions{}, fmt.Errorf("usage: icoo package <file|dir> [output] [--name value] [--version value] [--export path]")
			}
			opts.Name = args[i]
		case strings.HasPrefix(arg, "--name="):
			opts.Name = strings.TrimPrefix(arg, "--name=")
		case arg == "--version":
			i++
			if i >= len(args) {
				return packageOptions{}, fmt.Errorf("usage: icoo package <file|dir> [output] [--name value] [--version value] [--export path]")
			}
			opts.Version = args[i]
		case strings.HasPrefix(arg, "--version="):
			opts.Version = strings.TrimPrefix(arg, "--version=")
		case arg == "--export":
			i++
			if i >= len(args) {
				return packageOptions{}, fmt.Errorf("usage: icoo package <file|dir> [output] [--name value] [--version value] [--export path]")
			}
			opts.ExportPath = args[i]
		case strings.HasPrefix(arg, "--export="):
			opts.ExportPath = strings.TrimPrefix(arg, "--export=")
		case strings.HasPrefix(arg, "--"):
			return packageOptions{}, fmt.Errorf("unknown option: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) < 1 || len(positionals) > 2 {
		return packageOptions{}, fmt.Errorf("usage: icoo package <file|dir> [output] [--name value] [--version value] [--export path]")
	}
	opts.Target = positionals[0]
	if len(positionals) == 2 {
		opts.Output = positionals[1]
	}
	opts.Name = strings.TrimSpace(opts.Name)
	opts.Version = strings.TrimSpace(opts.Version)
	opts.ExportPath = strings.TrimSpace(opts.ExportPath)
	return opts, nil
}

func loadPackageConfigInto(opts *packageOptions) error {
	if opts == nil {
		return nil
	}
	resolved, ok, err := tryLoadPackageTarget(opts.Target)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	manifest, err := loadPackageManifest(resolved.Root)
	if err != nil {
		return err
	}
	if opts.Name == "" {
		opts.Name = strings.TrimSpace(manifest.Name)
	}
	if opts.Version == "" {
		opts.Version = strings.TrimSpace(manifest.Version)
	}
	if opts.ExportPath == "" {
		opts.ExportPath = strings.TrimSpace(manifest.Export)
	}
	return nil
}
