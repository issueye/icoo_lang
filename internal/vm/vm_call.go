package vm

import "icoo_lang/internal/runtime"

func (vm *VM) CallValue(callee runtime.Value, argc int) error {
	switch fn := callee.(type) {
	case *runtime.Closure:
		return vm.callClosure(fn, argc)
	case *runtime.NativeFunction:
		return vm.callNative(fn, argc)
	default:
		return runtimeError("value is not callable: %s", runtime.KindName(callee))
	}
}

func (vm *VM) callClosure(cl *runtime.Closure, argc int) error {
	if cl == nil || cl.Proto == nil {
		return runtimeError("invalid closure")
	}
	if cl.Proto.Arity != argc {
		return runtimeError("expected %d arguments, got %d", cl.Proto.Arity, argc)
	}
	base := len(vm.stack) - argc - 1
	vm.frames = append(vm.frames, CallFrame{
		Closure: cl,
		IP:      0,
		Base:    base,
	})
	return nil
}

func (vm *VM) callNative(fn *runtime.NativeFunction, argc int) error {
	if fn.Arity >= 0 && fn.Arity != argc {
		return runtimeError("expected %d arguments, got %d", fn.Arity, argc)
	}
	base := len(vm.stack) - argc - 1
	args := append([]runtime.Value(nil), vm.stack[base+1:base+1+argc]...)
	result, err := fn.Fn(args)
	if err != nil {
		return err
	}
	vm.stack = vm.stack[:base]
	vm.Push(result)
	return nil
}
