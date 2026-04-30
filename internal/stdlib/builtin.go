package stdlib

import (
	"fmt"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/vm"
)

func RegisterBuiltins(machine *vm.VM) {
	machine.DefineBuiltin("print", &runtime.NativeFunction{Name: "print", Arity: -1, Fn: builtinPrint})
	machine.DefineBuiltin("println", &runtime.NativeFunction{Name: "println", Arity: -1, Fn: builtinPrintln})
	machine.DefineBuiltin("len", &runtime.NativeFunction{Name: "len", Arity: 1, Fn: builtinLen})
	machine.DefineBuiltin("typeOf", &runtime.NativeFunction{Name: "typeOf", Arity: 1, Fn: builtinTypeOf})
	machine.DefineBuiltin("panic", &runtime.NativeFunction{Name: "panic", Arity: 1, Fn: builtinPanic})
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

func builtinPanic(args []runtime.Value) (runtime.Value, error) {
	return nil, fmt.Errorf("panic: %s", stringify(args[0]))
}

func stringify(v runtime.Value) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
