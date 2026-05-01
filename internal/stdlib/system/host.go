package system

import (
	"os"
	"runtime"

	langruntime "icoo_lang/internal/runtime"
)

func LoadStdHostModule() *langruntime.Module {
	return &langruntime.Module{
		Name: "std.host",
		Path: "std.host",
		Exports: map[string]langruntime.Value{
			"arch":     &langruntime.NativeFunction{Name: "arch", Arity: 0, Fn: hostArch},
			"goos":     &langruntime.NativeFunction{Name: "goos", Arity: 0, Fn: hostGOOS},
			"hostname": &langruntime.NativeFunction{Name: "hostname", Arity: 0, Fn: hostHostname},
			"numCPU":   &langruntime.NativeFunction{Name: "numCPU", Arity: 0, Fn: hostNumCPU},
			"pid":      &langruntime.NativeFunction{Name: "pid", Arity: 0, Fn: hostPID},
		},
		Done: true,
	}
}

func hostArch(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.StringValue{Value: runtime.GOARCH}, nil
}

func hostGOOS(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.StringValue{Value: runtime.GOOS}, nil
}

func hostHostname(args []langruntime.Value) (langruntime.Value, error) {
	name, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return langruntime.StringValue{Value: name}, nil
}

func hostNumCPU(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.IntValue{Value: int64(runtime.NumCPU())}, nil
}

func hostPID(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.IntValue{Value: int64(os.Getpid())}, nil
}
