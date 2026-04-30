package stdlib

import (
	"encoding/json"
	"fmt"
	"math"

	"icoo_lang/internal/runtime"
)

func loadStdJSONModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.json",
		Path: "std.json",
		Exports: map[string]runtime.Value{
			"encode": &runtime.NativeFunction{Name: "encode", Arity: 1, Fn: jsonEncode},
			"decode": &runtime.NativeFunction{Name: "decode", Arity: 1, Fn: jsonDecode},
		},
		Done: true,
	}
}

func jsonEncode(args []runtime.Value) (runtime.Value, error) {
	plain, err := toJSONValue(args[0])
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(plain)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func jsonDecode(args []runtime.Value) (runtime.Value, error) {
	text, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("decode expects string")
	}
	var decoded any
	if err := json.Unmarshal([]byte(text.Value), &decoded); err != nil {
		return nil, err
	}
	return fromJSONValue(decoded), nil
}

func toJSONValue(v runtime.Value) (any, error) {
	switch value := v.(type) {
	case nil:
		return nil, nil
	case runtime.NullValue:
		return nil, nil
	case runtime.BoolValue:
		return value.Value, nil
	case runtime.IntValue:
		return value.Value, nil
	case runtime.FloatValue:
		return value.Value, nil
	case runtime.StringValue:
		return value.Value, nil
	case *runtime.ArrayValue:
		items := make([]any, 0, len(value.Elements))
		for _, elem := range value.Elements {
			plain, err := toJSONValue(elem)
			if err != nil {
				return nil, err
			}
			items = append(items, plain)
		}
		return items, nil
	case *runtime.ObjectValue:
		obj := make(map[string]any, len(value.Fields))
		for key, field := range value.Fields {
			plain, err := toJSONValue(field)
			if err != nil {
				return nil, err
			}
			obj[key] = plain
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("encode is not supported for %s", runtime.KindName(v))
	}
}

func fromJSONValue(v any) runtime.Value {
	switch value := v.(type) {
	case nil:
		return runtime.NullValue{}
	case bool:
		return runtime.BoolValue{Value: value}
	case float64:
		if math.Trunc(value) == value {
			return runtime.IntValue{Value: int64(value)}
		}
		return runtime.FloatValue{Value: value}
	case string:
		return runtime.StringValue{Value: value}
	case []any:
		elems := make([]runtime.Value, 0, len(value))
		for _, item := range value {
			elems = append(elems, fromJSONValue(item))
		}
		return &runtime.ArrayValue{Elements: elems}
	case map[string]any:
		fields := make(map[string]runtime.Value, len(value))
		for key, item := range value {
			fields[key] = fromJSONValue(item)
		}
		return &runtime.ObjectValue{Fields: fields}
	default:
		return runtime.NullValue{}
	}
}
