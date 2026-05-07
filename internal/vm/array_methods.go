package vm

import "icoo_lang/internal/runtime"

func arrayMethod(array *runtime.ArrayValue, name string) (*runtime.NativeFunction, bool) {
	switch name {
	case "map":
		return &runtime.NativeFunction{
			Name:  "array.map",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				result := make([]runtime.Value, 0, length)
				for i := 0; i < length; i++ {
					mapped, err := callArrayCallback(ctx, callback, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					result = append(result, mapped)
				}
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "filter":
		return &runtime.NativeFunction{
			Name:  "array.filter",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				result := make([]runtime.Value, 0, length)
				for i := 0; i < length; i++ {
					value := array.Elements[i]
					matched, err := callArrayCallback(ctx, callback, value, i, array)
					if err != nil {
						return nil, err
					}
					if runtime.IsTruthy(matched) {
						result = append(result, value)
					}
				}
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "find":
		return &runtime.NativeFunction{
			Name:  "array.find",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				for i := 0; i < length; i++ {
					value := array.Elements[i]
					matched, err := callArrayCallback(ctx, callback, value, i, array)
					if err != nil {
						return nil, err
					}
					if runtime.IsTruthy(matched) {
						return value, nil
					}
				}
				return runtime.NullValue{}, nil
			},
		}, true
	case "findIndex":
		return &runtime.NativeFunction{
			Name:  "array.findIndex",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				for i := 0; i < length; i++ {
					matched, err := callArrayCallback(ctx, callback, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					if runtime.IsTruthy(matched) {
						return runtime.IntValue{Value: int64(i)}, nil
					}
				}
				return runtime.IntValue{Value: -1}, nil
			},
		}, true
	case "some":
		return &runtime.NativeFunction{
			Name:  "array.some",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				for i := 0; i < length; i++ {
					matched, err := callArrayCallback(ctx, callback, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					if runtime.IsTruthy(matched) {
						return runtime.BoolValue{Value: true}, nil
					}
				}
				return runtime.BoolValue{Value: false}, nil
			},
		}, true
	case "every":
		return &runtime.NativeFunction{
			Name:  "array.every",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				for i := 0; i < length; i++ {
					matched, err := callArrayCallback(ctx, callback, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					if !runtime.IsTruthy(matched) {
						return runtime.BoolValue{Value: false}, nil
					}
				}
				return runtime.BoolValue{Value: true}, nil
			},
		}, true
	case "forEach":
		return &runtime.NativeFunction{
			Name:  "array.forEach",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				for i := 0; i < length; i++ {
					if _, err := callArrayCallback(ctx, callback, array.Elements[i], i, array); err != nil {
						return nil, err
					}
				}
				return runtime.NullValue{}, nil
			},
		}, true
	case "reduce":
		return &runtime.NativeFunction{
			Name:  "array.reduce",
			Arity: -1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				if len(args) < 1 || len(args) > 2 {
					return nil, runtimeError("array.reduce expects 1 or 2 arguments, got %d", len(args))
				}

				callback := args[0]
				length := len(array.Elements)
				if length == 0 && len(args) == 1 {
					return nil, runtimeError("array.reduce of empty array with no initial value")
				}

				start := 0
				accumulator := runtime.Value(runtime.NullValue{})
				if len(args) == 2 {
					accumulator = args[1]
				} else {
					accumulator = array.Elements[0]
					start = 1
				}

				for i := start; i < length; i++ {
					next, err := callArrayReduceCallback(ctx, callback, accumulator, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					accumulator = next
				}
				return accumulator, nil
			},
		}, true
	case "includes":
		return &runtime.NativeFunction{
			Name:  "array.includes",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				search := args[0]
				for _, value := range array.Elements {
					if runtime.ValueEqual(value, search) {
						return runtime.BoolValue{Value: true}, nil
					}
				}
				return runtime.BoolValue{Value: false}, nil
			},
		}, true
	case "indexOf":
		return &runtime.NativeFunction{
			Name:  "array.indexOf",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				search := args[0]
				for i, value := range array.Elements {
					if runtime.ValueEqual(value, search) {
						return runtime.IntValue{Value: int64(i)}, nil
					}
				}
				return runtime.IntValue{Value: -1}, nil
			},
		}, true
	case "append":
		return &runtime.NativeFunction{
			Name:  "array.append",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				result := make([]runtime.Value, 0, len(array.Elements)+1)
				result = append(result, array.Elements...)
				result = append(result, args[0])
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "concat":
		return &runtime.NativeFunction{
			Name:  "array.concat",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				other, ok := args[0].(*runtime.ArrayValue)
				if !ok {
					return nil, runtimeError("array.concat expects array argument")
				}
				result := make([]runtime.Value, 0, len(array.Elements)+len(other.Elements))
				result = append(result, array.Elements...)
				result = append(result, other.Elements...)
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "prepend":
		return &runtime.NativeFunction{
			Name:  "array.prepend",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				result := make([]runtime.Value, 0, len(array.Elements)+1)
				result = append(result, args[0])
				result = append(result, array.Elements...)
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "slice":
		return &runtime.NativeFunction{
			Name:  "array.slice",
			Arity: -1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				if len(args) > 2 {
					return nil, runtimeError("array.slice expects 0, 1, or 2 arguments, got %d", len(args))
				}

				length := len(array.Elements)
				start := 0
				end := length

				if len(args) >= 1 {
					value, ok := args[0].(runtime.IntValue)
					if !ok {
						return nil, runtimeError("array.slice start must be int")
					}
					start = normalizeSliceIndex(int(value.Value), length)
				}
				if len(args) == 2 {
					value, ok := args[1].(runtime.IntValue)
					if !ok {
						return nil, runtimeError("array.slice end must be int")
					}
					end = normalizeSliceIndex(int(value.Value), length)
				}

				if start > end {
					start = end
				}

				result := make([]runtime.Value, 0, end-start)
				result = append(result, array.Elements[start:end]...)
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "flatMap":
		return &runtime.NativeFunction{
			Name:  "array.flatMap",
			Arity: 1,
			CtxFn: func(ctx *runtime.NativeContext, args []runtime.Value) (runtime.Value, error) {
				callback := args[0]
				length := len(array.Elements)
				result := make([]runtime.Value, 0, length)
				for i := 0; i < length; i++ {
					mapped, err := callArrayCallback(ctx, callback, array.Elements[i], i, array)
					if err != nil {
						return nil, err
					}
					if nested, ok := mapped.(*runtime.ArrayValue); ok {
						result = append(result, nested.Elements...)
						continue
					}
					result = append(result, mapped)
				}
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "take":
		return &runtime.NativeFunction{
			Name:  "array.take",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				countValue, ok := args[0].(runtime.IntValue)
				if !ok {
					return nil, runtimeError("array.take count must be int")
				}
				length := len(array.Elements)
				count := normalizeTakeDropCount(int(countValue.Value), length)
				result := make([]runtime.Value, 0, count)
				result = append(result, array.Elements[:count]...)
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "drop":
		return &runtime.NativeFunction{
			Name:  "array.drop",
			Arity: 1,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				countValue, ok := args[0].(runtime.IntValue)
				if !ok {
					return nil, runtimeError("array.drop count must be int")
				}
				length := len(array.Elements)
				count := normalizeTakeDropCount(int(countValue.Value), length)
				result := make([]runtime.Value, 0, length-count)
				result = append(result, array.Elements[count:]...)
				return &runtime.ArrayValue{Elements: result}, nil
			},
		}, true
	case "first":
		return &runtime.NativeFunction{
			Name:  "array.first",
			Arity: 0,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				if len(array.Elements) == 0 {
					return runtime.NullValue{}, nil
				}
				return array.Elements[0], nil
			},
		}, true
	case "last":
		return &runtime.NativeFunction{
			Name:  "array.last",
			Arity: 0,
			Fn: func(args []runtime.Value) (runtime.Value, error) {
				if len(array.Elements) == 0 {
					return runtime.NullValue{}, nil
				}
				return array.Elements[len(array.Elements)-1], nil
			},
		}, true
	default:
		return nil, false
	}
}

func normalizeSliceIndex(index, length int) int {
	if index < 0 {
		index = length + index
	}
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}

func normalizeTakeDropCount(count, length int) int {
	if count < 0 {
		return 0
	}
	if count > length {
		return length
	}
	return count
}

func callArrayCallback(ctx *runtime.NativeContext, callback, value runtime.Value, index int, array *runtime.ArrayValue) (runtime.Value, error) {
	args, err := prepareArrayCallbackArgs(callback, []runtime.Value{
		value,
		runtime.IntValue{Value: int64(index)},
		array,
	})
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		if ctx.CallInline != nil {
			return ctx.CallInline(callback, args)
		}
		if ctx.CallDetached != nil {
			return ctx.CallDetached(callback, args)
		}
	}
	return nil, runtimeError("array callback invocation is unavailable")
}

func callArrayReduceCallback(ctx *runtime.NativeContext, callback, accumulator, value runtime.Value, index int, array *runtime.ArrayValue) (runtime.Value, error) {
	args, err := prepareArrayCallbackArgs(callback, []runtime.Value{
		accumulator,
		value,
		runtime.IntValue{Value: int64(index)},
		array,
	})
	if err != nil {
		return nil, err
	}
	if ctx != nil {
		if ctx.CallInline != nil {
			return ctx.CallInline(callback, args)
		}
		if ctx.CallDetached != nil {
			return ctx.CallDetached(callback, args)
		}
	}
	return nil, runtimeError("array callback invocation is unavailable")
}

func prepareArrayCallbackArgs(callback runtime.Value, available []runtime.Value) ([]runtime.Value, error) {
	switch fn := callback.(type) {
	case *runtime.Closure:
		if fn == nil || fn.Proto == nil {
			return nil, runtimeError("invalid callback")
		}
		if fn.Proto.Arity >= len(available) {
			return available, nil
		}
		return available[:fn.Proto.Arity], nil
	case *runtime.NativeFunction:
		if fn == nil {
			return nil, runtimeError("invalid callback")
		}
		if fn.Arity < 0 || fn.Arity >= len(available) {
			return available, nil
		}
		return available[:fn.Arity], nil
	case *runtime.BoundMethod:
		if fn == nil || fn.Method == nil || fn.Method.Callable == nil {
			return nil, runtimeError("invalid callback")
		}
		return prepareArrayCallbackArgs(fn.Method.Callable, available)
	case *runtime.MethodProxy:
		if fn == nil || fn.Method == nil || fn.Method.Proto == nil {
			return nil, runtimeError("invalid callback")
		}
		if fn.Method.Proto.Arity >= len(available) {
			return available, nil
		}
		return available[:fn.Method.Proto.Arity], nil
	default:
		return available, nil
	}
}
