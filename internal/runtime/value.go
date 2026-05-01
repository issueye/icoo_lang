package runtime

import (
	"fmt"
	"strings"
	"sync"
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
	InterfaceKind
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

type InterfaceMethodSig struct {
	Name       string
	ParamCount int
}

type InterfaceValue struct {
	Name    string
	Methods []InterfaceMethodSig
}

func (v *InterfaceValue) Kind() ValueKind { return InterfaceKind }
func (v *InterfaceValue) String() string  { return "<interface " + v.Name + ">" }

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
type NativeFuncWithContext func(ctx *NativeContext, args []Value) (Value, error)

type NativeContext struct {
	CallDetached         func(callee Value, args []Value) (Value, error)
	CallDetachedWithArgs func(callee Value, args []Value) (Value, []Value, error)
}

type NativeFunction struct {
	Name  string
	Arity int
	Fn    NativeFunc
	CtxFn NativeFuncWithContext
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

type Upvalue struct {
	Location *Value
	Closed   Value
}

func (uv *Upvalue) Get() Value {
	if uv.Location != nil {
		return *uv.Location
	}
	return uv.Closed
}

func (uv *Upvalue) Set(v Value) {
	if uv.Location != nil {
		*uv.Location = v
	} else {
		uv.Closed = v
	}
}

type Closure struct {
	Proto    *FunctionProto
	Upvalues []*Upvalue
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

type ChannelValue struct {
	ch     chan Value
	closed bool
	mu     sync.Mutex
}

func NewChannelValue(size int) *ChannelValue {
	if size < 0 {
		size = 0
	}
	return &ChannelValue{ch: make(chan Value, size)}
}

func (c *ChannelValue) Kind() ValueKind { return ChannelKind }
func (c *ChannelValue) String() string {
	return fmt.Sprintf("<channel %d/%d>", len(c.ch), cap(c.ch))
}

func (c *ChannelValue) Send(v Value) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	c.ch <- v
	return true
}

func (c *ChannelValue) Recv() (Value, bool) {
	v, ok := <-c.ch
	if !ok {
		return NullValue{}, false
	}
	return v, true
}

func (c *ChannelValue) TrySend(v Value) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	select {
	case c.ch <- v:
		return true
	default:
		return false
	}
}

func (c *ChannelValue) TryRecv() (Value, bool) {
	select {
	case v, ok := <-c.ch:
		if !ok {
			return NullValue{}, false
		}
		return v, true
	default:
		return NullValue{}, false
	}
}

func (c *ChannelValue) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.ch)
	}
}

func (c *ChannelValue) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *ChannelValue) RawChannel() chan Value {
	return c.ch
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
