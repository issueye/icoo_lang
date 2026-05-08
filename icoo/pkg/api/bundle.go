package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icoo_lang/internal/runtime"
)

const BundleVersion = 1
const (
	ArchiveKindApplication = "application"
	ArchiveKindPackage     = "package"
)

type BundleArchive struct {
	Version        int                       `json:"version"`
	Kind           string                    `json:"kind,omitempty"`
	Entry          string                    `json:"entry"`
	EntryFunction  string                    `json:"entry_function,omitempty"`
	ProjectRoot    string                    `json:"project_root,omitempty"`
	RootAlias      string                    `json:"root_alias,omitempty"`
	PackageName    string                    `json:"package_name,omitempty"`
	PackageVersion string                    `json:"package_version,omitempty"`
	Export         string                    `json:"export,omitempty"`
	Modules        map[string]string         `json:"modules"`
	Packages       map[string]*BundleArchive `json:"packages,omitempty"`
}

func LoadBundle(data []byte) (*BundleArchive, error) {
	var archive BundleArchive
	if err := json.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	if archive.Version != BundleVersion {
		return nil, fmt.Errorf("unsupported bundle version: %d", archive.Version)
	}
	if len(archive.Modules) == 0 {
		return nil, fmt.Errorf("bundle modules are required")
	}
	archive.Kind = archiveKindOrDefault(archive.Kind)
	if archive.Entry == "" && archive.Export == "" {
		return nil, fmt.Errorf("bundle entry or export is required")
	}
	if archive.Entry != "" {
		if _, ok := archive.Modules[archive.Entry]; !ok {
			return nil, fmt.Errorf("bundle entry source not found: %s", archive.Entry)
		}
	}
	if archive.Export != "" {
		if _, ok := archive.Modules[archive.Export]; !ok {
			return nil, fmt.Errorf("bundle export source not found: %s", archive.Export)
		}
	}
	return &archive, nil
}

func archiveKindOrDefault(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return ArchiveKindApplication
	}
	return kind
}

func LoadBundleFile(path string) (*BundleArchive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	return LoadBundle(data)
}

func archiveVirtualBase(path string) string {
	baseDir := filepath.Dir(path)
	ext := filepath.Ext(path)
	if strings.EqualFold(ext, ".icpkg") {
		name := strings.TrimSuffix(filepath.Base(path), ext)
		return filepath.Join(baseDir, "__bundle__", name)
	}
	return filepath.Join(baseDir, "__bundle__")
}

func (r *Runtime) SetBundledSources(sources map[string]string) {
	r.bundledSources = make(map[string]string, len(sources))
	for path, src := range sources {
		r.bundledSources[filepath.Clean(path)] = src
	}
}

func (r *Runtime) SetBundledPackages(packages map[string]*BundleArchive) {
	r.bundledPackages = make(map[string]*BundleArchive, len(packages))
	for path, archive := range packages {
		r.bundledPackages[filepath.Clean(path)] = archive
	}
}

func (r *Runtime) CheckBundleFile(path string) []error {
	archive, err := LoadBundleFile(path)
	if err != nil {
		return []error{err}
	}
	return r.CheckBundleArchive(archive)
}

func (r *Runtime) CheckBundleArchive(archive *BundleArchive) []error {
	errs := make([]error, 0)
	for relPath, src := range archive.Modules {
		moduleErrs := r.CheckSource(src)
		for _, moduleErr := range moduleErrs {
			errs = append(errs, fmt.Errorf("%s: %w", relPath, moduleErr))
		}
	}
	return errs
}

func (r *Runtime) RunBundleFile(path string) (runtime.Value, error) {
	archive, err := LoadBundleFile(path)
	if err != nil {
		return nil, err
	}
	return r.RunBundleArchive(path, archive)
}

func (r *Runtime) RunBundleArchive(path string, archive *BundleArchive) (runtime.Value, error) {
	restoreArgs := r.applyScriptArgs()
	defer restoreArgs()
	if strings.TrimSpace(archive.Entry) == "" {
		return nil, fmt.Errorf("bundle entry is required to run archive: %s", path)
	}

	virtualBase := archiveVirtualBase(path)
	sources := make(map[string]string, len(archive.Modules))
	for relPath, src := range archive.Modules {
		absPath := filepath.Join(virtualBase, filepath.FromSlash(relPath))
		sources[absPath] = src
	}
	packages := make(map[string]*BundleArchive, len(archive.Packages))
	for relPath, pkgArchive := range archive.Packages {
		absPath := filepath.Join(virtualBase, filepath.FromSlash(relPath))
		packages[absPath] = pkgArchive
	}
	r.SetBundledSources(sources)
	r.SetBundledPackages(packages)
	if archive.RootAlias != "" {
		virtualProjectRoot := virtualBase
		if archive.ProjectRoot != "" {
			virtualProjectRoot = filepath.Join(virtualBase, filepath.FromSlash(archive.ProjectRoot))
		}
		r.SetProjectRoot(virtualProjectRoot, archive.RootAlias)
	}

	entryPath := filepath.Join(virtualBase, filepath.FromSlash(archive.Entry))
	result, err := r.runModuleSource(entryPath, archive.Modules[archive.Entry])
	if err != nil {
		return nil, err
	}
	if archive.EntryFunction != "" {
		return r.InvokeGlobal(archive.EntryFunction)
	}
	return result, nil
}

func (r *Runtime) LoadPackageArchive(path string, archive *BundleArchive) (*runtime.Module, error) {
	if strings.TrimSpace(archive.Export) == "" {
		return nil, fmt.Errorf("bundle export is required to import archive: %s", path)
	}

	originalSources := r.bundledSources
	originalPackages := r.bundledPackages
	originalProjectRoot := r.projectRoot
	originalProjectRootAlias := r.projectRootAlias
	defer func() {
		r.bundledSources = originalSources
		r.bundledPackages = originalPackages
		r.projectRoot = originalProjectRoot
		r.projectRootAlias = originalProjectRootAlias
	}()

	virtualBase := archiveVirtualBase(path)
	sources := make(map[string]string, len(archive.Modules))
	for relPath, src := range archive.Modules {
		absPath := filepath.Join(virtualBase, filepath.FromSlash(relPath))
		sources[absPath] = src
	}
	packages := make(map[string]*BundleArchive, len(archive.Packages))
	for relPath, pkgArchive := range archive.Packages {
		absPath := filepath.Join(virtualBase, filepath.FromSlash(relPath))
		packages[absPath] = pkgArchive
	}
	r.SetBundledSources(sources)
	r.SetBundledPackages(packages)
	if archive.RootAlias != "" {
		virtualProjectRoot := virtualBase
		if archive.ProjectRoot != "" {
			virtualProjectRoot = filepath.Join(virtualBase, filepath.FromSlash(archive.ProjectRoot))
		}
		r.SetProjectRoot(virtualProjectRoot, archive.RootAlias)
	}

	exportPath := filepath.Join(virtualBase, filepath.FromSlash(archive.Export))
	return r.loadModule("", exportPath)
}
