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

type BundleArchive struct {
	Version       int               `json:"version"`
	Entry         string            `json:"entry"`
	EntryFunction string            `json:"entry_function,omitempty"`
	ProjectRoot   string            `json:"project_root,omitempty"`
	RootAlias     string            `json:"root_alias,omitempty"`
	Modules       map[string]string `json:"modules"`
}

func LoadBundle(data []byte) (*BundleArchive, error) {
	var archive BundleArchive
	if err := json.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("parse bundle: %w", err)
	}
	if archive.Version != BundleVersion {
		return nil, fmt.Errorf("unsupported bundle version: %d", archive.Version)
	}
	if strings.TrimSpace(archive.Entry) == "" {
		return nil, fmt.Errorf("bundle entry is required")
	}
	if len(archive.Modules) == 0 {
		return nil, fmt.Errorf("bundle modules are required")
	}
	if _, ok := archive.Modules[archive.Entry]; !ok {
		return nil, fmt.Errorf("bundle entry source not found: %s", archive.Entry)
	}
	return &archive, nil
}

func LoadBundleFile(path string) (*BundleArchive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	return LoadBundle(data)
}

func (r *Runtime) SetBundledSources(sources map[string]string) {
	r.bundledSources = make(map[string]string, len(sources))
	for path, src := range sources {
		r.bundledSources[filepath.Clean(path)] = src
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

	virtualBase := filepath.Join(filepath.Dir(path), "__bundle__")
	sources := make(map[string]string, len(archive.Modules))
	for relPath, src := range archive.Modules {
		absPath := filepath.Join(virtualBase, filepath.FromSlash(relPath))
		sources[absPath] = src
	}
	r.SetBundledSources(sources)
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
