package format

import (
	"fmt"
	"os"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"

	"gopkg.in/yaml.v3"
)

func LoadStdYAMLModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.yaml",
		Path: "std.yaml",
		Exports: map[string]runtime.Value{
			"encode":     &runtime.NativeFunction{Name: "encode", Arity: 1, Fn: yamlEncode},
			"decode":     &runtime.NativeFunction{Name: "decode", Arity: 1, Fn: yamlDecode},
			"fromFile":   &runtime.NativeFunction{Name: "fromFile", Arity: 1, Fn: yamlFromFile},
			"saveToFile": &runtime.NativeFunction{Name: "saveToFile", Arity: 2, Fn: yamlSaveToFile},
		},
		Done: true,
	}
}

func yamlEncode(args []runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(args[0])
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(plain)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func yamlDecode(args []runtime.Value) (runtime.Value, error) {
	text, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("decode expects string")
	}
	var decoded any
	if err := yaml.Unmarshal([]byte(text.Value), &decoded); err != nil {
		return nil, err
	}
	return utils.PlainToRuntimeValue(decoded), nil
}

func yamlFromFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("fromFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return yamlDecode([]runtime.Value{runtime.StringValue{Value: string(data)}})
}

func yamlSaveToFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("saveToFile", args[0])
	if err != nil {
		return nil, err
	}
	encoded, err := yamlEncode([]runtime.Value{args[1]})
	if err != nil {
		return nil, err
	}
	text := encoded.(runtime.StringValue)
	if err := os.WriteFile(path, []byte(text.Value), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}
