package vm

import (
	"context"
	"fmt"
	"maps"
	"os"
	"sync"
	"time"

	"icoo_lang/internal/concurrency"
	"icoo_lang/internal/runtime"
)

type ModuleLoader func(importerPath, spec string) (*runtime.Module, error)

type CallFrame struct {
	Closure  *runtime.Closure
	Module   *runtime.Module
	Receiver *runtime.ObjectValue
	Super    *runtime.ClassValue
	IP       int
	Base     int
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

	openUpvalues map[int]*runtime.Upvalue

	mu sync.RWMutex

	loadModule ModuleLoader
	lastModule *runtime.Module

	pool          *concurrency.GoroutinePool
	poolOnce      sync.Once
	poolWorkers   int
	poolQueueSize int
}

func New() *VM {
	vm := &VM{
		stack:        make([]runtime.Value, 0, 256),
		frames:       make([]CallFrame, 0, 64),
		handlers:     make([]ExceptionHandler, 0, 16),
		globals:      make(map[string]runtime.Value),
		builtins:     make(map[string]runtime.Value),
		modules:      make(map[string]*runtime.Module),
		openUpvalues: make(map[int]*runtime.Upvalue),
		poolWorkers:  8,
	}
	vm.poolQueueSize = vm.poolWorkers * 16
	return vm
}

func (vm *VM) Pool() *concurrency.GoroutinePool {
	vm.poolOnce.Do(func() {
		vm.pool = concurrency.NewPool(vm.poolWorkers, vm.poolQueueSize, vm.goExecutor)
	})
	return vm.pool
}

func (vm *VM) ConfigureGoPool(workers, queueSize int) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.pool != nil {
		stats := vm.pool.Stats()
		if stats.Active > 0 || stats.Queued > 0 {
			return fmt.Errorf("cannot reconfigure goroutine pool while %d active and %d queued tasks remain", stats.Active, stats.Queued)
		}
		if err := vm.pool.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("shutdown goroutine pool: %w", err)
		}
		vm.pool = nil
		vm.poolOnce = sync.Once{}
	}

	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = workers * 16
	}
	vm.poolWorkers = workers
	vm.poolQueueSize = queueSize
	return nil
}

func (vm *VM) Shutdown(ctx context.Context) error {
	vm.mu.Lock()
	pool := vm.pool
	vm.pool = nil
	vm.poolOnce = sync.Once{}
	vm.mu.Unlock()
	if pool == nil {
		return nil
	}
	return pool.Shutdown(ctx)
}

func (vm *VM) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return vm.Shutdown(ctx)
}

func (vm *VM) goExecutor(task *concurrency.GoTask) {
	switch callee := task.Callee.(type) {
	case *runtime.Closure:
		vm.mu.RLock()
		globals := make(map[string]runtime.Value, len(vm.globals))
		maps.Copy(globals, vm.globals)
		modules := make(map[string]*runtime.Module, len(vm.modules))
		maps.Copy(modules, vm.modules)
		vm.mu.RUnlock()
		sub := &VM{
			stack:    make([]runtime.Value, 0, 64),
			frames:   make([]CallFrame, 0, 8),
			handlers: nil,
			globals:  globals,
			builtins: vm.builtins,
			modules:  modules,
		}
		sub.stack = append(sub.stack, callee)
		for _, arg := range task.Args {
			sub.stack = append(sub.stack, arg)
		}
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
	case *runtime.BoundMethod:
		sub := vm.cloneForInvocation()
		if err := sub.beginMethodCall(callee.Name, callee.Method, callee.Receiver, callee.Super, task.Args); err != nil {
			fmt.Fprintf(os.Stderr, "goroutine: %v\n", err)
			return
		}
		if _, err := sub.runLoop(); err != nil {
			fmt.Fprintf(os.Stderr, "goroutine: %v\n", err)
		}
	case *runtime.MethodProxy:
		fmt.Fprintf(os.Stderr, "goroutine: method proxy requires active method receiver\n")
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
	vm.mu.Lock()
	vm.builtins[name] = v
	vm.globals[name] = v
	vm.mu.Unlock()
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

func (vm *VM) GlobalNames() []string {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	names := make([]string, 0, len(vm.globals))
	for k := range vm.globals {
		names = append(names, k)
	}
	return names
}

func (vm *VM) GetGlobal(name string) (runtime.Value, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	value, ok := vm.globals[name]
	return value, ok
}

func runtimeError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func (vm *VM) captureUpvalue(slot int) *runtime.Upvalue {
	if uv, ok := vm.openUpvalues[slot]; ok {
		return uv
	}
	uv := &runtime.Upvalue{Location: &vm.stack[slot]}
	vm.openUpvalues[slot] = uv
	return uv
}

func (vm *VM) closeUpvalues(fromSlot int) {
	for slot, uv := range vm.openUpvalues {
		if slot >= fromSlot && uv.Location != nil {
			uv.Closed = *uv.Location
			uv.Location = nil
			delete(vm.openUpvalues, slot)
		}
	}
}
