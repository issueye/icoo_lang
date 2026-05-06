package core

import (
	"fmt"
	"sort"

	"icoo_lang/internal/runtime"
)

// LoadStdCoreObjectModule 加载 std.core.object 模块
func LoadStdCoreObjectModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.core.object",
		Path: "std.core.object",
		Exports: map[string]runtime.Value{
			"get":   &runtime.NativeFunction{Name: "get", Arity: -1, Fn: objectGet},
			"has":   &runtime.NativeFunction{Name: "has", Arity: 2, Fn: objectHas},
			"keys":  &runtime.NativeFunction{Name: "keys", Arity: 1, Fn: objectKeys},
			"merge": &runtime.NativeFunction{Name: "merge", Arity: -1, Fn: objectMerge},
		},
		Done: true,
	}
}

// objectGet 获取对象属性值
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

// objectHas 检查对象是否有指定属性
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

// objectKeys 获取对象所有键
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

// objectMerge 合并多个对象
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

// requireObjectLike 要求参数为对象或null
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

// requireObjectKey 要求参数为字符串键
func requireObjectKey(name string, value runtime.Value) (string, error) {
	key, ok := value.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s key must be string", name)
	}
	return key.Value, nil
}
