package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type initSubpackageOptions struct {
	Target     string
	ParentName string
	Version    string
	RootAlias  string
}

func runInitSubpackage(args []string) error {
	opts, err := parseInitSubpackageArgs(args)
	if err != nil {
		return err
	}

	name := buildSubpackageName(opts.ParentName, opts.Target)
	return runInitPackage([]string{
		opts.Target,
		"--name", name,
		"--version", opts.Version,
		"--root-alias", opts.RootAlias,
	})
}

func parseInitSubpackageArgs(args []string) (initSubpackageOptions, error) {
	opts := initSubpackageOptions{
		Version:   defaultPackageVersion,
		RootAlias: "@",
	}

	positionals := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--parent":
			i++
			if i >= len(args) {
				return initSubpackageOptions{}, errors.New("usage: icoo init-subpkg <dir> --parent value [--version value] [--root-alias name]")
			}
			opts.ParentName = args[i]
		case strings.HasPrefix(arg, "--parent="):
			opts.ParentName = strings.TrimPrefix(arg, "--parent=")
		case arg == "--version":
			i++
			if i >= len(args) {
				return initSubpackageOptions{}, errors.New("usage: icoo init-subpkg <dir> --parent value [--version value] [--root-alias name]")
			}
			opts.Version = args[i]
		case strings.HasPrefix(arg, "--version="):
			opts.Version = strings.TrimPrefix(arg, "--version=")
		case arg == "--root-alias":
			i++
			if i >= len(args) {
				return initSubpackageOptions{}, errors.New("usage: icoo init-subpkg <dir> --parent value [--version value] [--root-alias name]")
			}
			opts.RootAlias = args[i]
		case strings.HasPrefix(arg, "--root-alias="):
			opts.RootAlias = strings.TrimPrefix(arg, "--root-alias=")
		case strings.HasPrefix(arg, "--"):
			return initSubpackageOptions{}, fmt.Errorf("unknown option: %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) != 1 {
		return initSubpackageOptions{}, errors.New("usage: icoo init-subpkg <dir> --parent value [--version value] [--root-alias name]")
	}
	if strings.TrimSpace(opts.ParentName) == "" {
		return initSubpackageOptions{}, errors.New("subpackage parent name is required")
	}
	rootAlias, err := normalizeProjectRootAlias(opts.RootAlias)
	if err != nil {
		return initSubpackageOptions{}, err
	}

	opts.Target = positionals[0]
	opts.ParentName = strings.Trim(strings.TrimSpace(opts.ParentName), "/")
	opts.Version = strings.TrimSpace(opts.Version)
	opts.RootAlias = rootAlias
	return opts, nil
}

func buildSubpackageName(parentName string, target string) string {
	parentName = strings.Trim(strings.TrimSpace(parentName), "/")
	normalized := filepath.ToSlash(filepath.Clean(strings.TrimSpace(target)))
	if index := strings.Index(normalized, "/pkg/"); index >= 0 {
		normalized = normalized[index+1:]
	} else if strings.HasPrefix(normalized, "pkg/") {
		normalized = "pkg/" + strings.TrimPrefix(normalized, "pkg/")
	} else {
		normalized = filepath.ToSlash(filepath.Base(normalized))
	}
	normalized = strings.Trim(normalized, "/")
	return parentName + "/" + normalized
}
