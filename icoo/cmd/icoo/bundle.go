package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
	"icoo_lang/pkg/api"
)

func runBundle(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: icoo bundle <file|dir> [output]")
	}

	archive, outputPath, err := buildArchive(buildArchiveOptions{
		Target:     args[0],
		Output:     optionalArg(args, 1),
		ArchiveExt: bundleFileExt,
		Kind:       api.ArchiveKindApplication,
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bundle: %w", err)
	}
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	fmt.Printf("bundled project: %s\n", outputPath)
	return nil
}

func optionalArg(args []string, index int) string {
	if index >= 0 && index < len(args) {
		return args[index]
	}
	return ""
}

type buildArchiveOptions struct {
	Target         string
	Output         string
	ArchiveExt     string
	Kind           string
	PackageName    string
	PackageVersion string
	ExportPath     string
}

func buildArchive(opts buildArchiveOptions) (*api.BundleArchive, string, error) {
	resolved, err := resolveArchiveTarget(opts)
	if err != nil {
		return nil, "", err
	}

	graph, err := collectBundleSources(resolved, opts.ExportPath)
	if err != nil {
		return nil, "", err
	}

	baseRoot := resolved.Root
	if baseRoot == "" {
		baseRoot, err = commonAncestor(graph.Paths)
		if err != nil {
			return nil, "", err
		}
	}
	entryRel, err := relBundlePath(baseRoot, resolved.EntryPath)
	if err != nil {
		return nil, "", err
	}

	projectRootRel := ""
	if resolved.Root != "" {
		projectRootRel, err = relBundlePath(baseRoot, resolved.Root)
		if err != nil {
			return nil, "", err
		}
		if projectRootRel == "." {
			projectRootRel = ""
		}
	}

	modules := make(map[string]string, len(graph.Sources))
	for _, absPath := range graph.Paths {
		relPath, err := relBundlePath(baseRoot, absPath)
		if err != nil {
			return nil, "", err
		}
		modules[relPath] = graph.Sources[absPath]
	}
	packages := make(map[string]*api.BundleArchive, len(graph.Packages))
	for absPath, packageArchive := range graph.Packages {
		relPath, err := relBundlePath(baseRoot, absPath)
		if err != nil {
			return nil, "", err
		}
		packages[relPath] = packageArchive
	}

	exportRel := ""
	if graph.ExportPath != "" {
		exportRel, err = relBundlePath(baseRoot, graph.ExportPath)
		if err != nil {
			return nil, "", err
		}
	}

	archive := &api.BundleArchive{
		Version:        api.BundleVersion,
		Kind:           opts.Kind,
		Entry:          entryRel,
		EntryFunction:  resolved.EntryFunction,
		ProjectRoot:    projectRootRel,
		RootAlias:      resolved.RootAlias,
		PackageName:    opts.PackageName,
		PackageVersion: opts.PackageVersion,
		Export:         exportRel,
		Modules:        modules,
		Packages:       packages,
	}

	outputPath, err := resolveArchiveOutput(opts.Target, opts.Output, opts.ArchiveExt)
	if err != nil {
		return nil, "", err
	}
	return archive, outputPath, nil
}

func resolveArchiveTarget(opts buildArchiveOptions) (resolvedProject, error) {
	if opts.Kind == api.ArchiveKindPackage {
		if resolved, ok, err := tryLoadPackageTarget(opts.Target); err != nil {
			return resolvedProject{}, err
		} else if ok {
			return resolved, nil
		}
	}
	return resolveRunTarget(opts.Target)
}

type bundleGraph struct {
	Paths      []string
	Sources    map[string]string
	Packages   map[string]*api.BundleArchive
	ExportPath string
}

func collectBundleSources(resolved resolvedProject, exportPath string) (*bundleGraph, error) {
	sources := make(map[string]string)
	packages := make(map[string]*api.BundleArchive)
	visiting := make(map[string]bool)
	var err error

	var visit func(path string) error
	visit = func(path string) error {
		path = filepath.Clean(path)
		if _, ok := sources[path]; ok {
			return nil
		}
		if visiting[path] {
			return fmt.Errorf("cyclic import detected: %s", path)
		}
		visiting[path] = true
		defer delete(visiting, path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read module: %w", err)
		}
		src := string(data)
		sources[path] = src

		imports, err := parseImportSpecs(src)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, spec := range imports {
			if isStdImport(spec) {
				continue
			}
			if packagePath, ok, err := resolveArchivePackageImport(path, spec, resolved.Root); err != nil {
				return err
			} else if ok {
				archive, err := api.LoadBundleFile(packagePath)
				if err != nil {
					return err
				}
				packages[filepath.Clean(packagePath)] = archive
				continue
			}
			resolvedPath, err := resolveBundleImport(path, spec, resolved.Root, resolved.RootAlias)
			if err != nil {
				return err
			}
			if err := visit(resolvedPath); err != nil {
				return err
			}
		}
		return nil
	}

	if err := visit(resolved.EntryPath); err != nil {
		return nil, err
	}

	exportAbsPath := ""
	if strings.TrimSpace(exportPath) != "" {
		exportAbsPath, err = resolveArchiveExportPath(resolved, exportPath)
		if err != nil {
			return nil, err
		}
		if err := visit(exportAbsPath); err != nil {
			return nil, err
		}
	}

	if exportAbsPath == "" {
		exportAbsPath = resolved.EntryPath
	}

	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return &bundleGraph{
		Paths:      paths,
		Sources:    sources,
		Packages:   packages,
		ExportPath: filepath.Clean(exportAbsPath),
	}, nil
}

func parseImportSpecs(src string) ([]string, error) {
	tokens := lexer.LexAll(src)
	p := parser.New(tokens)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, p.Errors()[0]
	}
	specs := make([]string, 0)
	for _, node := range program.Nodes {
		switch decl := node.(type) {
		case *ast.ImportDecl:
			specs = append(specs, decl.Path)
		case *ast.ExportDecl:
			if importDecl, ok := decl.Decl.(*ast.ImportDecl); ok {
				specs = append(specs, importDecl.Path)
			}
		}
	}
	return specs, nil
}

func isStdImport(spec string) bool {
	return strings.HasPrefix(strings.TrimSpace(spec), "std.")
}

func resolveBundleImport(importerPath string, spec string, projectRoot string, rootAlias string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("empty module path")
	}
	if filepath.IsAbs(spec) {
		return filepath.Clean(spec), nil
	}
	if rootAlias != "" && (spec == rootAlias || strings.HasPrefix(spec, rootAlias+"/")) {
		if projectRoot == "" {
			return "", fmt.Errorf("project root is not configured for import: %s", spec)
		}
		rel := strings.TrimPrefix(spec, rootAlias)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return "", fmt.Errorf("project root import must include a file path: %s", spec)
		}
		return resolveProjectEntryPath(projectRoot, rel)
	}

	baseDir := filepath.Dir(importerPath)
	return filepath.Abs(filepath.Join(baseDir, spec))
}

func resolveArchivePackageImport(importerPath string, spec string, projectRoot string) (string, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", false, nil
	}
	if strings.HasPrefix(spec, "pkg:") {
		if projectRoot == "" {
			return "", false, fmt.Errorf("named package import requires a project root: %s", spec)
		}
		name := strings.TrimPrefix(spec, "pkg:")
		name = strings.TrimSpace(strings.TrimPrefix(name, "//"))
		name = strings.Trim(name, "/")
		if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\\") {
			return "", false, fmt.Errorf("invalid package import: %s", spec)
		}
		candidates := []string{
			filepath.Join(projectRoot, ".icoo", "packages", filepath.FromSlash(name)+packageFileExt),
			filepath.Join(projectRoot, "packages", filepath.FromSlash(name)+packageFileExt),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, true, nil
			} else if err != nil && !os.IsNotExist(err) {
				return "", false, fmt.Errorf("stat package: %w", err)
			}
		}
		return "", false, fmt.Errorf("package not found: %s", spec)
	}
	if !strings.EqualFold(filepath.Ext(spec), packageFileExt) {
		return "", false, nil
	}
	if filepath.IsAbs(spec) {
		return filepath.Clean(spec), true, nil
	}
	baseDir := filepath.Dir(importerPath)
	path, err := filepath.Abs(filepath.Join(baseDir, spec))
	return path, true, err
}

func relBundlePath(root string, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("resolve bundle path: %w", err)
	}
	return filepath.ToSlash(rel), nil
}

func commonAncestor(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("bundle requires at least one module")
	}
	parts := splitPath(filepath.Clean(paths[0]))
	for _, path := range paths[1:] {
		current := splitPath(filepath.Clean(path))
		i := 0
		for i < len(parts) && i < len(current) && equalPathSegment(parts[i], current[i]) {
			i++
		}
		parts = parts[:i]
		if len(parts) == 0 {
			return filepath.VolumeName(paths[0]) + string(filepath.Separator), nil
		}
	}
	root := filepath.Join(parts...)
	if filepath.VolumeName(paths[0]) != "" && !strings.HasPrefix(strings.ToLower(root), strings.ToLower(filepath.VolumeName(paths[0]))) {
		root = filepath.VolumeName(paths[0]) + string(filepath.Separator)
	}
	return root, nil
}

func splitPath(path string) []string {
	cleaned := filepath.Clean(path)
	volume := filepath.VolumeName(cleaned)
	trimmed := strings.TrimPrefix(cleaned, volume)
	trimmed = strings.TrimPrefix(trimmed, string(filepath.Separator))
	parts := []string{}
	if volume != "" {
		parts = append(parts, volume+string(filepath.Separator))
	}
	for _, part := range strings.Split(trimmed, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return []string{cleaned}
	}
	return parts
}

func equalPathSegment(left string, right string) bool {
	return strings.EqualFold(left, right)
}

func resolveArchiveExportPath(resolved resolvedProject, exportPath string) (string, error) {
	exportPath = strings.TrimSpace(exportPath)
	if exportPath == "" {
		return "", nil
	}
	if filepath.IsAbs(exportPath) {
		return filepath.Abs(exportPath)
	}
	if resolved.Root != "" {
		return resolveProjectEntryPath(resolved.Root, filepath.ToSlash(exportPath))
	}
	baseDir := filepath.Dir(resolved.EntryPath)
	return filepath.Abs(filepath.Join(baseDir, exportPath))
}

func resolveArchiveOutput(target string, output string, ext string) (string, error) {
	if strings.TrimSpace(output) != "" {
		return filepath.Abs(output)
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("stat bundle target: %w", err)
	}
	if info.IsDir() {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return "", fmt.Errorf("resolve bundle target: %w", err)
		}
		return filepath.Join(filepath.Dir(absTarget), filepath.Base(absTarget)+ext), nil
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve bundle target: %w", err)
	}
	base := strings.TrimSuffix(absTarget, filepath.Ext(absTarget))
	return base + ext, nil
}
