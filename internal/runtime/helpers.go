package runtime

func IsTruthy(v Value) bool {
	switch value := v.(type) {
	case nil:
		return false
	case NullValue:
		return false
	case BoolValue:
		return value.Value
	case IntValue:
		return value.Value != 0
	case FloatValue:
		return value.Value != 0
	case StringValue:
		return value.Value != ""
	default:
		return true
	}
}

func ValueEqual(a, b Value) bool {
	switch left := a.(type) {
	case nil:
		return b == nil
	case NullValue:
		_, ok := b.(NullValue)
		return ok
	case BoolValue:
		right, ok := b.(BoolValue)
		return ok && left.Value == right.Value
	case IntValue:
		switch right := b.(type) {
		case IntValue:
			return left.Value == right.Value
		case FloatValue:
			return float64(left.Value) == right.Value
		default:
			return false
		}
	case FloatValue:
		switch right := b.(type) {
		case FloatValue:
			return left.Value == right.Value
		case IntValue:
			return left.Value == float64(right.Value)
		default:
			return false
		}
	case StringValue:
		right, ok := b.(StringValue)
		return ok && left.Value == right.Value
	default:
		return a == b
	}
}

func KindName(v Value) string {
	if v == nil {
		return "nil"
	}
	switch v.Kind() {
	case NullKind:
		return "null"
	case BoolKind:
		return "bool"
	case IntKind:
		return "int"
	case FloatKind:
		return "float"
	case StringKind:
		return "string"
	case ArrayKind:
		return "array"
	case ObjectKind:
		return "object"
	case NativeFunctionKind:
		return "native_function"
	case ClosureKind:
		return "function"
	case ModuleKind:
		return "module"
	case ChannelKind:
		return "channel"
	case ErrorKind:
		return "error"
	case IteratorKind:
		return "iterator"
	case InterfaceKind:
		return "interface"
	case ClassKind:
		return "class"
	case BoundMethodKind:
		return "function"
	case MethodProxyKind:
		return "function"
	case MethodDefKind:
		return "function"
	default:
		return "unknown"
	}
}
