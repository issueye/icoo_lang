package bytecode

import "fmt"

type Opcode byte

const (
	OpConstant Opcode = iota
	OpNull
	OpTrue
	OpFalse
	OpPop
	OpDup

	OpGetLocal
	OpSetLocal
	OpGetGlobal
	OpDefineGlobal
	OpSetGlobal
	OpGetUpvalue
	OpSetUpvalue
	OpCloseUpvalue

	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpNegate
	OpNot
	OpEqual
	OpNotEqual
	OpGreater
	OpGreaterEqual
	OpLess
	OpLessEqual

	OpJump
	OpJumpIfFalse
	OpJumpIfTrue
	OpLoop

	OpCall
	OpClosure
	OpReturn

	OpArray
	OpObject
	OpGetProperty
	OpSetProperty
	OpGetIndex
	OpSetIndex

	OpImportModule
	OpExport

	OpPushExceptionHandler
	OpPopExceptionHandler
	OpThrow

	OpChanSend
	OpChanRecv
	OpChanTrySend
	OpChanTryRecv
	OpChanClose
)

func (op Opcode) String() string {
	switch op {
	case OpConstant:
		return "OpConstant"
	case OpNull:
		return "OpNull"
	case OpTrue:
		return "OpTrue"
	case OpFalse:
		return "OpFalse"
	case OpPop:
		return "OpPop"
	case OpDup:
		return "OpDup"
	case OpGetLocal:
		return "OpGetLocal"
	case OpSetLocal:
		return "OpSetLocal"
	case OpGetGlobal:
		return "OpGetGlobal"
	case OpDefineGlobal:
		return "OpDefineGlobal"
	case OpSetGlobal:
		return "OpSetGlobal"
	case OpGetUpvalue:
		return "OpGetUpvalue"
	case OpSetUpvalue:
		return "OpSetUpvalue"
	case OpCloseUpvalue:
		return "OpCloseUpvalue"
	case OpAdd:
		return "OpAdd"
	case OpSub:
		return "OpSub"
	case OpMul:
		return "OpMul"
	case OpDiv:
		return "OpDiv"
	case OpMod:
		return "OpMod"
	case OpNegate:
		return "OpNegate"
	case OpNot:
		return "OpNot"
	case OpEqual:
		return "OpEqual"
	case OpNotEqual:
		return "OpNotEqual"
	case OpGreater:
		return "OpGreater"
	case OpGreaterEqual:
		return "OpGreaterEqual"
	case OpLess:
		return "OpLess"
	case OpLessEqual:
		return "OpLessEqual"
	case OpJump:
		return "OpJump"
	case OpJumpIfFalse:
		return "OpJumpIfFalse"
	case OpJumpIfTrue:
		return "OpJumpIfTrue"
	case OpLoop:
		return "OpLoop"
	case OpCall:
		return "OpCall"
	case OpClosure:
		return "OpClosure"
	case OpReturn:
		return "OpReturn"
	case OpArray:
		return "OpArray"
	case OpObject:
		return "OpObject"
	case OpGetProperty:
		return "OpGetProperty"
	case OpSetProperty:
		return "OpSetProperty"
	case OpGetIndex:
		return "OpGetIndex"
	case OpSetIndex:
		return "OpSetIndex"
	case OpImportModule:
		return "OpImportModule"
	case OpExport:
		return "OpExport"
	case OpPushExceptionHandler:
		return "OpPushExceptionHandler"
	case OpPopExceptionHandler:
		return "OpPopExceptionHandler"
	case OpThrow:
		return "OpThrow"
	case OpChanSend:
		return "OpChanSend"
	case OpChanRecv:
		return "OpChanRecv"
	case OpChanTrySend:
		return "OpChanTrySend"
	case OpChanTryRecv:
		return "OpChanTryRecv"
	case OpChanClose:
		return "OpChanClose"
	default:
		return fmt.Sprintf("Opcode(%d)", byte(op))
	}
}
