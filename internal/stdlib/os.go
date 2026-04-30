package stdlib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icoo_lang/internal/runtime"
)

func loadStdOSModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.os",
		Path: "std.os",
		Exports: map[string]runtime.Value{
			"args":      &runtime.NativeFunction{Name: "args", Arity: 0, Fn: osArgs},
			"cwd":       &runtime.NativeFunction{Name: "cwd", Arity: 0, Fn: osCwd},
			"tempDir":   &runtime.NativeFunction{Name: "tempDir", Arity: 0, Fn: osTempDir},
			"getEnv":    &runtime.NativeFunction{Name: "getEnv", Arity: 1, Fn: osGetEnv},
			"setEnv":    &runtime.NativeFunction{Name: "setEnv", Arity: 2, Fn: osSetEnv},
			"mkdirAll":  &runtime.NativeFunction{Name: "mkdirAll", Arity: 1, Fn: osMkdirAll},
			"remove":    &runtime.NativeFunction{Name: "remove", Arity: 1, Fn: osRemove},
			"removeAll": &runtime.NativeFunction{Name: "removeAll", Arity: 1, Fn: osRemoveAll},
		},
		Done: true,
	}
}

func osArgs(args []runtime.Value) (runtime.Value, error) {
	items := make([]runtime.Value, 0, len(os.Args))
	for _, arg := range os.Args {
		items = append(items, runtime.StringValue{Value: arg})
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

func osCwd(args []runtime.Value) (runtime.Value, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: dir}, nil
}

func osTempDir(args []runtime.Value) (runtime.Value, error) {
	return runtime.StringValue{Value: os.TempDir()}, nil
}

func osGetEnv(args []runtime.Value) (runtime.Value, error) {
	key, err := requireStringArg("getEnv", args[0])
	if err != nil {
		return nil, err
	}
	value, ok := os.LookupEnv(key)
	if !ok {
		return runtime.NullValue{}, nil
	}
	return runtime.StringValue{Value: value}, nil
}

func osSetEnv(args []runtime.Value) (runtime.Value, error) {
	key, err := requireStringArg("setEnv", args[0])
	if err != nil {
		return nil, err
	}
	value, err := requireStringArg("setEnv", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.Setenv(key, value); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func osMkdirAll(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("mkdirAll", args[0])
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("mkdirAll expects non-empty path")
	}
	if err := os.MkdirAll(filepath.Clean(path), 0o755); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func osRemove(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("remove", args[0])
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func osRemoveAll(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("removeAll", args[0])
	if err != nil {
		return nil, err
	}
	if path == "" || path == "." || strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("removeAll expects non-empty path")
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}
