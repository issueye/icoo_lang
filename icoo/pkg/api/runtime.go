package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"icoo_lang/internal/compiler"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
	"icoo_lang/internal/runtime"
	"icoo_lang/internal/sema"
	"icoo_lang/internal/stdlib"
	"icoo_lang/internal/vm"
)

type Runtime struct {
	vm               *vm.VM
	modules          map[string]*runtime.Module
	bundledSources   map[string]string
	projectRoot      string
	projectRootAlias string
	scriptArgs       []string
}

func NewRuntime() *Runtime {
	machine := vm.New()
	rt := &Runtime{
		vm:             machine,
		modules:        make(map[string]*runtime.Module),
		bundledSources: make(map[string]string),
	}
	machine.SetModuleLoader(rt.loadModule)
	stdlib.RegisterBuiltins(machine)
	goruntime.SetFinalizer(rt, func(runtime *Runtime) {
		_ = runtime.vm.Close()
	})
	return rt
}

func (r *Runtime) SetProjectRoot(root string, alias string) {
	r.projectRoot = filepath.Clean(root)
	r.projectRootAlias = strings.TrimSpace(alias)
}

func (r *Runtime) SetScriptArgs(args []string) {
	r.scriptArgs = append([]string(nil), args...)
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

func (r *Runtime) VM() *vm.VM {
	return r.vm
}

func (r *Runtime) ConfigureGoPool(workers, queueSize int) error {
	return r.vm.ConfigureGoPool(workers, queueSize)
}

func (r *Runtime) Stats() vm.RuntimeStats {
	return r.vm.Stats()
}

func (r *Runtime) CollectGarbage() vm.RuntimeStats {
	return r.vm.CollectGarbage()
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	return r.vm.Shutdown(ctx)
}

func (r *Runtime) Close() error {
	return r.vm.Close()
}

func (r *Runtime) InvokeGlobal(name string) (runtime.Value, error) {
	restoreArgs := r.applyScriptArgs()
	defer restoreArgs()

	value, ok := r.vm.GetGlobal(name)
	if !ok {
		return nil, fmt.Errorf("undefined global: %s", name)
	}

	switch value.(type) {
	case *runtime.Closure, *runtime.NativeFunction:
		return r.vm.CallDetached(value, nil)
	default:
		return nil, fmt.Errorf("global is not callable: %s", name)
	}
}

func (r *Runtime) RunReplLine(line string) (runtime.Value, error) {
	// If the line is a pure expression, wrap as return to capture value
	wrapped := line
	if isExpression(line) {
		wrapped = "return " + line
	}

	tokens := lexer.LexAll(wrapped)
	p := parser.New(tokens)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return nil, p.Errors()[0]
	}
	if len(program.Nodes) == 0 {
		return nil, nil
	}

	// Skip sema for REPL; compiler resolves unknowns as globals
	compiled, compileErrs := compiler.Compile(program)
	if len(compileErrs) > 0 {
		return nil, compileErrs[0]
	}

	result, err := r.vm.RunModule("", &runtime.Closure{Proto: compiled.Proto})
	return result, err
}

func isExpression(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "@") {
		return false
	}
	keywords := []string{"let ", "const ", "fn ", "class ", "import ", "export ", "if ", "while ", "for ", "match ", "try ", "throw ", "break ", "continue ", "go ", "select ", "return ", "type ", "interface "}
	for _, kw := range keywords {
		if strings.HasPrefix(trimmed, kw) {
			return false
		}
	}
	return true
}

func (r *Runtime) runModuleSource(path, src string) (runtime.Value, error) {
	restoreArgs := r.applyScriptArgs()
	defer restoreArgs()

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

func (r *Runtime) applyScriptArgs() func() {
	original := append([]string(nil), os.Args...)
	execName := ""
	if len(original) > 0 {
		execName = original[0]
	}
	scriptArgs := r.scriptArgs
	if scriptArgs == nil {
		scriptArgs = []string{}
	}
	os.Args = append([]string{execName}, scriptArgs...)
	return func() {
		os.Args = original
	}
}

func (r *Runtime) loadModule(importerPath, spec string) (*runtime.Module, error) {
	if mod, ok := stdlib.LoadModule(spec); ok {
		return mod, nil
	}

	resolved, err := r.resolveModulePath(importerPath, spec)
	if err != nil {
		return nil, err
	}
	if mod, ok := r.modules[resolved]; ok {
		return mod, nil
	}

	src, ok := r.bundledSources[filepath.Clean(resolved)]
	if !ok {
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, fmt.Errorf("read module: %w", err)
		}
		src = string(data)
	}

	childVM := vm.New()
	childVM.SetModuleLoader(r.loadModule)
	stdlib.RegisterBuiltins(childVM)

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

func (r *Runtime) resolveModulePath(importerPath, spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("empty module path")
	}
	if filepath.IsAbs(spec) {
		return filepath.Clean(spec), nil
	}
	if r.projectRootAlias != "" && (spec == r.projectRootAlias || strings.HasPrefix(spec, r.projectRootAlias+"/")) {
		if r.projectRoot == "" {
			return "", fmt.Errorf("project root is not configured for import: %s", spec)
		}
		rel := strings.TrimPrefix(spec, r.projectRootAlias)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return "", fmt.Errorf("project root import must include a file path: %s", spec)
		}
		return resolvePathWithinRoot(r.projectRoot, rel, spec)
	}
	baseDir := "."
	if importerPath != "" {
		baseDir = filepath.Dir(importerPath)
	}
	return filepath.Abs(filepath.Join(baseDir, spec))
}

func resolvePathWithinRoot(root, rel, original string) (string, error) {
	joined := filepath.Join(root, filepath.FromSlash(rel))
	absPath, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve module path: %w", err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	relToRoot, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("resolve project root relation: %w", err)
	}
	relToRoot = filepath.ToSlash(relToRoot)
	if relToRoot == ".." || strings.HasPrefix(relToRoot, "../") {
		return "", fmt.Errorf("project root import must stay within project root: %s", original)
	}
	return absPath, nil
}
