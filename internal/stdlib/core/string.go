package core

import (
	"fmt"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdCoreStringModule 加载 std.core.string 模块
func LoadStdCoreStringModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.core.string",
		Path: "std.core.string",
		Exports: map[string]runtime.Value{
			"contains":  &runtime.NativeFunction{Name: "contains", Arity: 2, Fn: stringContains},
			"hasPrefix": &runtime.NativeFunction{Name: "hasPrefix", Arity: 2, Fn: stringHasPrefix},
			"hasSuffix": &runtime.NativeFunction{Name: "hasSuffix", Arity: 2, Fn: stringHasSuffix},
			"join":      &runtime.NativeFunction{Name: "join", Arity: 2, Fn: stringJoin},
			"replace":   &runtime.NativeFunction{Name: "replace", Arity: -1, Fn: stringReplace},
			"split":     &runtime.NativeFunction{Name: "split", Arity: 2, Fn: stringSplit},
			"trimSpace": &runtime.NativeFunction{Name: "trimSpace", Arity: 1, Fn: stringTrimSpace},
		},
		Done: true,
	}
}

func stringContains(args []runtime.Value) (runtime.Value, error) {
	text, needle, err := requireTwoStrings("contains", args)
	if err != nil {
		return nil, err
	}
	return runtime.BoolValue{Value: strings.Contains(text, needle)}, nil
}

func stringHasPrefix(args []runtime.Value) (runtime.Value, error) {
	text, prefix, err := requireTwoStrings("hasPrefix", args)
	if err != nil {
		return nil, err
	}
	return runtime.BoolValue{Value: strings.HasPrefix(text, prefix)}, nil
}

func stringHasSuffix(args []runtime.Value) (runtime.Value, error) {
	text, suffix, err := requireTwoStrings("hasSuffix", args)
	if err != nil {
		return nil, err
	}
	return runtime.BoolValue{Value: strings.HasSuffix(text, suffix)}, nil
}

func stringTrimSpace(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("trimSpace", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: strings.TrimSpace(text)}, nil
}

func stringSplit(args []runtime.Value) (runtime.Value, error) {
	text, sep, err := requireTwoStrings("split", args)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(text, sep)
	items := make([]runtime.Value, 0, len(parts))
	for _, part := range parts {
		items = append(items, runtime.StringValue{Value: part})
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

func stringJoin(args []runtime.Value) (runtime.Value, error) {
	arr, ok := args[0].(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("join expects array and separator")
	}
	sep, err := utils.RequireStringArg("join", args[1])
	if err != nil {
		return nil, err
	}
	parts := make([]string, 0, len(arr.Elements))
	for _, elem := range arr.Elements {
		text, ok := elem.(runtime.StringValue)
		if !ok {
			return nil, fmt.Errorf("join expects array of strings")
		}
		parts = append(parts, text.Value)
	}
	return runtime.StringValue{Value: strings.Join(parts, sep)}, nil
}

func stringReplace(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 3 || len(args) > 4 {
		return nil, fmt.Errorf("replace expects old, new, text, and optional count")
	}
	text, err := utils.RequireStringArg("replace", args[0])
	if err != nil {
		return nil, err
	}
	oldText, err := utils.RequireStringArg("replace", args[1])
	if err != nil {
		return nil, err
	}
	newText, err := utils.RequireStringArg("replace", args[2])
	if err != nil {
		return nil, err
	}
	count := -1
	if len(args) == 4 {
		intValue, ok := args[3].(runtime.IntValue)
		if !ok {
			return nil, fmt.Errorf("replace count must be int")
		}
		count = int(intValue.Value)
	}
	return runtime.StringValue{Value: strings.Replace(text, oldText, newText, count)}, nil
}

func requireTwoStrings(name string, args []runtime.Value) (string, string, error) {
	left, err := utils.RequireStringArg(name, args[0])
	if err != nil {
		return "", "", err
	}
	right, err := utils.RequireStringArg(name, args[1])
	if err != nil {
		return "", "", err
	}
	return left, right, nil
}
