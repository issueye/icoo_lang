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
	vm.handlers = vm.handlers[:0]
	vm.openUpvalues = make(map[int]*runtime.Upvalue)
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
	return vm.runLoopUntil(0)
}

func (vm *VM) runLoopUntil(stopDepth int) (runtime.Value, error) {
	for {
		frame := &vm.frames[len(vm.frames)-1]
		chunk := frame.Closure.Proto.Chunk.(*bytecode.Chunk)
		op := bytecode.Opcode(vm.readByte(frame, chunk))

		switch op {
		case bytecode.OpConstant:
			idx := vm.readShort(frame, chunk)
			value, err := chunk.GetConstant(idx)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
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
		case bytecode.OpSwap:
			top := vm.Pop()
			next := vm.Pop()
			vm.Push(top)
			vm.Push(next)
		case bytecode.OpGetLocal:
			slot := int(vm.readShort(frame, chunk))
			vm.Push(vm.stack[frame.Base+slot])
		case bytecode.OpSetLocal:
			slot := int(vm.readShort(frame, chunk))
			vm.stack[frame.Base+slot] = vm.Peek(0)
		case bytecode.OpGetGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.mu.RLock()
			value, ok := vm.globals[name]
			vm.mu.RUnlock()
			if !ok {
				if raised := vm.raise(runtimeError("undefined global: %s", name)); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.Push(value)
		case bytecode.OpDefineGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.mu.Lock()
			vm.globals[name] = vm.Pop()
			vm.mu.Unlock()
		case bytecode.OpSetGlobal:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.mu.Lock()
			if _, ok := vm.globals[name]; !ok {
				vm.mu.Unlock()
				if raised := vm.raise(runtimeError("undefined global: %s", name)); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.globals[name] = vm.Peek(0)
			vm.mu.Unlock()
		case bytecode.OpAdd:
			if err := vm.execAdd(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpSub, bytecode.OpMul, bytecode.OpDiv, bytecode.OpMod:
			if err := vm.execBinaryNumeric(op); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpNegate:
			if err := vm.execNegate(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpNot:
			v := vm.Pop()
			vm.Push(runtime.BoolValue{Value: !runtime.IsTruthy(v)})
		case bytecode.OpEqual, bytecode.OpNotEqual, bytecode.OpGreater, bytecode.OpGreaterEqual, bytecode.OpLess, bytecode.OpLessEqual:
			if err := vm.execCompare(op); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
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
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpClosure:
			idx := vm.readShort(frame, chunk)
			value, err := chunk.GetConstant(idx)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			template, ok := value.(*runtime.Closure)
			if !ok {
				if raised := vm.raise(runtimeError("closure constant is not a closure")); raised != nil {
					return nil, raised
				}
				continue
			}
			closure := &runtime.Closure{Proto: template.Proto}
			upvalueCount := closure.Proto.UpvalueCount
			if upvalueCount > 0 {
				closure.Upvalues = make([]*runtime.Upvalue, upvalueCount)
				for i := 0; i < upvalueCount; i++ {
					isLocal := vm.readByte(frame, chunk)
					uvIdx := int(vm.readByte(frame, chunk))
					if isLocal == 1 {
						closure.Upvalues[i] = vm.captureUpvalue(frame.Base + uvIdx)
					} else {
						currentClosure := frame.Closure
						if currentClosure == nil || uvIdx >= len(currentClosure.Upvalues) {
							if raised := vm.raise(runtimeError("invalid upvalue reference")); raised != nil {
								return nil, raised
							}
							continue
						}
						closure.Upvalues[i] = currentClosure.Upvalues[uvIdx]
					}
				}
			}
			vm.Push(closure)
		case bytecode.OpGetUpvalue:
			uvIdx := int(vm.readShort(frame, chunk))
			if frame.Closure == nil || uvIdx >= len(frame.Closure.Upvalues) {
				if raised := vm.raise(runtimeError("invalid upvalue index")); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.Push(frame.Closure.Upvalues[uvIdx].Get())
		case bytecode.OpSetUpvalue:
			uvIdx := int(vm.readShort(frame, chunk))
			if frame.Closure == nil || uvIdx >= len(frame.Closure.Upvalues) {
				if raised := vm.raise(runtimeError("invalid upvalue index")); raised != nil {
					return nil, raised
				}
				continue
			}
			frame.Closure.Upvalues[uvIdx].Set(vm.Peek(0))
		case bytecode.OpCloseUpvalue:
			uvIdx := int(vm.readShort(frame, chunk))
			if frame.Closure == nil || uvIdx >= len(frame.Closure.Upvalues) {
				continue
			}
			uv := frame.Closure.Upvalues[uvIdx]
			if uv.Location != nil {
				uv.Closed = *uv.Location
				for slot, openUV := range vm.openUpvalues {
					if openUV == uv {
						delete(vm.openUpvalues, slot)
						break
					}
				}
				uv.Location = nil
			}
		case bytecode.OpImportModule:
			spec, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			if vm.loadModule == nil {
				if raised := vm.raise(runtimeError("module loader is not configured")); raised != nil {
					return nil, raised
				}
				continue
			}
			importerPath := ""
			if frame.Module != nil {
				importerPath = frame.Module.Path
			}
			module, err := vm.loadModule(importerPath, spec)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			vm.Push(module)
		case bytecode.OpExport:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			if frame.Module == nil {
				if raised := vm.raise(runtimeError("export used without module context")); raised != nil {
					return nil, raised
				}
				continue
			}
			if frame.Module.Exports == nil {
				frame.Module.Exports = make(map[string]runtime.Value)
			}
			frame.Module.Exports[name] = vm.Pop()
		case bytecode.OpReturn:
			result := vm.Pop()
			frameBase := frame.Base
			if frameBase >= 0 && frameBase < len(vm.stack) {
				if bound, ok := vm.stack[frameBase].(*runtime.BoundMethod); ok && bound.Init {
					result = bound.Receiver
				}
			}
			vm.closeUpvalues(frameBase)
			vm.frames = vm.frames[:len(vm.frames)-1]
			for len(vm.handlers) > 0 && vm.handlers[len(vm.handlers)-1].FrameIndex >= len(vm.frames) {
				vm.handlers = vm.handlers[:len(vm.handlers)-1]
			}
			vm.stack = vm.stack[:frameBase]
			if len(vm.frames) == stopDepth {
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
					if raised := vm.raise(runtimeError("object key must be string")); raised != nil {
						return nil, raised
					}
					continue
				}
				fields[key.Value] = value
			}
			vm.Push(&runtime.ObjectValue{Fields: fields})
		case bytecode.OpGetProperty:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			if err := vm.execGetProperty(name); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpSetProperty:
			name, err := vm.readStringConstant(frame, chunk)
			if err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
				continue
			}
			if err := vm.execSetProperty(name); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpGetIndex:
			if err := vm.execGetIndex(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpSetIndex:
			if err := vm.execSetIndex(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpPushExceptionHandler:
			catchIP := int(vm.readShort(frame, chunk))
			vm.handlers = append(vm.handlers, ExceptionHandler{FrameIndex: len(vm.frames) - 1, StackDepth: len(vm.stack), CatchIP: catchIP})
		case bytecode.OpPopExceptionHandler:
			if len(vm.handlers) > 0 {
				vm.handlers = vm.handlers[:len(vm.handlers)-1]
			}
		case bytecode.OpThrow:
			value := vm.Pop()
			err, ok := value.(*runtime.ErrorValue)
			if !ok {
				err = &runtime.ErrorValue{Message: value.String()}
			}
			if raised := vm.raise(err); raised != nil {
				return nil, raised
			}
		case bytecode.OpGo:
			argc := int(vm.readByte(frame, chunk))
			callee := vm.Peek(argc)
			args := make([]runtime.Value, argc)
			start := len(vm.stack) - argc
			for i := 0; i < argc; i++ {
				args[i] = vm.stack[start+i]
			}
			vm.stack = vm.stack[:start-1]
			switch fn := callee.(type) {
			case *runtime.Closure:
				vm.Pool().Submit(fn, args)
			case *runtime.NativeFunction:
				vm.Pool().Submit(fn, args)
			default:
				if raised := vm.raise(runtimeError("go requires a callable value, got %s", runtime.KindName(callee))); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpChanSend:
			if err := vm.execChanSend(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpChanRecv:
			if err := vm.execChanRecv(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpChanTrySend:
			if err := vm.execChanTrySend(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpChanTryRecv:
			if err := vm.execChanTryRecv(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
			}
		case bytecode.OpChanClose:
			if err := vm.execChanClose(); err != nil {
				if raised := vm.raise(err); raised != nil {
					return nil, raised
				}
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
