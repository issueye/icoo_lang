package vm

import (
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/runtime"
)

func (vm *VM) Run(closure *runtime.Closure) (runtime.Value, error) {
	return vm.RunModule("", closure)
}

func (vm *VM) RunModule(path string, closure *runtime.Closure) (runtime.Value, error) {
	vm.frames = vm.frames[:0]
	vm.stack = vm.stack[:0]
	module := &runtime.Module{
		Path:    path,
		Exports: make(map[string]runtime.Value),
		Done:    true,
	}
	vm.lastModule = module
	vm.frames = append(vm.frames, CallFrame{
		Closure: closure,
		Module:  module,
		IP:      0,
		Base:    0,
	})
	return vm.runLoop()
}

func (vm *VM) runLoop() (runtime.Value, error) {
	for {
		frame := &vm.frames[len(vm.frames)-1]
		chunk := frame.Closure.Proto.Chunk.(*bytecode.Chunk)
		op := bytecode.Opcode(vm.readByte(frame, chunk))

		switch op {
		case bytecode.OpConstant:
			idx := vm.readShort(frame, chunk)
			value, err := chunk.GetConstant(idx)
			if err != nil {
				return nil, err
			}
			vm.Push(value)
		case bytecode.OpNull:
			vm.Push(runtime.NullValue{})
		case bytecode.OpTrue:
			vm.Push(runtime.BoolValue{Value: true})
		case bytecode.OpFalse:
			vm.Push(runtime.BoolValue{Value: false})
		case bytecode.OpPop:
			vm.Pop()
		case bytecode.OpDup:
			vm.Push(vm.Peek(0))
		case bytecode.OpGetLocal:
			slot := int(vm.readShort(frame, chunk))
			vm.Push(vm.stack[frame.Base+slot])
		case bytecode.OpSetLocal:
			slot := int(vm.readShort(frame, chunk))
			vm.stack[frame.Base+slot] = vm.Peek(0)
		case bytecode.OpGetGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			value, ok := vm.globals[name]
			if !ok {
				return nil, runtimeError("undefined global: %s", name)
			}
			vm.Push(value)
		case bytecode.OpDefineGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			vm.globals[name] = vm.Pop()
		case bytecode.OpSetGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			if _, ok := vm.globals[name]; !ok {
				return nil, runtimeError("undefined global: %s", name)
			}
			vm.globals[name] = vm.Peek(0)
		case bytecode.OpAdd:
			if err := vm.execAdd(); err != nil {
				return nil, err
			}
		case bytecode.OpSub, bytecode.OpMul, bytecode.OpDiv, bytecode.OpMod:
			if err := vm.execBinaryNumeric(op); err != nil {
				return nil, err
			}
		case bytecode.OpNegate:
			if err := vm.execNegate(); err != nil {
				return nil, err
			}
		case bytecode.OpNot:
			v := vm.Pop()
			vm.Push(runtime.BoolValue{Value: !runtime.IsTruthy(v)})
		case bytecode.OpEqual, bytecode.OpNotEqual, bytecode.OpGreater, bytecode.OpGreaterEqual, bytecode.OpLess, bytecode.OpLessEqual:
			if err := vm.execCompare(op); err != nil {
				return nil, err
			}
		case bytecode.OpJump:
			offset := int(vm.readShort(frame, chunk))
			frame.IP += offset
		case bytecode.OpJumpIfFalse:
			offset := int(vm.readShort(frame, chunk))
			if !runtime.IsTruthy(vm.Peek(0)) {
				frame.IP += offset
			}
		case bytecode.OpJumpIfTrue:
			offset := int(vm.readShort(frame, chunk))
			if runtime.IsTruthy(vm.Peek(0)) {
				frame.IP += offset
			}
		case bytecode.OpLoop:
			offset := int(vm.readShort(frame, chunk))
			frame.IP -= offset
		case bytecode.OpCall:
			argc := int(vm.readByte(frame, chunk))
			callee := vm.Peek(argc)
			if err := vm.CallValue(callee, argc); err != nil {
				return nil, err
			}
		case bytecode.OpClosure:
			idx := vm.readShort(frame, chunk)
			value, err := chunk.GetConstant(idx)
			if err != nil {
				return nil, err
			}
			vm.Push(value)
		case bytecode.OpImportModule:
			spec, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			if vm.loadModule == nil {
				return nil, runtimeError("module loader is not configured")
			}
			importerPath := ""
			if frame.Module != nil {
				importerPath = frame.Module.Path
			}
			module, err := vm.loadModule(importerPath, spec)
			if err != nil {
				return nil, err
			}
			vm.Push(module)
		case bytecode.OpExport:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			if frame.Module == nil {
				return nil, runtimeError("export used without module context")
			}
			if frame.Module.Exports == nil {
				frame.Module.Exports = make(map[string]runtime.Value)
			}
			frame.Module.Exports[name] = vm.Pop()
		case bytecode.OpReturn:
			result := vm.Pop()
			frameBase := frame.Base
			vm.frames = vm.frames[:len(vm.frames)-1]
			vm.stack = vm.stack[:frameBase]
			if len(vm.frames) == 0 {
				return result, nil
			}
			vm.Push(result)
		case bytecode.OpArray:
			count := int(vm.readShort(frame, chunk))
			items := make([]runtime.Value, count)
			for i := count - 1; i >= 0; i-- {
				items[i] = vm.Pop()
			}
			vm.Push(&runtime.ArrayValue{Elements: items})
		case bytecode.OpObject:
			count := int(vm.readShort(frame, chunk))
			fields := make(map[string]runtime.Value, count)
			for i := 0; i < count; i++ {
				value := vm.Pop()
				keyValue := vm.Pop()
				key, ok := keyValue.(runtime.StringValue)
				if !ok {
					return nil, runtimeError("object key must be string")
				}
				fields[key.Value] = value
			}
			vm.Push(&runtime.ObjectValue{Fields: fields})
		case bytecode.OpGetProperty:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			if err := vm.execGetProperty(name); err != nil {
				return nil, err
			}
		case bytecode.OpSetProperty:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				return nil, err
			}
			if err := vm.execSetProperty(name); err != nil {
				return nil, err
			}
		case bytecode.OpGetIndex:
			if err := vm.execGetIndex(); err != nil {
				return nil, err
			}
		case bytecode.OpSetIndex:
			if err := vm.execSetIndex(); err != nil {
				return nil, err
			}
		default:
			return nil, runtimeError("unsupported opcode: %s", op.String())
		}
	}
}

func (vm *VM) readByte(frame *CallFrame, chunk *bytecode.Chunk) byte {
	b := chunk.Code[frame.IP]
	frame.IP++
	return b
}

func (vm *VM) readShort(frame *CallFrame, chunk *bytecode.Chunk) uint16 {
	hi := uint16(chunk.Code[frame.IP])
	lo := uint16(chunk.Code[frame.IP+1])
	frame.IP += 2
	return (hi << 8) | lo
}

func (vm *VM) readStringConstant(frame *CallFrame, chunk *bytecode.Chunk) (string, error) {
	idx := vm.readShort(frame, chunk)
	value, err := chunk.GetConstant(idx)
	if err != nil {
		return "", err
	}
	str, ok := value.(runtime.StringValue)
	if !ok {
		return "", runtimeError("constant is not string")
	}
	return str.Value, nil
}
