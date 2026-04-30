package api

import (
	"fmt"
	"os"

	"icoo_lang/internal/compiler"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
	"icoo_lang/internal/runtime"
	"icoo_lang/internal/sema"
	"icoo_lang/internal/stdlib"
	"icoo_lang/internal/vm"
)

type Runtime struct {
	vm *vm.VM
}

func NewRuntime() *Runtime {
	machine := vm.New()
	stdlib.RegisterBuiltins(machine)
	return &Runtime{vm: machine}
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

	result, err := r.vm.Run(&runtime.Closure{Proto: compiled.Proto})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Runtime) CheckFile(path string) []error {
	src, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("read file: %w", err)}
	}
	return r.CheckSource(string(src))
}

func (r *Runtime) RunFile(path string) (runtime.Value, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return r.RunSource(string(src))
}
