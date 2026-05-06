package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"

	langruntime "icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdSysExecModule 加载 std.sys.exec 模块
func LoadStdSysExecModule() *langruntime.Module {
	return &langruntime.Module{
		Name: "std.sys.exec",
		Path: "std.sys.exec",
		Exports: map[string]langruntime.Value{
			"run": &langruntime.NativeFunction{Name: "run", Arity: -1, Fn: execRun},
		},
		Done: true,
	}
}

func execRun(args []langruntime.Value) (langruntime.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("run expects command and optional args array, or options object")
	}

	command := ""
	argv := []string{}
	cwd := ""
	shell := false
	if len(args) == 1 {
		if options, ok := args[0].(*langruntime.ObjectValue); ok {
			parsedCommand, parsedArgs, parsedCwd, parsedShell, err := parseExecOptions(options)
			if err != nil {
				return nil, err
			}
			command = parsedCommand
			argv = parsedArgs
			cwd = parsedCwd
			shell = parsedShell
		} else {
			parsedCommand, err := utils.RequireStringArg("run", args[0])
			if err != nil {
				return nil, err
			}
			command = parsedCommand
		}
	} else {
		var err error
		command, err = utils.RequireStringArg("run", args[0])
		if err != nil {
			return nil, err
		}
		argv, err = requireStringArrayArg("run", args[1])
		if err != nil {
			return nil, err
		}
	}

	if shell {
		command, argv = shellCommand(command)
	}

	cmd := exec.Command(command, argv...)
	if cwd != "" {
		cmd.Dir = cwd
	}
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

	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"ok":       langruntime.BoolValue{Value: success},
		"code":     langruntime.IntValue{Value: exitCode},
		"stdout":   langruntime.StringValue{Value: stdout.String()},
		"stderr":   langruntime.StringValue{Value: stderr.String()},
		"command":  langruntime.StringValue{Value: command},
		"exitCode": langruntime.IntValue{Value: exitCode},
	}}, nil
}

func parseExecOptions(obj *langruntime.ObjectValue) (string, []string, string, bool, error) {
	commandValue, ok := obj.Fields["command"]
	if !ok {
		return "", nil, "", false, fmt.Errorf("run options require command")
	}
	command, err := utils.RequireStringArg("run", commandValue)
	if err != nil {
		return "", nil, "", false, err
	}
	argv := []string{}
	if argsValue, ok := obj.Fields["args"]; ok {
		argv, err = requireStringArrayArg("run", argsValue)
		if err != nil {
			return "", nil, "", false, err
		}
	}
	cwd := ""
	if cwdValue, ok := obj.Fields["cwd"]; ok {
		cwd, err = utils.RequireStringArg("run", cwdValue)
		if err != nil {
			return "", nil, "", false, err
		}
	}
	shell := false
	if shellValue, ok := obj.Fields["shell"]; ok {
		boolValue, ok := shellValue.(langruntime.BoolValue)
		if !ok {
			return "", nil, "", false, fmt.Errorf("run shell must be bool")
		}
		shell = boolValue.Value
	}
	return command, argv, cwd, shell, nil
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", command}
	}
	return "sh", []string{"-lc", command}
}

func requireStringArrayArg(name string, v langruntime.Value) ([]string, error) {
	arr, ok := v.(*langruntime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s expects array of strings", name)
	}
	out := make([]string, 0, len(arr.Elements))
	for _, elem := range arr.Elements {
		text, ok := elem.(langruntime.StringValue)
		if !ok {
			return nil, fmt.Errorf("%s expects array of strings", name)
		}
		out = append(out, text.Value)
	}
	return out, nil
}
