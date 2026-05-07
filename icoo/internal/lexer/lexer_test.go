package lexer

import (
	"testing"

	"icoo_lang/internal/token"
)

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		input    string
		expected token.Type
	}{
		{"fn", token.Fn},
		{"return", token.Return},
		{"if", token.If},
		{"else", token.Else},
		{"for", token.For},
		{"while", token.While},
		{"match", token.Match},
		{"break", token.Break},
		{"continue", token.Continue},
		{"const", token.Const},
		{"let", token.Let},
		{"import", token.Import},
		{"export", token.Export},
		{"try", token.Try},
		{"catch", token.Catch},
		{"finally", token.Finally},
		{"throw", token.Throw},
		{"go", token.Go},
		{"select", token.Select},
		{"interface", token.Interface},
		{"type", token.TypeKw},
		{"class", token.Class},
		{"this", token.This},
		{"super", token.Super},
		{"null", token.Null},
		{"true", token.True},
		{"false", token.False},
		{"in", token.In},
		{"as", token.As},
		{"recv", token.Recv},
		{"send", token.Send},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := LexAll(tt.input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tokens[0].Type)
			}
			if tokens[1].Type != token.EOF {
				t.Errorf("expected EOF, got %s", tokens[1].Type)
			}
		})
	}
}

func TestLexer_Identifiers(t *testing.T) {
	inputs := []string{"x", "hello", "camelCase", "snake_case", "X123", "_private", "_"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens := LexAll(input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens")
			}
			if input == "_" {
				if tokens[0].Type != token.Underscore {
					t.Errorf("expected Underscore, got %s", tokens[0].Type)
				}
			} else {
				if tokens[0].Type != token.Ident {
					t.Errorf("expected Ident, got %s", tokens[0].Type)
				}
			}
			if tokens[0].Lexeme != input {
				t.Errorf("expected lexeme %q, got %q", input, tokens[0].Lexeme)
			}
		})
	}
}

func TestLexer_Integers(t *testing.T) {
	inputs := []string{"0", "42", "100", "999999"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens := LexAll(input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens")
			}
			if tokens[0].Type != token.Int {
				t.Errorf("expected Int, got %s", tokens[0].Type)
			}
			if tokens[0].Lexeme != input {
				t.Errorf("expected lexeme %q, got %q", input, tokens[0].Lexeme)
			}
		})
	}
}

func TestLexer_Floats(t *testing.T) {
	inputs := []string{"0.5", "3.14", "1.0", "42.42"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens := LexAll(input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens")
			}
			if tokens[0].Type != token.Float {
				t.Errorf("expected Float, got %s", tokens[0].Type)
			}
			if tokens[0].Lexeme != input {
				t.Errorf("expected lexeme %q, got %q", input, tokens[0].Lexeme)
			}
		})
	}
}

func TestLexer_Strings(t *testing.T) {
	inputs := []struct {
		input   string
		lexeme string
	}{
		{`"hello"`, `"hello"`},
		{`""`, `""`},
		{`"multi word string"`, `"multi word string"`},
		{`"escaped \" quote"`, `"escaped \" quote"`},
	}
	for _, tt := range inputs {
		t.Run(tt.input, func(t *testing.T) {
			tokens := LexAll(tt.input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens")
			}
			if tokens[0].Type != token.String {
				t.Errorf("expected String, got %s", tokens[0].Type)
			}
			if tokens[0].Lexeme != tt.lexeme {
				t.Errorf("expected lexeme %q, got %q", tt.lexeme, tokens[0].Lexeme)
			}
		})
	}
}

func TestLexer_UnterminatedString(t *testing.T) {
	tokens := LexAll(`"unterminated`)
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens")
	}
	if tokens[0].Type != token.Illegal {
		t.Errorf("expected Illegal for unterminated string, got %s", tokens[0].Type)
	}
}

func TestLexer_Comments(t *testing.T) {
	input := `// this is a comment
let x = 1`

	tokens := LexAll(input)
	// Should see: let, x, =, 1, EOF — comment skipped
	if len(tokens) < 5 {
		t.Fatalf("expected at least 5 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != token.Let {
		t.Errorf("expected Let, got %s", tokens[0].Type)
	}
}

func TestLexer_SingleLineCommentAtEnd(t *testing.T) {
	input := `let x = 1 // trailing`
	tokens := LexAll(input)
	// let, x, =, 1, EOF
	if tokens[0].Type != token.Let {
		t.Errorf("expected Let, got %s", tokens[0].Type)
	}
	if tokens[3].Type != token.Int {
		t.Errorf("expected Int, got %s", tokens[3].Type)
	}
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		input    string
		expected token.Type
	}{
		{"+", token.Plus},
		{"-", token.Minus},
		{"*", token.Star},
		{"/", token.Slash},
		{"%", token.Percent},
		{"!", token.Bang},
		{"=", token.Assign},
		{"==", token.Eq},
		{"!=", token.Neq},
		{"<-", token.Inherit},
		{"<", token.Lt},
		{"<=", token.Lte},
		{">", token.Gt},
		{">=", token.Gte},
		{"&&", token.AndAnd},
		{"||", token.OrOr},
		{"+=", token.PlusAssign},
		{"-=", token.MinusAssign},
		{"*=", token.StarAssign},
		{"/=", token.SlashAssign},
		{"=>", token.Arrow},
		{"?", token.Question},
		{"@", token.At},
		{".", token.Dot},
		{",", token.Comma},
		{":", token.Colon},
		{";", token.Semicolon},
		{"(", token.LParen},
		{")", token.RParen},
		{"{", token.LBrace},
		{"}", token.RBrace},
		{"[", token.LBracket},
		{"]", token.RBracket},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := LexAll(tt.input)
			if len(tokens) < 2 {
				t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestLexer_IllegalChars(t *testing.T) {
	tokens := LexAll("$ ~ `")
	// Should produce Illegal tokens for each
	nonEOF := 0
	for _, tok := range tokens {
		if tok.Type == token.EOF {
			break
		}
		nonEOF++
		if tok.Type != token.Illegal {
			t.Errorf("expected Illegal, got %s for %q", tok.Type, tok.Lexeme)
		}
	}
	if nonEOF != 3 {
		t.Errorf("expected 3 non-EOF tokens, got %d", nonEOF)
	}
}

func TestLexer_PositionTracking(t *testing.T) {
	input := "let\nx = 1"
	tokens := LexAll(input)

	// let at line 1
	if tokens[0].Span.Start.Line != 1 {
		t.Errorf("expected let line 1, got %d", tokens[0].Span.Start.Line)
	}
	// x at line 2
	if tokens[1].Span.Start.Line != 2 {
		t.Errorf("expected x line 2, got %d", tokens[1].Span.Start.Line)
	}
}

func TestLexer_MultiTokenExpression(t *testing.T) {
	input := "let result = 1 + 2 * 3"
	tokens := LexAll(input)

	expected := []token.Type{
		token.Let, token.Ident, token.Assign,
		token.Int, token.Plus, token.Int, token.Star, token.Int,
		token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token %d: expected %s, got %s (%q)", i, expected[i], tok.Type, tok.Lexeme)
		}
	}
}

func TestLexer_FunctionDeclaration(t *testing.T) {
	input := "fn add(a, b) {\n  return a + b\n}"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	expected := []token.Type{
		token.Fn, token.Ident, token.LParen, token.Ident, token.Comma,
		token.Ident, token.RParen, token.LBrace, token.Return, token.Ident,
		token.Plus, token.Ident, token.RBrace, token.EOF,
	}

	if len(types) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(types), types)
	}

	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("token %d: expected %s, got %s", i, tt, types[i])
		}
	}
}

func TestLexer_IfElseExpression(t *testing.T) {
	input := "if x > 0 { return true } else { return false }"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	expected := []token.Type{
		token.If, token.Ident, token.Gt, token.Int, token.LBrace,
		token.Return, token.True, token.RBrace, token.Else, token.LBrace,
		token.Return, token.False, token.RBrace, token.EOF,
	}

	if len(types) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(types), types)
	}

	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("token %d: expected %s, got %s", i, tt, types[i])
		}
	}
}

func TestLexer_TryCatchFinally(t *testing.T) {
	input := "try { } catch err { } finally { }"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	expected := []token.Type{
		token.Try, token.LBrace, token.RBrace, token.Catch, token.Ident,
		token.LBrace, token.RBrace, token.Finally, token.LBrace, token.RBrace,
		token.EOF,
	}

	if len(types) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(types), types)
	}

	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("token %d: expected %s, got %s", i, tt, types[i])
		}
	}
}

func TestLexer_GoChanSelect(t *testing.T) {
	input := "go fn() { send ch <- 1 }"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	if len(types) < 5 {
		t.Fatalf("expected at least 5 tokens, got %d: %v", len(types), types)
	}
	if types[0] != token.Go {
		t.Errorf("expected Go, got %s", types[0])
	}

	// Find send and chan tokens
	foundSend := false
	for _, tt := range types {
		if tt == token.Send {
			foundSend = true
		}
	}
	if !foundSend {
		t.Error("expected Send token in go/send expression")
	}
}

func TestLexer_ClassDeclaration(t *testing.T) {
	input := "class Dog <- Animal { fn bark() { } }"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	if types[0] != token.Class {
		t.Errorf("expected Class, got %s", types[0])
	}
	if types[1] != token.Ident || tokens[1].Lexeme != "Dog" {
		t.Errorf("expected Dog Ident, got %s (%q)", types[1], tokens[1].Lexeme)
	}
	if types[2] != token.Inherit {
		t.Errorf("expected <-, got %s", types[2])
	}
}

func TestLexer_TernaryAndTryExpr(t *testing.T) {
	input := "a ? b : c"
	tokens := LexAll(input)

	types := make([]token.Type, 0)
	for _, tok := range tokens {
		types = append(types, tok.Type)
	}

	expected := []token.Type{
		token.Ident, token.Question, token.Ident, token.Colon, token.Ident, token.EOF,
	}

	if len(types) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(types))
	}
	for i, tt := range expected {
		if types[i] != tt {
			t.Errorf("token %d: expected %s, got %s", i, tt, types[i])
		}
	}
}

func TestLexer_ImportPath(t *testing.T) {
	input := `import "@/lib/config.ic" as config`
	tokens := LexAll(input)

	if len(tokens) < 5 {
		t.Fatalf("expected at least 5 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != token.Import {
		t.Errorf("expected Import, got %s", tokens[0].Type)
	}
	if tokens[1].Type != token.String || tokens[1].Lexeme != `"@/lib/config.ic"` {
		t.Errorf("expected string path, got %s (%q)", tokens[1].Type, tokens[1].Lexeme)
	}
	if tokens[2].Type != token.As {
		t.Errorf("expected As, got %s", tokens[2].Type)
	}
}

func TestLexer_FromImportKeyword(t *testing.T) {
	input := `from std.io.console import println as log`
	tokens := LexAll(input)

	if len(tokens) < 10 {
		t.Fatalf("expected at least 10 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != token.From {
		t.Errorf("expected From, got %s", tokens[0].Type)
	}
	if tokens[1].Type != token.Ident || tokens[2].Type != token.Dot || tokens[3].Type != token.Ident {
		t.Errorf("expected module path tokens, got %s %s %s", tokens[1].Type, tokens[2].Type, tokens[3].Type)
	}
	if tokens[4].Type != token.Dot || tokens[5].Type != token.Ident {
		t.Errorf("expected module path continuation, got %s %s", tokens[4].Type, tokens[5].Type)
	}
	if tokens[6].Type != token.Import {
		t.Errorf("expected Import, got %s", tokens[6].Type)
	}
}
