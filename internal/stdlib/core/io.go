package core

import (
	"fmt"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
)

func LoadStdIOModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io",
		Path: "std.io",
		Exports: map[string]runtime.Value{
			"print":   &runtime.NativeFunction{Name: "print", Arity: -1, Fn: Print},
			"println": &runtime.NativeFunction{Name: "println", Arity: -1, Fn: Println},
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

func stringify(v runtime.Value) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
