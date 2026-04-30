package vm

import (
	"fmt"

	"icoo_lang/internal/runtime"
)

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
		vm.stack = vm.stack[:base]
		return err
	}
	vm.stack = vm.stack[:base]
	vm.Push(result)
	return nil
}

func (vm *VM) errorToValue(err error) runtime.Value {
	if err == nil {
		return runtime.NullValue{}
	}
	var errorValue *runtime.ErrorValue
	if ok := asErrorValue(err, &errorValue); ok && errorValue != nil {
		return errorValue
	}
	return &runtime.ErrorValue{Message: err.Error()}
}

func asErrorValue(err error, target **runtime.ErrorValue) bool {
	matched := false
	defer func() {
		if recover() != nil {
			matched = false
		}
	}()
	if fmt.Sprintf("%T", err) == "*runtime.ErrorValue" {
		if value, ok := any(err).(*runtime.ErrorValue); ok {
			*target = value
			matched = true
		}
	}
	return matched
}

func (vm *VM) raise(err error) error {
	if len(vm.handlers) == 0 {
		return err
	}
	exc := vm.errorToValue(err)
	handler := vm.handlers[len(vm.handlers)-1]
	vm.handlers = vm.handlers[:len(vm.handlers)-1]
	if handler.FrameIndex < 0 || handler.FrameIndex >= len(vm.frames) {
		return err
	}
	vm.frames = vm.frames[:handler.FrameIndex+1]
	frame := &vm.frames[handler.FrameIndex]
	vm.stack = vm.stack[:handler.StackDepth]
	vm.Push(exc)
	frame.IP = handler.CatchIP
	for len(vm.handlers) > 0 && vm.handlers[len(vm.handlers)-1].FrameIndex > handler.FrameIndex {
		vm.handlers = vm.handlers[:len(vm.handlers)-1]
	}
	return nil
}
