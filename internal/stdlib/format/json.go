package format

import (
	"encoding/json"
	"fmt"
	"os"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdJSONModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.json",
		Path: "std.json",
		Exports: map[string]runtime.Value{
			"encode":     &runtime.NativeFunction{Name: "encode", Arity: 1, Fn: jsonEncode},
			"decode":     &runtime.NativeFunction{Name: "decode", Arity: 1, Fn: jsonDecode},
			"fromFile":   &runtime.NativeFunction{Name: "fromFile", Arity: 1, Fn: jsonFromFile},
			"saveToFile": &runtime.NativeFunction{Name: "saveToFile", Arity: 2, Fn: jsonSaveToFile},
		},
		Done: true,
	}
}

func jsonEncode(args []runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(args[0])
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(plain)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func jsonDecode(args []runtime.Value) (runtime.Value, error) {
	text, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("decode expects string")
	}
	var decoded any
	if err := json.Unmarshal([]byte(text.Value), &decoded); err != nil {
		return nil, err
	}
	return utils.PlainToRuntimeValue(decoded), nil
}

func jsonFromFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("fromFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jsonDecode([]runtime.Value{runtime.StringValue{Value: string(data)}})
}

func jsonSaveToFile(args []runtime.Value) (runtime.Value, error) {
	path, err := utils.RequireStringArg("saveToFile", args[0])
	if err != nil {
		return nil, err
	}
	encoded, err := jsonEncode([]runtime.Value{args[1]})
	if err != nil {
		return nil, err
	}
	text := encoded.(runtime.StringValue)
	if err := os.WriteFile(path, []byte(text.Value), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}
