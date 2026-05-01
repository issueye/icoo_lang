package core

import (
	"fmt"
	"io"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

type ioHandle struct {
	name   string
	file   *os.File
	reader io.Reader
	writer io.Writer
	closer io.Closer
}

func LoadStdIOModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io",
		Path: "std.io",
		Exports: map[string]runtime.Value{
			"copy":       &runtime.NativeFunction{Name: "copy", Arity: 2, Fn: ioCopy},
			"openReader": &runtime.NativeFunction{Name: "openReader", Arity: 1, Fn: ioOpenReader},
			"openWriter": &runtime.NativeFunction{Name: "openWriter", Arity: 1, Fn: ioOpenWriter},
			"print":      &runtime.NativeFunction{Name: "print", Arity: -1, Fn: Print},
			"println":    &runtime.NativeFunction{Name: "println", Arity: -1, Fn: Println},
			"readAll":    &runtime.NativeFunction{Name: "readAll", Arity: 1, Fn: ioReadAll},
		},
		Done: true,
	}
}

func Print(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprint(os.Stdout, strings.Join(parts, ""))
	return runtime.NullValue{}, err
}

func Println(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprintln(os.Stdout, strings.Join(parts, " "))
	return runtime.NullValue{}, err
}

func ioOpenReader(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("openReader", args[0])
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return newIOHandle("reader", file, file, nil, file), nil
}

func ioOpenWriter(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("openWriter", args[0])
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return newIOHandle("writer", file, nil, file, file), nil
}

func ioCopy(args []runtime.Value) (runtime.Value, error) {
	dst, err := requireWriter("copy", args[0])
	if err != nil {
		return nil, err
	}
	src, err := requireReader("copy", args[1])
	if err != nil {
		return nil, err
	}
	written, err := io.Copy(dst, src)
	if err != nil {
		return nil, err
	}
	return runtime.IntValue{Value: written}, nil
}

func ioReadAll(args []runtime.Value) (runtime.Value, error) {
	reader, err := requireReader("readAll", args[0])
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func ioClose(args []runtime.Value) (runtime.Value, error) {
	handle, err := requireIOHandle("close", args[0])
	if err != nil {
		return nil, err
	}
	if handle.closer == nil {
		return runtime.NullValue{}, nil
	}
	if err := handle.closer.Close(); err != nil {
		return nil, err
	}
	handle.file = nil
	handle.reader = nil
	handle.writer = nil
	handle.closer = nil
	return runtime.NullValue{}, nil
}

func newIOHandle(name string, file *os.File, reader io.Reader, writer io.Writer, closer io.Closer) *runtime.ObjectValue {
	handle := &ioHandle{name: name, file: file, reader: reader, writer: writer, closer: closer}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"close": &runtime.NativeFunction{Name: name + ".close", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if handle.closer == nil {
				return runtime.NullValue{}, nil
			}
			if err := handle.closer.Close(); err != nil {
				return nil, err
			}
			handle.file = nil
			handle.reader = nil
			handle.writer = nil
			handle.closer = nil
			return runtime.NullValue{}, nil
		}},
		"kind": runtime.StringValue{Value: name},
		"raw":  &ioHandleValue{handle: handle},
	}}
}

func requireReader(name string, v runtime.Value) (io.Reader, error) {
	handle, err := requireIOHandle(name, v)
	if err != nil {
		return nil, err
	}
	if handle.reader == nil {
		return nil, fmt.Errorf("%s expects readable io handle", name)
	}
	return handle.reader, nil
}

func requireWriter(name string, v runtime.Value) (io.Writer, error) {
	handle, err := requireIOHandle(name, v)
	if err != nil {
		return nil, err
	}
	if handle.writer == nil {
		return nil, fmt.Errorf("%s expects writable io handle", name)
	}
	return handle.writer, nil
}

func requireIOHandle(name string, v runtime.Value) (*ioHandle, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects io handle", name)
	}
	raw, ok := obj.Fields["raw"].(*ioHandleValue)
	if !ok || raw.handle == nil {
		return nil, fmt.Errorf("%s expects io handle", name)
	}
	return raw.handle, nil
}

type ioHandleValue struct {
	handle *ioHandle
}

func (v *ioHandleValue) Kind() runtime.ValueKind { return runtime.ObjectKind }
func (v *ioHandleValue) String() string {
	if v == nil || v.handle == nil {
		return "<io nil>"
	}
	return "<io " + v.handle.name + ">"
}

func stringify(v runtime.Value) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
