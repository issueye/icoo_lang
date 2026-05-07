package utils

import (
	"fmt"
	"math"

	"icoo_lang/internal/runtime"
)

func RequireStringArg(name string, v runtime.Value) (string, error) {
	text, ok := v.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string argument", name)
	}
	return text.Value, nil
}

func RuntimeToPlainValue(v runtime.Value) (any, error) {
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
			plain, err := RuntimeToPlainValue(elem)
			if err != nil {
				return nil, err
			}
			items = append(items, plain)
		}
		return items, nil
	case *runtime.ObjectValue:
		obj := make(map[string]any, len(value.Fields))
		for key, field := range value.Fields {
			plain, err := RuntimeToPlainValue(field)
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

func PlainToRuntimeValue(v any) runtime.Value {
	switch value := v.(type) {
	case nil:
		return runtime.NullValue{}
	case bool:
		return runtime.BoolValue{Value: value}
	case int:
		return runtime.IntValue{Value: int64(value)}
	case int8:
		return runtime.IntValue{Value: int64(value)}
	case int16:
		return runtime.IntValue{Value: int64(value)}
	case int32:
		return runtime.IntValue{Value: int64(value)}
	case int64:
		return runtime.IntValue{Value: value}
	case uint:
		return runtime.IntValue{Value: int64(value)}
	case uint8:
		return runtime.IntValue{Value: int64(value)}
	case uint16:
		return runtime.IntValue{Value: int64(value)}
	case uint32:
		return runtime.IntValue{Value: int64(value)}
	case uint64:
		if value <= math.MaxInt64 {
			return runtime.IntValue{Value: int64(value)}
		}
		return runtime.FloatValue{Value: float64(value)}
	case float32:
		return runtime.FloatValue{Value: float64(value)}
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
			elems = append(elems, PlainToRuntimeValue(item))
		}
		return &runtime.ArrayValue{Elements: elems}
	case map[string]any:
		fields := make(map[string]runtime.Value, len(value))
		for key, item := range value {
			fields[key] = PlainToRuntimeValue(item)
		}
		return &runtime.ObjectValue{Fields: fields}
	case map[any]any:
		fields := make(map[string]runtime.Value, len(value))
		for key, item := range value {
			fields[fmt.Sprint(key)] = PlainToRuntimeValue(item)
		}
		return &runtime.ObjectValue{Fields: fields}
	default:
		return runtime.StringValue{Value: fmt.Sprint(value)}
	}
}
