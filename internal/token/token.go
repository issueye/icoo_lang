package token

import "fmt"

type Type int

const (
	Illegal Type = iota
	EOF

	Ident
	Int
	Float
	String

	Assign      // =
	Plus        // +
	Minus       // -
	Star        // *
	Slash       // /
	Percent     // %
	Bang        // !
	Dot         // .
	Comma       // ,
	Colon       // :
	Semicolon   // ;
	LParen      // (
	RParen      // )
	LBrace      // {
	RBrace      // }
	LBracket    // [
	RBracket    // ]
	Underscore  // _

	Eq        // ==
	Neq       // !=
	Lt        // <
	Lte       // <=
	Gt        // >
	Gte       // >=
	AndAnd    // &&
	OrOr      // ||
	PlusAssign  // +=
	MinusAssign // -=
	StarAssign  // *=
	SlashAssign // /=
	Arrow       // =>
	Question    // ?

	Fn
	Return
	If
	Else
	For
	While
	Match
	Break
	Continue
	Const
	Let
	Import
	Export
	Try
	Catch
	Finally
	Throw
	Go
	Select
	Interface
	TypeKw
	Class
	This
	Super
	Null
	True
	False
	In
	As
	Recv
	Send
)

var Keywords = map[string]Type{
	"fn":        Fn,
	"return":    Return,
	"if":        If,
	"else":      Else,
	"for":       For,
	"while":     While,
	"match":     Match,
	"break":     Break,
	"continue":  Continue,
	"const":     Const,
	"let":       Let,
	"import":    Import,
	"export":    Export,
	"try":       Try,
	"catch":     Catch,
	"finally":   Finally,
	"throw":     Throw,
	"go":        Go,
	"select":    Select,
	"interface": Interface,
	"type":      TypeKw,
	"class":     Class,
	"this":      This,
	"super":     Super,
	"null":      Null,
	"true":      True,
	"false":     False,
	"in":        In,
	"as":        As,
	"recv":      Recv,
	"send":      Send,
}

type Position struct {
	Offset int
	Line   int
	Column int
}

type Span struct {
	Start Position
	End   Position
}

type Token struct {
	Type   Type
	Lexeme string
	Span   Span
}

func LookupIdent(ident string) Type {
	if ident == "_" {
		return Underscore
	}
	if tt, ok := Keywords[ident]; ok {
		return tt
	}
	return Ident
}

func (t Type) String() string {
	switch t {
	case Illegal:
		return "Illegal"
	case EOF:
		return "EOF"
	case Ident:
		return "Ident"
	case Int:
		return "Int"
	case Float:
		return "Float"
	case String:
		return "String"
	case Assign:
		return "Assign"
	case Plus:
		return "Plus"
	case Minus:
		return "Minus"
	case Star:
		return "Star"
	case Slash:
		return "Slash"
	case Percent:
		return "Percent"
	case Bang:
		return "Bang"
	case Dot:
		return "Dot"
	case Comma:
		return "Comma"
	case Colon:
		return "Colon"
	case Semicolon:
		return "Semicolon"
	case LParen:
		return "LParen"
	case RParen:
		return "RParen"
	case LBrace:
		return "LBrace"
	case RBrace:
		return "RBrace"
	case LBracket:
		return "LBracket"
	case RBracket:
		return "RBracket"
	case Underscore:
		return "Underscore"
	case Eq:
		return "Eq"
	case Neq:
		return "Neq"
	case Lt:
		return "Lt"
	case Lte:
		return "Lte"
	case Gt:
		return "Gt"
	case Gte:
		return "Gte"
	case AndAnd:
		return "AndAnd"
	case OrOr:
		return "OrOr"
	case PlusAssign:
		return "PlusAssign"
	case MinusAssign:
		return "MinusAssign"
	case StarAssign:
		return "StarAssign"
	case SlashAssign:
		return "SlashAssign"
	case Arrow:
		return "Arrow"
	case Question:
		return "Question"
	case Fn:
		return "Fn"
	case Return:
		return "Return"
	case If:
		return "If"
	case Else:
		return "Else"
	case For:
		return "For"
	case While:
		return "While"
	case Match:
		return "Match"
	case Break:
		return "Break"
	case Continue:
		return "Continue"
	case Const:
		return "Const"
	case Let:
		return "Let"
	case Import:
		return "Import"
	case Export:
		return "Export"
	case Try:
		return "Try"
	case Catch:
		return "Catch"
	case Finally:
		return "Finally"
	case Throw:
		return "Throw"
	case Go:
		return "Go"
	case Select:
		return "Select"
	case Class:
		return "Class"
	case This:
		return "This"
	case Super:
		return "Super"
	case Interface:
		return "Interface"
	case TypeKw:
		return "TypeKw"
	case Null:
		return "Null"
	case True:
		return "True"
	case False:
		return "False"
	case In:
		return "In"
	case As:
		return "As"
	case Recv:
		return "Recv"
	case Send:
		return "Send"
	default:
		return fmt.Sprintf("Type(%d)", int(t))
	}
}
