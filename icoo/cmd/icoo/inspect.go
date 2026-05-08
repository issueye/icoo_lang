package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"icoo_lang/pkg/api"
)

func runInspect(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: icoo inspect <bundle|executable>")
	}

	target := args[0]
	archive, kind, err := loadArchiveForInspect(target)
	if err != nil {
		return err
	}
	printArchiveSummary(target, kind, archive)
	return nil
}

func loadArchiveForInspect(path string) (*api.BundleArchive, string, error) {
	if isArchivePath(path) {
		archive, err := api.LoadBundleFile(path)
		if err != nil {
			return nil, "", err
		}
		kind := "bundle"
		if strings.EqualFold(filepath.Ext(path), packageFileExt) {
			kind = "package"
		}
		return archive, kind, nil
	}

	data, err := readEmbeddedBundle(path)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("no embedded bundle found: %s", path)
	}
	archive, err := api.LoadBundle(data)
	if err != nil {
		return nil, "", err
	}
	return archive, "executable", nil
}

func printArchiveSummary(path string, kind string, archive *api.BundleArchive) {
	moduleNames := make([]string, 0, len(archive.Modules))
	for name := range archive.Modules {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	fmt.Printf("type: %s\n", kind)
	fmt.Printf("path: %s\n", path)
	fmt.Printf("archive_kind: %s\n", archive.Kind)
	fmt.Printf("entry: %s\n", archive.Entry)
	if archive.Export != "" {
		fmt.Printf("export: %s\n", archive.Export)
	}
	if archive.EntryFunction != "" {
		fmt.Printf("entry_function: %s\n", archive.EntryFunction)
	}
	if archive.PackageName != "" {
		fmt.Printf("package_name: %s\n", archive.PackageName)
	}
	if archive.PackageVersion != "" {
		fmt.Printf("package_version: %s\n", archive.PackageVersion)
	}
	if archive.RootAlias != "" {
		fmt.Printf("root_alias: %s\n", archive.RootAlias)
	}
	if archive.ProjectRoot != "" {
		fmt.Printf("project_root: %s\n", archive.ProjectRoot)
	}
	fmt.Printf("module_count: %d\n", len(moduleNames))
	fmt.Println("modules:")
	for _, name := range moduleNames {
		fmt.Printf("  - %s\n", name)
	}
}
