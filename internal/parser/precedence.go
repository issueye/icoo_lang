package parser

import "icoo_lang/internal/token"

type Precedence int

const (
	PrecLowest Precedence = iota
	PrecAssign
	PrecOr
	PrecAnd
	PrecEquality
	PrecCompare
	PrecTerm
	PrecFactor
	PrecUnary
	PrecPostfix
)

func precedenceOf(tt token.Type) Precedence {
	switch tt {
	case token.Assign, token.PlusAssign, token.MinusAssign, token.StarAssign, token.SlashAssign:
		return PrecAssign
	case token.OrOr:
		return PrecOr
	case token.AndAnd:
		return PrecAnd
	case token.Eq, token.Neq:
		return PrecEquality
	case token.Lt, token.Lte, token.Gt, token.Gte:
		return PrecCompare
	case token.Plus, token.Minus:
		return PrecTerm
	case token.Star, token.Slash, token.Percent:
		return PrecFactor
	case token.LParen, token.Dot, token.LBracket:
		return PrecPostfix
	default:
		return PrecLowest
	}
}
