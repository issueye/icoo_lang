package stdlib

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/vm"
)

func RegisterBuiltins(machine *vm.VM) {
	machine.DefineBuiltin("print", &runtime.NativeFunction{Name: "print", Arity: -1, Fn: builtinPrint})
	machine.DefineBuiltin("println", &runtime.NativeFunction{Name: "println", Arity: -1, Fn: builtinPrintln})
	machine.DefineBuiltin("len", &runtime.NativeFunction{Name: "len", Arity: 1, Fn: builtinLen})
	machine.DefineBuiltin("typeOf", &runtime.NativeFunction{Name: "typeOf", Arity: 1, Fn: builtinTypeOf})
	machine.DefineBuiltin("chan", &runtime.NativeFunction{Name: "chan", Arity: -1, Fn: builtinChan})
	machine.DefineBuiltin("panic", &runtime.NativeFunction{Name: "panic", Arity: 1, Fn: builtinPanic})
	machine.DefineBuiltin("error", &runtime.NativeFunction{Name: "error", Arity: -1, Fn: builtinError})
	machine.DefineBuiltin("__select", &runtime.NativeFunction{Name: "__select", Arity: 1, Fn: builtinSelect})
}

func builtinPrint(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprint(os.Stdout, strings.Join(parts, ""))
	return runtime.NullValue{}, err
}

func builtinPrintln(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprintln(os.Stdout, strings.Join(parts, " "))
	return runtime.NullValue{}, err
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
