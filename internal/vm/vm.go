package vm

import (
	"fmt"

	"icoo_lang/internal/runtime"
)

type CallFrame struct {
	Closure *runtime.Closure
	IP      int
	Base    int
}

type VM struct {
	stack    []runtime.Value
	frames   []CallFrame
	globals  map[string]runtime.Value
	builtins map[string]runtime.Value
	modules  map[string]*runtime.Module
}

func New() *VM {
	vm := &VM{
		stack:    make([]runtime.Value, 0, 256),
		frames:   make([]CallFrame, 0, 64),
		globals:  make(map[string]runtime.Value),
		builtins: make(map[string]runtime.Value),
		modules:  make(map[string]*runtime.Module),
	}
	return vm
}

func (vm *VM) Push(v runtime.Value) {
	vm.stack = append(vm.stack, v)
}

func (vm *VM) Pop() runtime.Value {
	if len(vm.stack) == 0 {
		return nil
	}
	v := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return v
}

func (vm *VM) Peek(distance int) runtime.Value {
	idx := len(vm.stack) - 1 - distance
	if idx < 0 || idx >= len(vm.stack) {
		return nil
	}
	return vm.stack[idx]
}

func (vm *VM) DefineBuiltin(name string, v runtime.Value) {
	vm.builtins[name] = v
	vm.globals[name] = v
}

func runtimeError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
