package vm

import (
	"sort"

	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

func (vm *VM) execAdd() error {
	right := vm.Pop()
	left := vm.Pop()

	switch l := left.(type) {
	case runtime.IntValue:
		switch r := right.(type) {
		case runtime.IntValue:
			vm.Push(runtime.IntValue{Value: l.Value + r.Value})
			return nil
		case runtime.FloatValue:
			vm.Push(runtime.FloatValue{Value: float64(l.Value) + r.Value})
			return nil
		}
	case runtime.FloatValue:
		switch r := right.(type) {
		case runtime.IntValue:
			vm.Push(runtime.FloatValue{Value: l.Value + float64(r.Value)})
			return nil
		case runtime.FloatValue:
			vm.Push(runtime.FloatValue{Value: l.Value + r.Value})
			return nil
		}
	case runtime.StringValue:
		if r, ok := right.(runtime.StringValue); ok {
			vm.Push(runtime.StringValue{Value: l.Value + r.Value})
			return nil
		}
	}

	return runtimeError("unsupported operands for +")
}

func (vm *VM) execBinaryNumeric(op bytecode.Opcode) error {
	right := vm.Pop()
	left := vm.Pop()

	lf, rf, ok := numericOperands(left, right)
	if !ok {
		return runtimeError("numeric operands required")
	}

	switch op {
	case bytecode.OpSub:
		vm.Push(numberResult(left, right, lf-rf))
	case bytecode.OpMul:
		vm.Push(numberResult(left, right, lf*rf))
	case bytecode.OpDiv:
		vm.Push(runtime.FloatValue{Value: lf / rf})
	case bytecode.OpMod:
		li, lok := left.(runtime.IntValue)
		ri, rok := right.(runtime.IntValue)
		if !lok || !rok {
			return runtimeError("integer operands required for %%")
		}
		vm.Push(runtime.IntValue{Value: li.Value % ri.Value})
	default:
		return runtimeError("unsupported numeric opcode")
	}
	return nil
}

func (vm *VM) execNegate() error {
	value := vm.Pop()
	switch v := value.(type) {
	case runtime.IntValue:
		vm.Push(runtime.IntValue{Value: -v.Value})
	case runtime.FloatValue:
		vm.Push(runtime.FloatValue{Value: -v.Value})
	default:
		return runtimeError("numeric operand required for unary -")
	}
	return nil
}

func (vm *VM) execCompare(op bytecode.Opcode) error {
	right := vm.Pop()
	left := vm.Pop()

	switch op {
	case bytecode.OpEqual:
		vm.Push(runtime.BoolValue{Value: runtime.ValueEqual(left, right)})
		return nil
	case bytecode.OpNotEqual:
		vm.Push(runtime.BoolValue{Value: !runtime.ValueEqual(left, right)})
		return nil
	}

	lf, rf, ok := numericOperands(left, right)
	if !ok {
		return runtimeError("numeric operands required for comparison")
	}

	switch op {
	case bytecode.OpGreater:
		vm.Push(runtime.BoolValue{Value: lf > rf})
	case bytecode.OpGreaterEqual:
		vm.Push(runtime.BoolValue{Value: lf >= rf})
	case bytecode.OpLess:
		vm.Push(runtime.BoolValue{Value: lf < rf})
	case bytecode.OpLessEqual:
		vm.Push(runtime.BoolValue{Value: lf <= rf})
	default:
		return runtimeError("unsupported comparison opcode")
	}
	return nil
}

func (vm *VM) execGetProperty(name string) error {
	obj := vm.Pop()
	switch value := obj.(type) {
	case runtime.StringValue:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "string.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					return &runtime.StringIterator{Runes: []rune(value.Value)}, nil
				},
			})
			return nil
		}
	case *runtime.StringIterator:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					return value, nil
				},
			})
			return nil
		}
		if name == "next" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.next",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					if value.Index >= len(value.Runes) {
						return &runtime.ObjectValue{Fields: map[string]runtime.Value{
							"key":   runtime.NullValue{},
							"value": runtime.NullValue{},
							"item":  runtime.NullValue{},
							"done":  runtime.BoolValue{Value: true},
						}}, nil
					}
					idx := value.Index
					item := runtime.StringValue{Value: string(value.Runes[idx])}
					value.Index++
					return &runtime.ObjectValue{Fields: map[string]runtime.Value{
						"key":   runtime.IntValue{Value: int64(idx)},
						"value": item,
						"item":  item,
						"done":  runtime.BoolValue{Value: false},
					}}, nil
				},
			})
			return nil
		}
	case *runtime.ArrayValue:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "array.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					return &runtime.ArrayIterator{Array: value}, nil
				},
			})
			return nil
		}
	case *runtime.ArrayIterator:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					return value, nil
				},
			})
			return nil
		}
		if name == "next" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.next",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					if value.Array == nil || value.Index >= len(value.Array.Elements) {
						return &runtime.ObjectValue{Fields: map[string]runtime.Value{
							"key":   runtime.NullValue{},
							"value": runtime.NullValue{},
							"item":  runtime.NullValue{},
							"done":  runtime.BoolValue{Value: true},
						}}, nil
					}
					idx := value.Index
					item := value.Array.Elements[idx]
					value.Index++
					return &runtime.ObjectValue{Fields: map[string]runtime.Value{
						"key":   runtime.IntValue{Value: int64(idx)},
						"value": item,
						"item":  item,
						"done":  runtime.BoolValue{Value: false},
					}}, nil
				},
			})
			return nil
		}
	case *runtime.ObjectIterator:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					return value, nil
				},
			})
			return nil
		}
		if name == "next" {
			vm.Push(&runtime.NativeFunction{
				Name:  "iterator.next",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					if value.Index >= len(value.Items) {
						return &runtime.ObjectValue{Fields: map[string]runtime.Value{
							"key":   runtime.NullValue{},
							"value": runtime.NullValue{},
							"item":  runtime.NullValue{},
							"done":  runtime.BoolValue{Value: true},
						}}, nil
					}
					item := value.Items[value.Index]
					value.Index++
					pair, _ := item.(*runtime.ObjectValue)
					return &runtime.ObjectValue{Fields: map[string]runtime.Value{
						"key":   pair.Fields["key"],
						"value": pair.Fields["value"],
						"item":  item,
						"done":  runtime.BoolValue{Value: false},
					}}, nil
				},
			})
			return nil
		}
	case *runtime.ObjectValue:
		field, ok := value.Fields[name]
		if ok {
			vm.Push(field)
			return nil
		}
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "object.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					keys := make([]string, 0, len(value.Fields))
					for key := range value.Fields {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					items := make([]runtime.Value, 0, len(keys))
					for _, key := range keys {
						items = append(items, &runtime.ObjectValue{Fields: map[string]runtime.Value{
							"key":   runtime.StringValue{Value: key},
							"value": value.Fields[key],
						}})
					}
					return &runtime.ObjectIterator{Items: items}, nil
				},
			})
			return nil
		}
		return runtimeError("undefined property: %s", name)
	case *runtime.Module:
		if name == "iter" {
			vm.Push(&runtime.NativeFunction{
				Name:  "module.iter",
				Arity: 0,
				Fn: func(args []runtime.Value) (runtime.Value, error) {
					keys := make([]string, 0, len(value.Exports))
					for key := range value.Exports {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					items := make([]runtime.Value, 0, len(keys))
					for _, key := range keys {
						items = append(items, &runtime.ObjectValue{Fields: map[string]runtime.Value{
							"key":   runtime.StringValue{Value: key},
							"value": value.Exports[key],
						}})
					}
					return &runtime.ObjectIterator{Items: items}, nil
				},
			})
			return nil
		}
		field, ok := value.Exports[name]
		if !ok {
			return runtimeError("undefined export: %s", name)
		}
		vm.Push(field)
		return nil
	}
	return runtimeError("property access not supported on %s", runtime.KindName(obj))
}

func (vm *VM) execSetProperty(name string) error {
	obj := vm.Pop()
	value := vm.Peek(0)
	object, ok := obj.(*runtime.ObjectValue)
	if !ok {
		return runtimeError("property assignment only supports object")
	}
	object.Fields[name] = value
	return nil
}

func (vm *VM) execGetIndex() error {
	index := vm.Pop()
	object := vm.Pop()

	switch value := object.(type) {
	case *runtime.ArrayValue:
		idx, ok := index.(runtime.IntValue)
		if !ok {
			return runtimeError("array index must be int")
		}
		if idx.Value < 0 || int(idx.Value) >= len(value.Elements) {
			return runtimeError("array index out of range")
		}
		vm.Push(value.Elements[idx.Value])
		return nil
	case *runtime.ObjectValue:
		key, ok := index.(runtime.StringValue)
		if !ok {
			return runtimeError("object index must be string")
		}
		field, exists := value.Fields[key.Value]
		if !exists {
			return runtimeError("undefined object key: %s", key.Value)
		}
		vm.Push(field)
		return nil
	default:
		return runtimeError("index access not supported on %s", runtime.KindName(object))
	}
}

func (vm *VM) execSetIndex() error {
	index := vm.Pop()
	object := vm.Pop()
	value := vm.Peek(0)

	switch target := object.(type) {
	case *runtime.ArrayValue:
		idx, ok := index.(runtime.IntValue)
		if !ok {
			return runtimeError("array index must be int")
		}
		if idx.Value < 0 || int(idx.Value) >= len(target.Elements) {
			return runtimeError("array index out of range")
		}
		target.Elements[idx.Value] = value
		return nil
	case *runtime.ObjectValue:
		key, ok := index.(runtime.StringValue)
		if !ok {
			return runtimeError("object index must be string")
		}
		target.Fields[key.Value] = value
		return nil
	default:
		return runtimeError("index assignment not supported on %s", runtime.KindName(object))
	}
}

func numericOperands(left, right runtime.Value) (float64, float64, bool) {
	lf, ok := toFloat(left)
	if !ok {
		return 0, 0, false
	}
	rf, ok := toFloat(right)
	if !ok {
		return 0, 0, false
	}
	return lf, rf, true
}

func toFloat(v runtime.Value) (float64, bool) {
	switch value := v.(type) {
	case runtime.IntValue:
		return float64(value.Value), true
	case runtime.FloatValue:
		return value.Value, true
	default:
		return 0, false
	}
}

func numberResult(left, right runtime.Value, value float64) runtime.Value {
	_, leftInt := left.(runtime.IntValue)
	_, rightInt := right.(runtime.IntValue)
	if leftInt && rightInt {
		return runtime.IntValue{Value: int64(value)}
	}
	return runtime.FloatValue{Value: value}
}
