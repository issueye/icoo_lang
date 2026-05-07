package stdlib

import (
	"fmt"
	"os"
	"reflect"

	"icoo_lang/internal/runtime"
	stdio "icoo_lang/internal/stdlib/io"
	"icoo_lang/internal/vm"
)

func RegisterBuiltins(machine *vm.VM) {
	machine.DefineBuiltin("print", &runtime.NativeFunction{Name: "print", Arity: -1, Fn: builtinPrint})
	machine.DefineBuiltin("println", &runtime.NativeFunction{Name: "println", Arity: -1, Fn: builtinPrintln})
	machine.DefineBuiltin("len", &runtime.NativeFunction{Name: "len", Arity: 1, Fn: builtinLen})
	machine.DefineBuiltin("typeOf", &runtime.NativeFunction{Name: "typeOf", Arity: 1, Fn: builtinTypeOf})
	machine.DefineBuiltin("argv", &runtime.NativeFunction{Name: "argv", Arity: 0, Fn: builtinArgv})
	machine.DefineBuiltin("chan", &runtime.NativeFunction{Name: "chan", Arity: -1, Fn: builtinChan})
	machine.DefineBuiltin("panic", &runtime.NativeFunction{Name: "panic", Arity: 1, Fn: builtinPanic})
	machine.DefineBuiltin("error", &runtime.NativeFunction{Name: "error", Arity: -1, Fn: builtinError})
	machine.DefineBuiltin("__select", &runtime.NativeFunction{Name: "__select", Arity: 1, Fn: builtinSelect})
	machine.DefineBuiltin("satisfies", &runtime.NativeFunction{Name: "satisfies", Arity: 2, Fn: builtinSatisfies})
	machine.DefineBuiltin("_tryCheck", &runtime.NativeFunction{Name: "_tryCheck", Arity: 1, Fn: builtinTryCheck})
	machine.DefineBuiltin("__buildClass", &runtime.NativeFunction{Name: "__buildClass", Arity: 4, Fn: builtinBuildClass})
	machine.DefineBuiltin("__methodDef", &runtime.NativeFunction{Name: "__methodDef", Arity: 4, Fn: builtinMethodDef})
	machine.DefineBuiltin("__methodProxy", &runtime.NativeFunction{Name: "__methodProxy", Arity: 3, Fn: builtinMethodProxy})
	machine.DefineBuiltin("__superGet", &runtime.NativeFunction{Name: "__superGet", Arity: 3, Fn: builtinSuperGet})
}

func builtinPrint(args []runtime.Value) (runtime.Value, error) {
	return stdio.Print(args)
}

func builtinPrintln(args []runtime.Value) (runtime.Value, error) {
	return stdio.Println(args)
}

func builtinLen(args []runtime.Value) (runtime.Value, error) {
	switch value := args[0].(type) {
	case runtime.StringValue:
		return runtime.IntValue{Value: int64(len(value.Value))}, nil
	case *runtime.ArrayValue:
		return runtime.IntValue{Value: int64(len(value.Elements))}, nil
	case *runtime.ObjectValue:
		return runtime.IntValue{Value: int64(len(value.Fields))}, nil
	default:
		return nil, fmt.Errorf("len is not supported for %s", runtime.KindName(args[0]))
	}
}

func builtinTypeOf(args []runtime.Value) (runtime.Value, error) {
	return runtime.StringValue{Value: runtime.KindName(args[0])}, nil
}

func builtinArgv(args []runtime.Value) (runtime.Value, error) {
	items := make([]runtime.Value, 0, max(len(os.Args)-1, 0))
	for _, arg := range os.Args[1:] {
		items = append(items, runtime.StringValue{Value: arg})
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

func builtinChan(args []runtime.Value) (runtime.Value, error) {
	size := 0
	if len(args) > 1 {
		return nil, fmt.Errorf("chan expects at most 1 argument (buffer size), got %d", len(args))
	}
	if len(args) == 1 {
		if intVal, ok := args[0].(runtime.IntValue); ok {
			size = int(intVal.Value)
		} else {
			return nil, fmt.Errorf("chan buffer size must be an integer")
		}
	}
	return runtime.NewChannelValue(size), nil
}

func builtinPanic(args []runtime.Value) (runtime.Value, error) {
	return nil, fmt.Errorf("panic: %s", stringify(args[0]))
}

func builtinError(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("error expects 1 or 2 arguments, got %d", len(args))
	}
	err := &runtime.ErrorValue{Message: stringify(args[0])}
	if len(args) == 2 {
		switch cause := args[1].(type) {
		case runtime.NullValue:
			// no cause
		case *runtime.ErrorValue:
			err.Cause = cause
		default:
			err.Cause = &runtime.ErrorValue{Message: stringify(args[1])}
		}
	}
	return err, nil
}

func stringify(v runtime.Value) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}

func builtinSelect(args []runtime.Value) (runtime.Value, error) {
	cases, ok := args[0].(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("__select expects an array of case descriptors")
	}

	if len(cases.Elements) == 0 {
		return nil, fmt.Errorf("select with no cases")
	}

	type caseInfo struct {
		index   int
		isRecv  bool
		isSend  bool
		isElse  bool
		hasOk   bool
		channel *runtime.ChannelValue
		value   runtime.Value
	}

	caseInfos := make([]caseInfo, len(cases.Elements))
	reflectCases := make([]reflect.SelectCase, 0, len(cases.Elements))
	caseMap := make([]int, 0, len(cases.Elements))
	elseIdx := -1

	for i, elem := range cases.Elements {
		obj, ok := elem.(*runtime.ObjectValue)
		if !ok {
			continue
		}
		kind, _ := obj.Fields["kind"].(runtime.StringValue)

		switch kind.Value {
		case "recv":
			ch, ok := obj.Fields["chan"].(*runtime.ChannelValue)
			if !ok {
				return nil, fmt.Errorf("recv case %d: expected channel", i)
			}
			hasOk := false
			if h, ok := obj.Fields["hasOk"].(runtime.IntValue); ok {
				hasOk = h.Value != 0
			}
			caseInfos[i] = caseInfo{index: i, isRecv: true, channel: ch, hasOk: hasOk}
			reflectCases = append(reflectCases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch.RawChannel()),
			})
			caseMap = append(caseMap, i)
		case "send":
			ch, ok := obj.Fields["chan"].(*runtime.ChannelValue)
			if !ok {
				return nil, fmt.Errorf("send case %d: expected channel", i)
			}
			if ch.IsClosed() {
				continue
			}
			val := obj.Fields["value"]
			caseInfos[i] = caseInfo{index: i, isSend: true, channel: ch, value: val}
			reflectCases = append(reflectCases, reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: reflect.ValueOf(ch.RawChannel()),
				Send: reflect.ValueOf(val),
			})
			caseMap = append(caseMap, i)
		case "else":
			if elseIdx >= 0 {
				return nil, fmt.Errorf("multiple else/default cases in select")
			}
			elseIdx = i
			caseInfos[i] = caseInfo{index: i, isElse: true}
		default:
			return nil, fmt.Errorf("unknown select case kind: %s", kind.Value)
		}
	}

	if len(reflectCases) == 0 {
		if elseIdx < 0 {
			return nil, fmt.Errorf("select with no cases")
		}
		return &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"index": runtime.IntValue{Value: int64(elseIdx)},
			"value": runtime.NullValue{},
			"ok":    runtime.BoolValue{Value: true},
		}}, nil
	}

	if elseIdx >= 0 {
		reflectCases = append(reflectCases, reflect.SelectCase{
			Dir: reflect.SelectDefault,
		})
	}

	chosen, recv, recvOK := reflect.Select(reflectCases)

	if elseIdx >= 0 && chosen == len(reflectCases)-1 {
		return &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"index": runtime.IntValue{Value: int64(elseIdx)},
			"value": runtime.NullValue{},
			"ok":    runtime.BoolValue{Value: true},
		}}, nil
	}

	origIdx := caseMap[chosen]
	ci := caseInfos[origIdx]
	result := &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"index": runtime.IntValue{Value: int64(ci.index)},
		"ok":    runtime.BoolValue{Value: recvOK},
	}}
	if ci.isRecv && recvOK {
		result.Fields["value"] = recv.Interface().(runtime.Value)
	} else {
		result.Fields["value"] = runtime.NullValue{}
	}
	return result, nil
}

func builtinSatisfies(args []runtime.Value) (runtime.Value, error) {
	obj, ok := args[0].(*runtime.ObjectValue)
	if !ok {
		return runtime.BoolValue{Value: false}, nil
	}
	iface, ok := args[1].(*runtime.InterfaceValue)
	if !ok {
		return nil, fmt.Errorf("satisfies: second argument must be an interface")
	}
	for _, method := range iface.Methods {
		field, exists := obj.Fields[method.Name]
		if !exists {
			return runtime.BoolValue{Value: false}, nil
		}
		fn, ok := field.(*runtime.Closure)
		if !ok {
			_, okNative := field.(*runtime.NativeFunction)
			if !okNative {
				return runtime.BoolValue{Value: false}, nil
			}
		}
		_ = fn
	}
	return runtime.BoolValue{Value: true}, nil
}

func builtinTryCheck(args []runtime.Value) (runtime.Value, error) {
	_, isError := args[0].(*runtime.ErrorValue)
	return runtime.BoolValue{Value: isError}, nil
}

func builtinBuildClass(args []runtime.Value) (runtime.Value, error) {
	nameValue, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("__buildClass: name must be string")
	}

	var super *runtime.ClassValue
	switch value := args[1].(type) {
	case runtime.NullValue:
	case *runtime.ClassValue:
		super = value
	default:
		return nil, fmt.Errorf("__buildClass: super must be class or null")
	}

	var init *runtime.MethodDef
	switch value := args[2].(type) {
	case runtime.NullValue:
	case *runtime.MethodDef:
		init = value
	default:
		return nil, fmt.Errorf("__buildClass: init must be method or null")
	}

	methodObj, ok := args[3].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("__buildClass: methods must be object")
	}
	methods := make(map[string]*runtime.MethodDef, len(methodObj.Fields))
	for name, value := range methodObj.Fields {
		method, ok := value.(*runtime.MethodDef)
		if !ok {
			return nil, fmt.Errorf("__buildClass: method %s must be method", name)
		}
		methods[name] = method
	}

	return &runtime.ClassValue{
		Name:    nameValue.Value,
		Super:   super,
		Init:    init,
		Methods: methods,
	}, nil
}

func builtinMethodDef(args []runtime.Value) (runtime.Value, error) {
	callable := args[0]
	switch callable.(type) {
	case *runtime.Closure, *runtime.NativeFunction, *runtime.MethodProxy:
	default:
		return nil, fmt.Errorf("__methodDef: first argument must be callable")
	}
	implicitValue, ok := args[1].(runtime.BoolValue)
	if !ok {
		return nil, fmt.Errorf("__methodDef: second argument must be bool")
	}
	initValue, ok := args[2].(runtime.BoolValue)
	if !ok {
		return nil, fmt.Errorf("__methodDef: third argument must be bool")
	}
	nameValue, ok := args[3].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("__methodDef: fourth argument must be string")
	}
	return &runtime.MethodDef{
		Name:         nameValue.Value,
		Callable:     callable,
		ImplicitThis: implicitValue.Value,
		Init:         initValue.Value,
	}, nil
}

func builtinMethodProxy(args []runtime.Value) (runtime.Value, error) {
	method, ok := args[0].(*runtime.Closure)
	if !ok || method == nil {
		return nil, fmt.Errorf("__methodProxy: first argument must be function")
	}
	nameValue, ok := args[1].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("__methodProxy: second argument must be string")
	}
	initValue, ok := args[2].(runtime.BoolValue)
	if !ok {
		return nil, fmt.Errorf("__methodProxy: third argument must be bool")
	}
	return &runtime.MethodProxy{
		Name:   nameValue.Value,
		Method: method,
		Init:   initValue.Value,
	}, nil
}

func builtinSuperGet(args []runtime.Value) (runtime.Value, error) {
	super, ok := args[0].(*runtime.ClassValue)
	if !ok || super == nil {
		return nil, fmt.Errorf("__superGet: first argument must be superclass")
	}
	receiver, ok := args[1].(*runtime.ObjectValue)
	if !ok || receiver == nil {
		return nil, fmt.Errorf("__superGet: second argument must be object")
	}
	nameValue, ok := args[2].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("__superGet: third argument must be string")
	}

	var (
		method *runtime.MethodDef
		owner  *runtime.ClassValue
		found  bool
	)
	if nameValue.Value == "init" {
		method, owner, found = super.FindInitializer()
	} else {
		method, owner, found = super.FindMethod(nameValue.Value)
	}
	if !found {
		return nil, fmt.Errorf("undefined super method: %s", nameValue.Value)
	}

	return &runtime.BoundMethod{
		Name:     nameValue.Value,
		Receiver: receiver,
		Method:   method,
		Super:    owner.Super,
		Init:     method.Init,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
