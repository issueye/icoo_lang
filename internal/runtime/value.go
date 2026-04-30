package runtime

import (
	"fmt"
	"strings"
)

type ValueKind uint8

const (
	NullKind ValueKind = iota
	BoolKind
	IntKind
	FloatKind
	StringKind
	ArrayKind
	ObjectKind
	NativeFunctionKind
	ClosureKind
	ModuleKind
	ChannelKind
	ErrorKind
	IteratorKind
)

type Value interface {
	Kind() ValueKind
	String() string
}

type NullValue struct{}

func (NullValue) Kind() ValueKind { return NullKind }
func (NullValue) String() string  { return "null" }

type BoolValue struct {
	Value bool
}

func (v BoolValue) Kind() ValueKind { return BoolKind }
func (v BoolValue) String() string {
	if v.Value {
		return "true"
	}
	return "false"
}

type IntValue struct {
	Value int64
}

func (v IntValue) Kind() ValueKind { return IntKind }
func (v IntValue) String() string  { return fmt.Sprintf("%d", v.Value) }

type FloatValue struct {
	Value float64
}

func (v FloatValue) Kind() ValueKind { return FloatKind }
func (v FloatValue) String() string  { return fmt.Sprintf("%g", v.Value) }

type StringValue struct {
	Value string
}

func (v StringValue) Kind() ValueKind { return StringKind }
func (v StringValue) String() string  { return v.Value }

type StringIterator struct {
	Runes []rune
	Index int
}

func (v *StringIterator) Kind() ValueKind { return IteratorKind }
func (v *StringIterator) String() string  { return "<string_iterator>" }

type ArrayValue struct {
	Elements []Value
}

func (v *ArrayValue) Kind() ValueKind { return ArrayKind }
func (v *ArrayValue) String() string {
	parts := make([]string, 0, len(v.Elements))
	for _, elem := range v.Elements {
		parts = append(parts, elem.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type ArrayIterator struct {
	Array *ArrayValue
	Index int
}

func (v *ArrayIterator) Kind() ValueKind { return IteratorKind }
func (v *ArrayIterator) String() string  { return "<array_iterator>" }

type ObjectIterator struct {
	Items []Value
	Index int
}

func (v *ObjectIterator) Kind() ValueKind { return IteratorKind }
func (v *ObjectIterator) String() string  { return "<object_iterator>" }

type ObjectValue struct {
	Fields map[string]Value
}

func (v *ObjectValue) Kind() ValueKind { return ObjectKind }
func (v *ObjectValue) String() string {
	parts := make([]string, 0, len(v.Fields))
	for key, value := range v.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", key, value.String()))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

type NativeFunc func(args []Value) (Value, error)

type NativeFunction struct {
	Name  string
	Arity int
	Fn    NativeFunc
}

func (f *NativeFunction) Kind() ValueKind { return NativeFunctionKind }
func (f *NativeFunction) String() string  { return "<native fn " + f.Name + ">" }

type FunctionProto struct {
	Name         string
	Arity        int
	Chunk        any
	LocalCount   int
	UpvalueCount int
}

type Closure struct {
	Proto *FunctionProto
}

func (c *Closure) Kind() ValueKind { return ClosureKind }
func (c *Closure) String() string {
	name := "anonymous"
	if c != nil && c.Proto != nil && c.Proto.Name != "" {
		name = c.Proto.Name
	}
	return "<fn " + name + ">"
}

type Module struct {
	Name    string
	Path    string
	Exports map[string]Value
	Done    bool
}

func (m *Module) Kind() ValueKind { return ModuleKind }
func (m *Module) String() string {
	name := m.Path
	if name == "" {
		name = m.Name
	}
	return "<module " + name + ">"
}

type StackFrame struct {
	Function string
	File     string
	Line     int
	Native   bool
}

type ErrorValue struct {
	Message string
	Stack   []StackFrame
	Cause   *ErrorValue
}

func (e *ErrorValue) Kind() ValueKind { return ErrorKind }
func (e *ErrorValue) String() string  { return e.Message }
func (e *ErrorValue) Error() string {
	if e == nil {
		return ""
	}
	if stack := e.StackString(); stack != "" {
		return stack
	}
	return e.Message
}

func (e *ErrorValue) StackString() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	e.writeChainString(&b, false, map[*ErrorValue]struct{}{})
	return b.String()
}

func (e *ErrorValue) writeChainString(b *strings.Builder, caused bool, seen map[*ErrorValue]struct{}) {
	if e == nil {
		return
	}
	if _, ok := seen[e]; ok {
		if caused {
			b.WriteString("Caused by: <cycle>")
		} else {
			b.WriteString("<cycle>")
		}
		return
	}
	seen[e] = struct{}{}
	if caused {
		b.WriteString("Caused by: ")
	}
	if len(e.Stack) == 0 {
		b.WriteString(e.Message)
	} else {
		e.writeStackSegment(b)
	}
	if e.Cause != nil {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		e.Cause.writeChainString(b, true, seen)
	}
}

func (e *ErrorValue) writeStackSegment(b *strings.Builder) {
	b.WriteString(e.Message)
	for _, frame := range e.Stack {
		b.WriteString("\n  at ")
		name := frame.Function
		if name == "" {
			name = "<anonymous>"
		}
		b.WriteString(name)
		b.WriteString(" (")
		switch {
		case frame.Native:
			b.WriteString("native")
		case frame.File != "" && frame.Line > 0:
			b.WriteString(frame.File)
			b.WriteString(":")
			b.WriteString(fmt.Sprintf("%d", frame.Line))
		case frame.File != "":
			b.WriteString(frame.File)
		case frame.Line > 0:
			b.WriteString("unknown:")
			b.WriteString(fmt.Sprintf("%d", frame.Line))
		default:
			b.WriteString("unknown")
		}
		b.WriteString(")")
	}
}
