package vm

import (
	"fmt"
	"os"
	"sync"

	"icoo_lang/internal/concurrency"
	"icoo_lang/internal/runtime"
)

type ModuleLoader func(importerPath, spec string) (*runtime.Module, error)

type CallFrame struct {
	Closure *runtime.Closure
	Module  *runtime.Module
	IP      int
	Base    int
}

type ExceptionHandler struct {
	FrameIndex int
	StackDepth int
	CatchIP    int
}

type VM struct {
	stack    []runtime.Value
	frames   []CallFrame
	handlers []ExceptionHandler
	globals  map[string]runtime.Value
	builtins map[string]runtime.Value
	modules  map[string]*runtime.Module

	loadModule ModuleLoader
	lastModule *runtime.Module

	pool     *concurrency.GoroutinePool
	poolOnce sync.Once
}

func New() *VM {
	vm := &VM{
		stack:    make([]runtime.Value, 0, 256),
		frames:   make([]CallFrame, 0, 64),
		handlers: make([]ExceptionHandler, 0, 16),
		globals:  make(map[string]runtime.Value),
		builtins: make(map[string]runtime.Value),
		modules:  make(map[string]*runtime.Module),
	}
	return vm
}

func (vm *VM) Pool() *concurrency.GoroutinePool {
	vm.poolOnce.Do(func() {
		vm.pool = concurrency.NewPool(8, vm.goExecutor)
	})
	return vm.pool
}

func (vm *VM) goExecutor(task *concurrency.GoTask) {
	switch callee := task.Callee.(type) {
	case *runtime.Closure:
		sub := &VM{
			stack:    make([]runtime.Value, 0, 64),
			frames:   make([]CallFrame, 0, 8),
			handlers: nil,
			globals:  vm.globals,
			builtins: vm.builtins,
			modules:  vm.modules,
		}
		for _, arg := range task.Args {
			sub.stack = append(sub.stack, arg)
		}
		sub.stack = append(sub.stack, callee)
		base := len(sub.stack) - len(task.Args) - 1
		sub.frames = append(sub.frames, CallFrame{
			Closure: callee,
			IP:      0,
			Base:    base,
		})
		_, err := sub.runLoop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "goroutine: %v\n", err)
		}
	case *runtime.NativeFunction:
		_, err := callee.Fn(task.Args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goroutine: %v\n", err)
		}
	}
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

func (vm *VM) SetModuleLoader(loader ModuleLoader) {
	vm.loadModule = loader
}

func (vm *VM) Frames() []CallFrame {
	return vm.frames
}

func (vm *VM) LastModule() *runtime.Module {
	return vm.lastModule
}

func runtimeError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
