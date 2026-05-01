package core

import (
	"fmt"
	"time"

	"icoo_lang/internal/runtime"
)

func LoadStdTimeModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.time",
		Path: "std.time",
		Exports: map[string]runtime.Value{
			"now":   &runtime.NativeFunction{Name: "now", Arity: 0, Fn: timeNow},
			"sleep": &runtime.NativeFunction{Name: "sleep", Arity: 1, Fn: timeSleep},
		},
		Done: true,
	}
}

func timeNow(args []runtime.Value) (runtime.Value, error) {
	return runtime.IntValue{Value: time.Now().UnixMilli()}, nil
}

func timeSleep(args []runtime.Value) (runtime.Value, error) {
	ms, ok := args[0].(runtime.IntValue)
	if !ok {
		return nil, fmt.Errorf("sleep expects int milliseconds")
	}
	if ms.Value < 0 {
		return nil, fmt.Errorf("sleep expects non-negative milliseconds")
	}
	time.Sleep(time.Duration(ms.Value) * time.Millisecond)
	return runtime.NullValue{}, nil
}
