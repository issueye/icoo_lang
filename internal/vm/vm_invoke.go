package vm

import (
	"icoo_lang/internal/runtime"
)

func (vm *VM) nativeContext() *runtime.NativeContext {
	return &runtime.NativeContext{
		CallDetached: vm.CallDetached,
	}
}

func (vm *VM) CallDetached(callee runtime.Value, args []runtime.Value) (runtime.Value, error) {
	sub := vm.cloneForInvocation()
	clonedCallee := cloneDetachedValue(callee)
	clonedArgs := make([]runtime.Value, len(args))
	for i, arg := range args {
		clonedArgs[i] = cloneDetachedValue(arg)
	}

	sub.stack = append(sub.stack, clonedCallee)
	sub.stack = append(sub.stack, clonedArgs...)
	if err := sub.CallValue(clonedCallee, len(clonedArgs)); err != nil {
		return nil, err
	}
	return sub.runLoop()
}

func (vm *VM) cloneForInvocation() *VM {
	vm.mu.RLock()
	globals := make(map[string]runtime.Value, len(vm.globals))
	for k, v := range vm.globals {
		globals[k] = v
	}
	modules := make(map[string]*runtime.Module, len(vm.modules))
	for k, v := range vm.modules {
		modules[k] = v
	}
	vm.mu.RUnlock()

	return &VM{
		stack:        make([]runtime.Value, 0, 64),
		frames:       make([]CallFrame, 0, 8),
		handlers:     make([]ExceptionHandler, 0, 4),
		globals:      globals,
		builtins:     vm.builtins,
		modules:      modules,
		openUpvalues: make(map[int]*runtime.Upvalue),
		loadModule:   vm.loadModule,
	}
}

func cloneDetachedValue(value runtime.Value) runtime.Value {
	switch v := value.(type) {
	case nil:
		return nil
	case runtime.NullValue, runtime.BoolValue, runtime.IntValue, runtime.FloatValue, runtime.StringValue:
		return v
	case *runtime.ArrayValue:
		items := make([]runtime.Value, len(v.Elements))
		for i, elem := range v.Elements {
			items[i] = cloneDetachedValue(elem)
		}
		return &runtime.ArrayValue{Elements: items}
	case *runtime.ObjectValue:
		fields := make(map[string]runtime.Value, len(v.Fields))
		for key, fieldValue := range v.Fields {
			fields[key] = cloneDetachedValue(fieldValue)
		}
		return &runtime.ObjectValue{Fields: fields}
	case *runtime.ErrorValue:
		cloned := &runtime.ErrorValue{
			Message: v.Message,
			Stack:   append([]runtime.StackFrame(nil), v.Stack...),
		}
		if v.Cause != nil {
			if cause, ok := cloneDetachedValue(v.Cause).(*runtime.ErrorValue); ok {
				cloned.Cause = cause
			}
		}
		return cloned
	case *runtime.Closure:
		cloned := &runtime.Closure{
			Proto:    v.Proto,
			Upvalues: make([]*runtime.Upvalue, len(v.Upvalues)),
		}
		for i, uv := range v.Upvalues {
			if uv == nil {
				continue
			}
			cloned.Upvalues[i] = &runtime.Upvalue{
				Closed: cloneDetachedValue(uv.Get()),
			}
		}
		return cloned
	default:
		return value
	}
}
