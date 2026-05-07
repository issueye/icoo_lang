package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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
	env := map[string]string{}
	if len(args) == 1 {
		if options, ok := args[0].(*langruntime.ObjectValue); ok {
			parsedCommand, parsedArgs, parsedCwd, parsedShell, parsedEnv, err := parseExecOptions(options)
			if err != nil {
				return nil, err
			}
			command = parsedCommand
			argv = parsedArgs
			cwd = parsedCwd
			shell = parsedShell
			env = parsedEnv
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
	// Always start from the host process environment so PATH and other
	// machine-level variables remain available, then apply caller overrides.
	cmd.Env = mergeExecEnv(env)
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

func parseExecOptions(obj *langruntime.ObjectValue) (string, []string, string, bool, map[string]string, error) {
	commandValue, ok := obj.Fields["command"]
	if !ok {
		return "", nil, "", false, nil, fmt.Errorf("run options require command")
	}
	command, err := utils.RequireStringArg("run", commandValue)
	if err != nil {
		return "", nil, "", false, nil, err
	}
	argv := []string{}
	if argsValue, ok := obj.Fields["args"]; ok {
		argv, err = requireStringArrayArg("run", argsValue)
		if err != nil {
			return "", nil, "", false, nil, err
		}
	}
	cwd := ""
	if cwdValue, ok := obj.Fields["cwd"]; ok {
		cwd, err = utils.RequireStringArg("run", cwdValue)
		if err != nil {
			return "", nil, "", false, nil, err
		}
	}
	shell := false
	if shellValue, ok := obj.Fields["shell"]; ok {
		boolValue, ok := shellValue.(langruntime.BoolValue)
		if !ok {
			return "", nil, "", false, nil, fmt.Errorf("run shell must be bool")
		}
		shell = boolValue.Value
	}
	env := map[string]string{}
	if envValue, ok := obj.Fields["env"]; ok {
		env, err = requireStringMapArg("run", envValue)
		if err != nil {
			return "", nil, "", false, nil, err
		}
	}
	return command, argv, cwd, shell, env, nil
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

func requireStringMapArg(name string, v langruntime.Value) (map[string]string, error) {
	obj, ok := v.(*langruntime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s env must be object", name)
	}
	out := make(map[string]string, len(obj.Fields))
	for key, value := range obj.Fields {
		text, ok := value.(langruntime.StringValue)
		if !ok {
			return nil, fmt.Errorf("%s env must contain string values", name)
		}
		out[key] = text.Value
	}
	return out, nil
}

func mergeExecEnv(overrides map[string]string) []string {
	base := make(map[string]string, len(overrides)+len(os.Environ()))
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		base[key] = value
	}
	for key, value := range overrides {
		base[key] = value
	}
	env := make([]string, 0, len(base))
	for key, value := range base {
		env = append(env, key+"="+value)
	}
	return env
}
