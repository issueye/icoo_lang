package stdlib

import (
	"fmt"
	"math"
	"time"

	"icoo_lang/internal/runtime"
)

func LoadModule(spec string) (*runtime.Module, bool) {
	switch spec {
	case "std.io":
		return &runtime.Module{
			Name: "std.io",
			Path: "std.io",
			Exports: map[string]runtime.Value{
				"print":   &runtime.NativeFunction{Name: "print", Arity: -1, Fn: builtinPrint},
				"println": &runtime.NativeFunction{Name: "println", Arity: -1, Fn: builtinPrintln},
			},
			Done: true,
		}, true
	case "std.time":
		return &runtime.Module{
			Name: "std.time",
			Path: "std.time",
			Exports: map[string]runtime.Value{
				"now":   &runtime.NativeFunction{Name: "now", Arity: 0, Fn: timeNow},
				"sleep": &runtime.NativeFunction{Name: "sleep", Arity: 1, Fn: timeSleep},
			},
			Done: true,
		}, true
	case "std.math":
		return &runtime.Module{
			Name: "std.math",
			Path: "std.math",
			Exports: map[string]runtime.Value{
				"abs":   &runtime.NativeFunction{Name: "abs", Arity: 1, Fn: mathAbs},
				"max":   &runtime.NativeFunction{Name: "max", Arity: 2, Fn: mathMax},
				"min":   &runtime.NativeFunction{Name: "min", Arity: 2, Fn: mathMin},
				"floor": &runtime.NativeFunction{Name: "floor", Arity: 1, Fn: mathFloor},
				"ceil":  &runtime.NativeFunction{Name: "ceil", Arity: 1, Fn: mathCeil},
			},
			Done: true,
		}, true
	default:
		return nil, false
	}
}

func timeNow(args []runtime.Value) (runtime.Value, error) {
	return runtime.IntValue{Value: time.Now().UnixMilli()}, nil
}

func timeSleep(args []runtime.Value) (runtime.Value, error) {
	ms, ok := args[0].(runtime.IntValue)
	if !ok {
		return nil, fmt.Errorf("sleep expects int milliseconds")
	}
	if ms.Value < 0 {
		return nil, fmt.Errorf("sleep expects non-negative milliseconds")
	}
	time.Sleep(time.Duration(ms.Value) * time.Millisecond)
	return runtime.NullValue{}, nil
}

func mathAbs(args []runtime.Value) (runtime.Value, error) {
	switch v := args[0].(type) {
	case runtime.IntValue:
		if v.Value < 0 {
			return runtime.IntValue{Value: -v.Value}, nil
		}
		return v, nil
	case runtime.FloatValue:
		return runtime.FloatValue{Value: math.Abs(v.Value)}, nil
	default:
		return nil, fmt.Errorf("abs expects int or float")
	}
}

func mathMax(args []runtime.Value) (runtime.Value, error) {
	return mathMinMax(args[0], args[1], true)
}

func mathMin(args []runtime.Value) (runtime.Value, error) {
	return mathMinMax(args[0], args[1], false)
}

func mathMinMax(left, right runtime.Value, wantMax bool) (runtime.Value, error) {
	lf, leftIsInt, ok := numericValue(left)
	if !ok {
		return nil, fmt.Errorf("numeric arguments required")
	}
	rf, rightIsInt, ok := numericValue(right)
	if !ok {
		return nil, fmt.Errorf("numeric arguments required")
	}
	pickLeft := lf <= rf
	if wantMax {
		pickLeft = lf >= rf
	}
	if pickLeft {
		if leftIsInt && rightIsInt {
			return left, nil
		}
		return runtime.FloatValue{Value: lf}, nil
	}
	if leftIsInt && rightIsInt {
		return right, nil
	}
	return runtime.FloatValue{Value: rf}, nil
}

func mathFloor(args []runtime.Value) (runtime.Value, error) {
	switch v := args[0].(type) {
	case runtime.IntValue:
		return v, nil
	case runtime.FloatValue:
		return runtime.FloatValue{Value: math.Floor(v.Value)}, nil
	default:
		return nil, fmt.Errorf("floor expects int or float")
	}
}

func mathCeil(args []runtime.Value) (runtime.Value, error) {
	switch v := args[0].(type) {
	case runtime.IntValue:
		return v, nil
	case runtime.FloatValue:
		return runtime.FloatValue{Value: math.Ceil(v.Value)}, nil
	default:
		return nil, fmt.Errorf("ceil expects int or float")
	}
}

func numericValue(v runtime.Value) (float64, bool, bool) {
	switch n := v.(type) {
	case runtime.IntValue:
		return float64(n.Value), true, true
	case runtime.FloatValue:
		return n.Value, false, true
	default:
		return 0, false, false
	}
}
