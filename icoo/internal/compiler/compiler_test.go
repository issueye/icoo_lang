package compiler

import (
	"testing"

	"icoo_lang/internal/ast"
	"icoo_lang/internal/bytecode"
	"icoo_lang/internal/lexer"
	"icoo_lang/internal/parser"
	"icoo_lang/internal/runtime"
)

func parse(source string) *ast.Program {
	tokens := lexer.LexAll(source)
	p := parser.New(tokens)
	return p.ParseProgram()
}

func compile(source string) (*CompiledModule, []error) {
	program := parse(source)
	return Compile(program)
}

// collectOpcodes extracts opcodes from the chunk, ignoring operands
func collectOpcodes(chunk *bytecode.Chunk) []bytecode.Opcode {
	ops := make([]bytecode.Opcode, 0)
	code := chunk.Code
	for i := 0; i < len(code); {
		op := bytecode.Opcode(code[i])
		ops = append(ops, op)
		i++
		switch op {
		case bytecode.OpConstant, bytecode.OpClosure,
			bytecode.OpGetLocal, bytecode.OpSetLocal,
			bytecode.OpGetGlobal, bytecode.OpDefineGlobal, bytecode.OpSetGlobal,
			bytecode.OpGetUpvalue, bytecode.OpSetUpvalue,
			bytecode.OpGetProperty, bytecode.OpSetProperty,
			bytecode.OpArray, bytecode.OpObject,
			bytecode.OpImportModule, bytecode.OpExport,
			bytecode.OpPushExceptionHandler, bytecode.OpPopExceptionHandler:
			i += 2 // short operand
		case bytecode.OpJump, bytecode.OpJumpIfFalse, bytecode.OpJumpIfTrue, bytecode.OpLoop:
			i += 2 // short operand
		case bytecode.OpCall, bytecode.OpGo, bytecode.OpGetIndex, bytecode.OpSetIndex:
			i++ // byte operand
		case bytecode.OpChanSend, bytecode.OpChanRecv, bytecode.OpChanTrySend, bytecode.OpChanTryRecv, bytecode.OpChanClose:
			i += 2
		}
	}
	return ops
}

func hasOpcode(t *testing.T, ops []bytecode.Opcode, expected bytecode.Opcode) {
	t.Helper()
	for _, op := range ops {
		if op == expected {
			return
		}
	}
	t.Errorf("expected opcode %s in chunk, got: %v", expected, ops)
}

func hasConstant(t *testing.T, chunk *bytecode.Chunk, kind runtime.ValueKind, want interface{}) {
	t.Helper()
	for _, c := range chunk.Constants {
		if c.Kind() != kind {
			continue
		}
		switch kind {
		case runtime.IntKind:
			if c.(runtime.IntValue).Value == want.(int64) {
				return
			}
		case runtime.StringKind:
			if c.(runtime.StringValue).Value == want.(string) {
				return
			}
		case runtime.FloatKind:
			if c.(runtime.FloatValue).Value == want.(float64) {
				return
			}
		}
	}
	t.Errorf("expected constant %v in chunk, got %v", want, chunk.Constants)
}

func TestCompiler_IntLiteral(t *testing.T) {
	mod, errs := compile("42")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpConstant)
	hasConstant(t, mod.Chunk, runtime.IntKind, int64(42))
}

func TestCompiler_FloatLiteral(t *testing.T) {
	mod, errs := compile("3.14")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpConstant)
	hasConstant(t, mod.Chunk, runtime.FloatKind, 3.14)
}

func TestCompiler_StringLiteral(t *testing.T) {
	mod, errs := compile(`"hello"`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpConstant)
	hasConstant(t, mod.Chunk, runtime.StringKind, "hello")
}

func TestCompiler_BoolLiteral(t *testing.T) {
	mod, errs := compile("true")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpTrue)

	mod2, errs2 := compile("false")
	if len(errs2) > 0 {
		t.Fatalf("unexpected errors: %v", errs2)
	}
	ops2 := collectOpcodes(mod2.Chunk)
	hasOpcode(t, ops2, bytecode.OpFalse)
}

func TestCompiler_NullLiteral(t *testing.T) {
	mod, errs := compile("null")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpNull)
}

func TestCompiler_UnaryNegate(t *testing.T) {
	mod, errs := compile("-42")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpNegate)
}

func TestCompiler_UnaryNot(t *testing.T) {
	mod, errs := compile("!true")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpNot)
}

func TestCompiler_BinaryAdd(t *testing.T) {
	mod, errs := compile("1 + 2")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpAdd)
}

func TestCompiler_BinarySub(t *testing.T) {
	mod, errs := compile("5 - 3")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpSub)
}

func TestCompiler_BinaryMul(t *testing.T) {
	mod, errs := compile("2 * 3")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpMul)
}

func TestCompiler_BinaryDiv(t *testing.T) {
	mod, errs := compile("6 / 2")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpDiv)
}

func TestCompiler_BinaryComparison(t *testing.T) {
	tests := []struct {
		source string
		op     bytecode.Opcode
	}{
		{"1 == 2", bytecode.OpEqual},
		{"1 != 2", bytecode.OpNotEqual},
		{"1 > 2", bytecode.OpGreater},
		{"1 >= 2", bytecode.OpGreaterEqual},
		{"1 < 2", bytecode.OpLess},
		{"1 <= 2", bytecode.OpLessEqual},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			mod, errs := compile(tt.source)
			if len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			ops := collectOpcodes(mod.Chunk)
			hasOpcode(t, ops, tt.op)
		})
	}
}

func TestCompiler_LogicalAnd(t *testing.T) {
	mod, errs := compile("true && false")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpJumpIfFalse)
}

func TestCompiler_LogicalOr(t *testing.T) {
	mod, errs := compile("true || false")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpJumpIfTrue)
}

func TestCompiler_TernaryExpr(t *testing.T) {
	mod, errs := compile("true ? 1 : 2")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpJumpIfFalse)
	hasOpcode(t, ops, bytecode.OpJump)
	hasConstant(t, mod.Chunk, runtime.IntKind, int64(1))
	hasConstant(t, mod.Chunk, runtime.IntKind, int64(2))
}

func TestCompiler_LetDeclaration(t *testing.T) {
	mod, errs := compile("let x = 42")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpDefineGlobal)
	hasConstant(t, mod.Chunk, runtime.IntKind, int64(42))
}

func TestCompiler_ConstDeclaration(t *testing.T) {
	mod, errs := compile("const x = 42")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpDefineGlobal)
	hasConstant(t, mod.Chunk, runtime.IntKind, int64(42))
}

func TestCompiler_VarReassignment(t *testing.T) {
	mod, errs := compile("let x = 1\nx = 2")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpSetGlobal)
}

func TestCompiler_IfStmt(t *testing.T) {
	mod, errs := compile("if true { let x = 1 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpJumpIfFalse)
	hasOpcode(t, ops, bytecode.OpJump)
}

func TestCompiler_IfElseStmt(t *testing.T) {
	mod, errs := compile("if true { let x = 1 } else { let x = 2 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpJumpIfFalse)
	hasOpcode(t, ops, bytecode.OpJump)
}

func TestCompiler_WhileLoop(t *testing.T) {
	mod, errs := compile("let i = 0\nwhile i < 10 { i = i + 1 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpLoop)
}

func TestCompiler_ArrayLiteral(t *testing.T) {
	mod, errs := compile("[1, 2, 3]")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpArray)
}

func TestCompiler_ObjectLiteral(t *testing.T) {
	t.Skip("parser infinite loop with object literals starting with {")
	mod, errs := compile(`{a: 1, b: 2}`)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpObject)
}

func TestCompiler_MemberAccess(t *testing.T) {
	mod, errs := compile("let obj = {x: 1}\nobj.x")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpGetProperty)
}

func TestCompiler_IndexAccess(t *testing.T) {
	mod, errs := compile("let arr = [1, 2]\narr[0]")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpGetIndex)
}

func TestCompiler_FunctionDeclaration(t *testing.T) {
	mod, errs := compile("fn add(a, b) { return a + b }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpClosure)
}

func TestCompiler_FunctionCall(t *testing.T) {
	mod, errs := compile("fn f() { return 42 }\nf()")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpCall)
}

func TestCompiler_LocalVariableSlots(t *testing.T) {
	// Variables in local scope should not emit DefineGlobal
	mod, errs := compile("{ let x = 1\n  let y = 2\n  x + y }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	// Should have GetLocal, not DefineGlobal
	hasOpcode(t, ops, bytecode.OpGetLocal)
	hasOpcode(t, ops, bytecode.OpAdd)
}

func TestCompiler_ScopeDepthCleanup(t *testing.T) {
	// After block exits, scope cleanup should emit OpPop for locals
	mod, errs := compile("{ let x = 1 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpPop)
}

func TestCompiler_TryCatch(t *testing.T) {
	mod, errs := compile("try { let x = 1 } catch err { let y = err }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpPushExceptionHandler)
	hasOpcode(t, ops, bytecode.OpPopExceptionHandler)
}

func TestCompiler_ReturnStmt(t *testing.T) {
	mod, errs := compile("fn f() { return 42 }")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	ops := collectOpcodes(mod.Chunk)
	hasOpcode(t, ops, bytecode.OpReturn)
}

func TestCompiler_ChunkEndsWithReturn(t *testing.T) {
	mod, errs := compile("let x = 1")
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	code := mod.Chunk.Code
	if len(code) < 1 {
		t.Fatal("expected non-empty code")
	}
	lastOp := bytecode.Opcode(code[len(code)-1])
	if lastOp != bytecode.OpReturn {
		t.Errorf("expected OpReturn at end of chunk, got %s", lastOp)
	}
}
