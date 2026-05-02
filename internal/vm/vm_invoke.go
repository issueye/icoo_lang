package vm

import (
	"icoo_lang/internal/runtime"
)

func (vm *VM) nativeContext() *runtime.NativeContext {
	return &runtime.NativeContext{
		CallDetached:         vm.CallDetached,
		CallDetachedWithArgs: vm.CallDetachedWithArgs,
		CallInline:           vm.CallInline,
		CallInlineWithArgs:   vm.CallInlineWithArgs,
	}
}

func (vm *VM) CallDetached(callee runtime.Value, args []runtime.Value) (runtime.Value, error) {
	result, _, err := vm.CallDetachedWithArgs(callee, args)
	return result, err
}

func (vm *VM) CallDetachedWithArgs(callee runtime.Value, args []runtime.Value) (runtime.Value, []runtime.Value, error) {
	sub := vm.cloneForInvocation()
	clonedCallee := cloneDetachedValue(callee)
	clonedArgs := make([]runtime.Value, len(args))
	for i, arg := range args {
		clonedArgs[i] = cloneDetachedValue(arg)
	}

	sub.stack = append(sub.stack, clonedCallee)
	sub.stack = append(sub.stack, clonedArgs...)
	if err := sub.CallValue(clonedCallee, len(clonedArgs)); err != nil {
		return nil, clonedArgs, err
	}
	result, err := sub.runLoop()
	return result, clonedArgs, err
}

func (vm *VM) CallInline(callee runtime.Value, args []runtime.Value) (runtime.Value, error) {
	result, _, err := vm.CallInlineWithArgs(callee, args)
	return result, err
}

func (vm *VM) CallInlineWithArgs(callee runtime.Value, args []runtime.Value) (runtime.Value, []runtime.Value, error) {
	stackDepth := len(vm.stack)
	frameDepth := len(vm.frames)

	vm.stack = append(vm.stack, callee)
	vm.stack = append(vm.stack, args...)
	if err := vm.CallValue(callee, len(args)); err != nil {
		vm.stack = vm.stack[:stackDepth]
		return nil, args, err
	}

	if len(vm.frames) == frameDepth {
		return vm.Pop(), args, nil
	}

	result, err := vm.runLoopUntil(frameDepth)
	return result, args, err
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
		var class *runtime.ClassValue
		if v.Class != nil {
			class = cloneDetachedValue(v.Class).(*runtime.ClassValue)
		}
		return &runtime.ObjectValue{Fields: fields, Class: class}
	case *runtime.ClassValue:
		methods := make(map[string]*runtime.MethodDef, len(v.Methods))
		for key, method := range v.Methods {
			if method == nil {
				continue
			}
			methods[key] = cloneDetachedValue(method).(*runtime.MethodDef)
		}
		var super *runtime.ClassValue
		if v.Super != nil {
			super = cloneDetachedValue(v.Super).(*runtime.ClassValue)
		}
		var init *runtime.MethodDef
		if v.Init != nil {
			init = cloneDetachedValue(v.Init).(*runtime.MethodDef)
		}
		return &runtime.ClassValue{Name: v.Name, Super: super, Init: init, Methods: methods}
	case *runtime.BoundMethod:
		var receiver *runtime.ObjectValue
		if v.Receiver != nil {
			receiver = cloneDetachedValue(v.Receiver).(*runtime.ObjectValue)
		}
		var method *runtime.MethodDef
		if v.Method != nil {
			method = cloneDetachedValue(v.Method).(*runtime.MethodDef)
		}
		var super *runtime.ClassValue
		if v.Super != nil {
			super = cloneDetachedValue(v.Super).(*runtime.ClassValue)
		}
		return &runtime.BoundMethod{
			Name:     v.Name,
			Receiver: receiver,
			Method:   method,
			Super:    super,
			Init:     v.Init,
		}
	case *runtime.MethodProxy:
		var method *runtime.Closure
		if v.Method != nil {
			method = cloneDetachedValue(v.Method).(*runtime.Closure)
		}
		return &runtime.MethodProxy{
			Name:   v.Name,
			Method: method,
			Init:   v.Init,
		}
	case *runtime.MethodDef:
		var callable runtime.Value
		if v.Callable != nil {
			callable = cloneDetachedValue(v.Callable)
		}
		return &runtime.MethodDef{
			Name:         v.Name,
			Callable:     callable,
			ImplicitThis: v.ImplicitThis,
			Init:         v.Init,
		}
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
