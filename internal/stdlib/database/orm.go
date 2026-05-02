package database

import (
	"fmt"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

func LoadStdORMModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.orm",
		Path: "std.orm",
		Exports: map[string]runtime.Value{
			"model": &runtime.NativeFunction{Name: "model", Arity: 2, Fn: ormModel},
		},
		Done: true,
	}
}

func ormModel(args []runtime.Value) (runtime.Value, error) {
	handle, err := requireDBHandle("model", args[0])
	if err != nil {
		return nil, err
	}
	if _, err := handle.requireDB("model"); err != nil {
		return nil, err
	}

	schema, ok := args[1].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("model expects schema object")
	}

	tableName := ""
	if nameValue, ok := schema.Fields["name"]; ok {
		tableName, err = utils.RequireStringArg("model", nameValue)
		if err != nil {
			return nil, err
		}
	}
	if tableName == "" {
		if tableValue, ok := schema.Fields["table"]; ok {
			tableName, err = utils.RequireStringArg("model", tableValue)
			if err != nil {
				return nil, err
			}
		}
	}
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return nil, fmt.Errorf("model schema requires non-empty name")
	}

	columnsValue, ok := schema.Fields["columns"]
	if !ok {
		return nil, fmt.Errorf("model schema requires columns")
	}
	columnsObject, ok := columnsValue.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("model columns must be object")
	}
	if len(columnsObject.Fields) == 0 {
		return nil, fmt.Errorf("model columns must not be empty")
	}

	columnDefs := make(map[string]string, len(columnsObject.Fields))
	for key, value := range columnsObject.Fields {
		definition, err := utils.RequireStringArg("model", value)
		if err != nil {
			return nil, fmt.Errorf("model column %s must be string: %w", key, err)
		}
		definition = strings.TrimSpace(definition)
		if definition == "" {
			return nil, fmt.Errorf("model column %s must not be empty", key)
		}
		columnDefs[key] = definition
	}

	return newORMQueryObject(&ormQueryState{
		handle:     handle,
		table:      tableName,
		columnDefs: columnDefs,
		selectExpr: "*",
	}), nil
}
