package diag

import (
	"fmt"

	"icoo_lang/internal/token"
)

type Severity int

const (
	Error Severity = iota
	Warning
)

type Diagnostic struct {
	Severity Severity
	Message  string
	Span     token.Span
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%d:%d: %s", d.Span.Start.Line, d.Span.Start.Column, d.Message)
}
