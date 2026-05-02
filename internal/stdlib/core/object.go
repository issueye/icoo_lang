package core

import (
	"fmt"
	"sort"

	"icoo_lang/internal/runtime"
)

func LoadStdObjectModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.object",
		Path: "std.object",
		Exports: map[string]runtime.Value{
			"get":   &runtime.NativeFunction{Name: "get", Arity: -1, Fn: objectGet},
			"has":   &runtime.NativeFunction{Name: "has", Arity: 2, Fn: objectHas},
			"keys":  &runtime.NativeFunction{Name: "keys", Arity: 1, Fn: objectKeys},
			"merge": &runtime.NativeFunction{Name: "merge", Arity: -1, Fn: objectMerge},
		},
		Done: true,
	}
}

func objectGet(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("get expects object, key, and optional fallback")
	}
	obj, err := requireObjectLike("get", args[0])
	if err != nil {
		return nil, err
	}
	key, err := requireObjectKey("get", args[1])
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if len(args) == 3 {
			return args[2], nil
		}
		return runtime.NullValue{}, nil
	}
	if value, ok := obj.Fields[key]; ok {
		return value, nil
	}
	if len(args) == 3 {
		return args[2], nil
	}
	return runtime.NullValue{}, nil
}

func objectHas(args []runtime.Value) (runtime.Value, error) {
	obj, err := requireObjectLike("has", args[0])
	if err != nil {
		return nil, err
	}
	key, err := requireObjectKey("has", args[1])
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return runtime.BoolValue{Value: false}, nil
	}
	_, ok := obj.Fields[key]
	return runtime.BoolValue{Value: ok}, nil
}

func objectKeys(args []runtime.Value) (runtime.Value, error) {
	obj, err := requireObjectLike("keys", args[0])
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return &runtime.ArrayValue{}, nil
	}
	keys := make([]string, 0, len(obj.Fields))
	for key := range obj.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]runtime.Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, runtime.StringValue{Value: key})
	}
	return &runtime.ArrayValue{Elements: values}, nil
}

func objectMerge(args []runtime.Value) (runtime.Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("merge expects at least one argument")
	}
	fields := map[string]runtime.Value{}
	for _, arg := range args {
		obj, err := requireObjectLike("merge", arg)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			continue
		}
		for key, value := range obj.Fields {
			fields[key] = value
		}
	}
	return &runtime.ObjectValue{Fields: fields}, nil
}

func requireObjectLike(name string, value runtime.Value) (*runtime.ObjectValue, error) {
	switch v := value.(type) {
	case runtime.NullValue:
		return nil, nil
	case *runtime.ObjectValue:
		return v, nil
	default:
		return nil, fmt.Errorf("%s expects object or null", name)
	}
}

func requireObjectKey(name string, value runtime.Value) (string, error) {
	key, ok := value.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s key must be string", name)
	}
	return key.Value, nil
}
