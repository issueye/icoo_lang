package api

import (
	"fmt"
	"os"
	"path/filepath"

	"icoo_lang/internal/compiler"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
	"icoo_lang/internal/runtime"
	"icoo_lang/internal/sema"
	"icoo_lang/internal/stdlib"
	"icoo_lang/internal/vm"
)

type Runtime struct {
	vm      *vm.VM
	modules map[string]*runtime.Module
}

func NewRuntime() *Runtime {
	machine := vm.New()
	rt := &Runtime{
		vm:      machine,
		modules: make(map[string]*runtime.Module),
	}
	machine.SetModuleLoader(rt.loadModule)
	stdlib.RegisterBuiltins(machine)
	return rt
}

func (r *Runtime) CheckSource(src string) []error {
	tokens := lexer.LexAll(src)
	p := parser.New(tokens)
	program := p.ParseProgram()

	errs := make([]error, 0, len(p.Errors()))
	errs = append(errs, p.Errors()...)
	for _, d := range sema.Analyze(program) {
		errs = append(errs, d)
	}
	return errs
}

func (r *Runtime) RunSource(src string) (runtime.Value, error) {
	return r.runModuleSource("", src)
}

func (r *Runtime) CheckFile(path string) []error {
	src, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("read file: %w", err)}
	}
	return r.CheckSource(string(src))
}

func (r *Runtime) RunFile(path string) (runtime.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return r.runModuleSource(absPath, string(src))
}

func (r *Runtime) runModuleSource(path, src string) (runtime.Value, error) {
	tokens := lexer.LexAll(src)
	p := parser.New(tokens)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, p.Errors()[0]
	}

	semaErrs := sema.Analyze(program)
	if len(semaErrs) > 0 {
		return nil, semaErrs[0]
	}

	compiled, compileErrs := compiler.Compile(program)
	if len(compileErrs) > 0 {
		return nil, compileErrs[0]
	}

	result, err := r.vm.RunModule(path, &runtime.Closure{Proto: compiled.Proto})
	if err != nil {
		return nil, err
	}
	if path != "" {
		if len(r.vm.Frames()) > 0 {
			return result, nil
		}
	}
	return result, nil
}

func (r *Runtime) loadModule(importerPath, spec string) (*runtime.Module, error) {
	if mod, ok := stdlib.LoadModule(spec); ok {
		return mod, nil
	}

	resolved, err := resolveModulePath(importerPath, spec)
	if err != nil {
		return nil, err
	}
	if mod, ok := r.modules[resolved]; ok {
		return mod, nil
	}

	src, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read module: %w", err)
	}

	childVM := vm.New()
	childVM.SetModuleLoader(r.loadModule)
	stdlib.RegisterBuiltins(childVM)

	tokens := lexer.LexAll(string(src))
	p := parser.New(tokens)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, p.Errors()[0]
	}
	semaErrs := sema.Analyze(program)
	if len(semaErrs) > 0 {
		return nil, semaErrs[0]
	}
	compiled, compileErrs := compiler.Compile(program)
	if len(compileErrs) > 0 {
		return nil, compileErrs[0]
	}

	module := &runtime.Module{
		Name:    filepath.Base(resolved),
		Path:    resolved,
		Exports: make(map[string]runtime.Value),
	}
	r.modules[resolved] = module

	if _, err := childVM.RunModule(resolved, &runtime.Closure{Proto: compiled.Proto}); err != nil {
		delete(r.modules, resolved)
		return nil, err
	}
	if len(childVM.Frames()) > 0 {
		delete(r.modules, resolved)
		return nil, fmt.Errorf("module execution did not finish: %s", resolved)
	}
	finished := childVM.LastModule()
	if finished != nil {
		module.Exports = finished.Exports
		module.Done = true
	}
	return module, nil
}

func resolveModulePath(importerPath, spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("empty module path")
	}
	if filepath.IsAbs(spec) {
		return filepath.Clean(spec), nil
	}
	baseDir := "."
	if importerPath != "" {
		baseDir = filepath.Dir(importerPath)
	}
	return filepath.Abs(filepath.Join(baseDir, spec))
}
