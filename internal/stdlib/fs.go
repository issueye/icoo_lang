package stdlib

import (
	"fmt"
	"os"

	"icoo_lang/internal/runtime"
)

func loadStdFSModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.fs",
		Path: "std.fs",
		Exports: map[string]runtime.Value{
			"readFile":  &runtime.NativeFunction{Name: "readFile", Arity: 1, Fn: fsReadFile},
			"writeFile": &runtime.NativeFunction{Name: "writeFile", Arity: 2, Fn: fsWriteFile},
			"exists":    &runtime.NativeFunction{Name: "exists", Arity: 1, Fn: fsExists},
		},
		Done: true,
	}
}

func fsReadFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("readFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func fsWriteFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("writeFile", args[0])
	if err != nil {
		return nil, err
	}
	content, err := requireStringArg("writeFile", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsExists(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("exists", args[0])
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	if statErr == nil {
		return runtime.BoolValue{Value: true}, nil
	}
	if os.IsNotExist(statErr) {
		return runtime.BoolValue{Value: false}, nil
	}
	return nil, statErr
}

func requireStringArg(name string, v runtime.Value) (string, error) {
	text, ok := v.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string argument", name)
	}
	return text.Value, nil
}
