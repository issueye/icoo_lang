package format

import (
	"fmt"
	"os"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"

	toml "github.com/pelletier/go-toml/v2"
)

func LoadStdTOMLModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.data.toml",
	Path: "std.data.toml",
		Exports: map[string]runtime.Value{
			"encode":     &runtime.NativeFunction{Name: "encode", Arity: 1, Fn: tomlEncode},
			"decode":     &runtime.NativeFunction{Name: "decode", Arity: 1, Fn: tomlDecode},
			"fromFile":   &runtime.NativeFunction{Name: "fromFile", Arity: 1, Fn: tomlFromFile},
			"saveToFile": &runtime.NativeFunction{Name: "saveToFile", Arity: 2, Fn: tomlSaveToFile},
		},
		Done: true,
	}
}

func tomlEncode(args []runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(args[0])
	if err != nil {
		return nil, err
	}
	data, err := toml.Marshal(plain)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func tomlDecode(args []runtime.Value) (runtime.Value, error) {
	text, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("decode expects string")
	}
	var decoded any
	if err := toml.Unmarshal([]byte(text.Value), &decoded); err != nil {
		return nil, err
	}
	return utils.PlainToRuntimeValue(decoded), nil
}

func tomlFromFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("fromFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return tomlDecode([]runtime.Value{runtime.StringValue{Value: string(data)}})
}

func tomlSaveToFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("saveToFile", args[0])
	if err != nil {
		return nil, err
	}
	encoded, err := tomlEncode([]runtime.Value{args[1]})
	if err != nil {
		return nil, err
	}
	text := encoded.(runtime.StringValue)
	if err := os.WriteFile(path, []byte(text.Value), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}
