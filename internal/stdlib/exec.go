package stdlib

import (
	"bytes"
	"fmt"
	"os/exec"

	"icoo_lang/internal/runtime"
)

func loadStdExecModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.exec",
		Path: "std.exec",
		Exports: map[string]runtime.Value{
			"run": &runtime.NativeFunction{Name: "run", Arity: -1, Fn: execRun},
		},
		Done: true,
	}
}

func execRun(args []runtime.Value) (runtime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("run expects command and optional args array")
	}
	command, err := requireStringArg("run", args[0])
	if err != nil {
		return nil, err
	}
	var argv []string
	if len(args) == 2 {
		argv, err = requireStringArrayArg("run", args[1])
		if err != nil {
			return nil, err
		}
	}

	cmd := exec.Command(command, argv...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	exitCode := int64(0)
	success := true
	if runErr != nil {
		success = false
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = int64(exitErr.ExitCode())
		} else {
			return nil, runErr
		}
	}

	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"ok":       runtime.BoolValue{Value: success},
		"code":     runtime.IntValue{Value: exitCode},
		"stdout":   runtime.StringValue{Value: stdout.String()},
		"stderr":   runtime.StringValue{Value: stderr.String()},
		"command":  runtime.StringValue{Value: command},
		"exitCode": runtime.IntValue{Value: exitCode},
	}}, nil
}

func requireStringArrayArg(name string, v runtime.Value) ([]string, error) {
	arr, ok := v.(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s expects array of strings", name)
	}
	out := make([]string, 0, len(arr.Elements))
	for _, elem := range arr.Elements {
		text, ok := elem.(runtime.StringValue)
		if !ok {
			return nil, fmt.Errorf("%s expects array of strings", name)
		}
		out = append(out, text.Value)
	}
	return out, nil
}
