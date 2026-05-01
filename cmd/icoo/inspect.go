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
	if strings.EqualFold(filepath.Ext(path), bundleFileExt) {
		archive, err := api.LoadBundleFile(path)
		if err != nil {
			return nil, "", err
		}
		return archive, "bundle", nil
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
	fmt.Printf("entry: %s\n", archive.Entry)
	if archive.EntryFunction != "" {
		fmt.Printf("entry_function: %s\n", archive.EntryFunction)
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
