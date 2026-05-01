package lexer

import (
	"unicode"

	"icoo_lang/internal/token"
)

type Lexer struct {
	src    []rune
	pos    int
	line   int
	column int
}

func New(src string) *Lexer {
	return &Lexer{
		src:    []rune(src),
		line:   1,
		column: 1,
	}
}

func LexAll(src string) []token.Token {
	l := New(src)
	tokens := make([]token.Token, 0, 64)
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	return tokens
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	start := l.position()
	ch := l.peek()

	switch {
	case ch == 0:
		return l.makeToken(token.EOF, "", start)
	case isIdentifierStart(ch):
		return l.lexIdentifierOrKeyword()
	case unicode.IsDigit(ch):
		return l.lexNumber()
	}

	switch ch {
	case '"':
		return l.lexString()
	case '(':
		l.advance()
		return l.makeToken(token.LParen, "(", start)
	case ')':
		l.advance()
		return l.makeToken(token.RParen, ")", start)
	case '{':
		l.advance()
		return l.makeToken(token.LBrace, "{", start)
	case '}':
		l.advance()
		return l.makeToken(token.RBrace, "}", start)
	case '[':
		l.advance()
		return l.makeToken(token.LBracket, "[", start)
	case ']':
		l.advance()
		return l.makeToken(token.RBracket, "]", start)
	case '.':
		l.advance()
		return l.makeToken(token.Dot, ".", start)
	case ',':
		l.advance()
		return l.makeToken(token.Comma, ",", start)
	case ':':
		l.advance()
		return l.makeToken(token.Colon, ":", start)
	case ';':
		l.advance()
		return l.makeToken(token.Semicolon, ";", start)
	case '+':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.PlusAssign, "+=", start)
		}
		return l.makeToken(token.Plus, "+", start)
	case '-':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.MinusAssign, "-=", start)
		}
		return l.makeToken(token.Minus, "-", start)
	case '*':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.StarAssign, "*=", start)
		}
		return l.makeToken(token.Star, "*", start)
	case '/':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.SlashAssign, "/=", start)
		}
		return l.makeToken(token.Slash, "/", start)
	case '%':
		l.advance()
		return l.makeToken(token.Percent, "%", start)
	case '?':
		l.advance()
		return l.makeToken(token.Question, "?", start)
	case '@':
		l.advance()
		return l.makeToken(token.At, "@", start)
	case '!':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.Neq, "!=", start)
		}
		return l.makeToken(token.Bang, "!", start)
	case '=':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.Eq, "==", start)
		}
		if l.peek() == '>' {
			l.advance()
			return l.makeToken(token.Arrow, "=>", start)
		}
		return l.makeToken(token.Assign, "=", start)
	case '<':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.Lte, "<=", start)
		}
		return l.makeToken(token.Lt, "<", start)
	case '>':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return l.makeToken(token.Gte, ">=", start)
		}
		return l.makeToken(token.Gt, ">", start)
	case '&':
		l.advance()
		if l.peek() == '&' {
			l.advance()
			return l.makeToken(token.AndAnd, "&&", start)
		}
		return l.makeToken(token.Illegal, "&", start)
	case '|':
		l.advance()
		if l.peek() == '|' {
			l.advance()
			return l.makeToken(token.OrOr, "||", start)
		}
		return l.makeToken(token.Illegal, "|", start)
	default:
		lexeme := string(ch)
		l.advance()
		return l.makeToken(token.Illegal, lexeme, start)
	}
}

func (l *Lexer) lexIdentifierOrKeyword() token.Token {
	start := l.position()
	startPos := l.pos
	for isIdentifierPart(l.peek()) {
		l.advance()
	}
	lexeme := string(l.src[startPos:l.pos])
	return l.makeToken(token.LookupIdent(lexeme), lexeme, start)
}

func (l *Lexer) lexNumber() token.Token {
	start := l.position()
	startPos := l.pos
	for unicode.IsDigit(l.peek()) {
		l.advance()
	}
	tt := token.Int
	if l.peek() == '.' && unicode.IsDigit(l.peekNext()) {
		tt = token.Float
		l.advance()
		for unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}
	lexeme := string(l.src[startPos:l.pos])
	return l.makeToken(tt, lexeme, start)
}

func (l *Lexer) lexString() token.Token {
	start := l.position()
	startPos := l.pos
	l.advance()
	for {
		ch := l.peek()
		if ch == 0 || ch == '\n' {
			lexeme := string(l.src[startPos:l.pos])
			return l.makeToken(token.Illegal, lexeme, start)
		}
		if ch == '"' {
			l.advance()
			break
		}
		if ch == '\\' {
			l.advance()
			if l.peek() != 0 {
				l.advance()
			}
			continue
		}
		l.advance()
	}
	lexeme := string(l.src[startPos:l.pos])
	return l.makeToken(token.String, lexeme, start)
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for unicode.IsSpace(l.peek()) {
			l.advance()
		}
		if l.peek() == '/' && l.peekNext() == '/' {
			for l.peek() != 0 && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		break
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func (l *Lexer) position() token.Position {
	return token.Position{
		Offset: l.pos,
		Line:   l.line,
		Column: l.column,
	}
}

func (l *Lexer) makeToken(tt token.Type, lexeme string, start token.Position) token.Token {
	return token.Token{
		Type:   tt,
		Lexeme: lexeme,
		Span: token.Span{
			Start: start,
			End:   l.position(),
		},
	}
}

func isIdentifierStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentifierPart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
