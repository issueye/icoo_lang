package io

import (
	"fmt"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
)

// LoadStdIOConsoleModule 加载 std.io.console 模块
func LoadStdIOConsoleModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io.console",
		Path: "std.io.console",
		Exports: map[string]runtime.Value{
			"print":   &runtime.NativeFunction{Name: "print", Arity: -1, Fn: Print},
			"println": &runtime.NativeFunction{Name: "println", Arity: -1, Fn: Println},
		},
		Done: true,
	}
}

// Print 打印输出（不换行）
func Print(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprint(os.Stdout, strings.Join(parts, ""))
	return runtime.NullValue{}, err
}

// Println 打印输出（换行）
func Println(args []runtime.Value) (runtime.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, stringify(arg))
	}
	_, err := fmt.Fprintln(os.Stdout, strings.Join(parts, " "))
	return runtime.NullValue{}, err
}

// stringify 将值转换为字符串
func stringify(v runtime.Value) string {
	if v == nil {
		return "nil"
	}
	return v.String()
}
