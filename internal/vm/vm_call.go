package vm

import (
	"errors"

	"icoo_lang/internal/bytecode"
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
	var module *runtime.Module
	if len(vm.frames) > 0 {
		module = vm.frames[len(vm.frames)-1].Module
	}
	vm.frames = append(vm.frames, CallFrame{
		Closure: cl,
		Module:  module,
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
		exc := vm.errorToValue(err)
		if len(exc.Stack) == 0 {
			exc.Stack = vm.captureStack()
		}
		exc.Stack = append([]runtime.StackFrame{{
			Function: fn.Name,
			Native:   true,
		}}, exc.Stack...)
		return exc
	}
	vm.stack = vm.stack[:base]
	vm.Push(result)
	return nil
}

func (vm *VM) errorToValue(err error) *runtime.ErrorValue {
	if err == nil {
		return &runtime.ErrorValue{}
	}
	var errorValue *runtime.ErrorValue
	if ok := asErrorValue(err, &errorValue); ok && errorValue != nil {
		return errorValue
	}
	exc := &runtime.ErrorValue{Message: err.Error()}
	if cause := errors.Unwrap(err); cause != nil {
		exc.Cause = vm.errorToValue(cause)
	}
	return exc
}

func asErrorValue(err error, target **runtime.ErrorValue) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}

func (vm *VM) captureStack() []runtime.StackFrame {
	frames := make([]runtime.StackFrame, 0, len(vm.frames))
	for i := len(vm.frames) - 1; i >= 0; i-- {
		frame := vm.frames[i]
		stackFrame := runtime.StackFrame{Function: "<anonymous>"}
		if frame.Closure != nil && frame.Closure.Proto != nil {
			if frame.Closure.Proto.Name != "" {
				stackFrame.Function = frame.Closure.Proto.Name
			}
			if chunk, ok := frame.Closure.Proto.Chunk.(*bytecode.Chunk); ok {
				ip := frame.IP - 1
				if ip < 0 {
					ip = 0
				}
				if ip < len(chunk.Lines) {
					stackFrame.Line = chunk.Lines[ip]
				}
			}
		}
		if frame.Module != nil {
			stackFrame.File = frame.Module.Path
		}
		frames = append(frames, stackFrame)
	}
	return frames
}

func (vm *VM) raise(err error) error {
	exc := vm.errorToValue(err)
	if len(exc.Stack) == 0 {
		exc.Stack = vm.captureStack()
	}
	if len(vm.handlers) == 0 {
		return exc
	}
	handler := vm.handlers[len(vm.handlers)-1]
	vm.handlers = vm.handlers[:len(vm.handlers)-1]
	if handler.FrameIndex < 0 || handler.FrameIndex >= len(vm.frames) {
		return exc
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
