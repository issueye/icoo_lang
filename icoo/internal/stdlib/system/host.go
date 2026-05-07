package system

import (
	"os"
	"runtime"

	langruntime "icoo_lang/internal/runtime"
)

// LoadStdSysHostModule 加载 std.sys.host 模块
func LoadStdSysHostModule() *langruntime.Module {
	return &langruntime.Module{
		Name: "std.sys.host",
		Path: "std.sys.host",
		Exports: map[string]langruntime.Value{
			"arch":       &langruntime.NativeFunction{Name: "arch", Arity: 0, Fn: hostArch},
			"goos":       &langruntime.NativeFunction{Name: "goos", Arity: 0, Fn: hostGOOS},
			"hostname":   &langruntime.NativeFunction{Name: "hostname", Arity: 0, Fn: hostHostname},
			"numCPU":     &langruntime.NativeFunction{Name: "numCPU", Arity: 0, Fn: hostNumCPU},
			"pid":        &langruntime.NativeFunction{Name: "pid", Arity: 0, Fn: hostPID},
			"goroutines": &langruntime.NativeFunction{Name: "goroutines", Arity: 0, Fn: hostGoroutines},
			"memory":     &langruntime.NativeFunction{Name: "memory", Arity: 0, Fn: hostMemory},
			"runtime":    &langruntime.NativeFunction{Name: "runtime", Arity: 0, CtxFn: hostRuntimeStats},
			"gc":         &langruntime.NativeFunction{Name: "gc", Arity: 0, CtxFn: hostGC},
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

func hostGoroutines(args []langruntime.Value) (langruntime.Value, error) {
	return langruntime.IntValue{Value: int64(runtime.NumGoroutine())}, nil
}

func hostMemory(args []langruntime.Value) (langruntime.Value, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"allocBytes":        langruntime.IntValue{Value: int64(mem.Alloc)},
		"totalAllocBytes":   langruntime.IntValue{Value: int64(mem.TotalAlloc)},
		"sysBytes":          langruntime.IntValue{Value: int64(mem.Sys)},
		"heapAllocBytes":    langruntime.IntValue{Value: int64(mem.HeapAlloc)},
		"heapSysBytes":      langruntime.IntValue{Value: int64(mem.HeapSys)},
		"heapIdleBytes":     langruntime.IntValue{Value: int64(mem.HeapIdle)},
		"heapInuseBytes":    langruntime.IntValue{Value: int64(mem.HeapInuse)},
		"heapReleasedBytes": langruntime.IntValue{Value: int64(mem.HeapReleased)},
		"heapObjects":       langruntime.IntValue{Value: int64(mem.HeapObjects)},
		"numGC":             langruntime.IntValue{Value: int64(mem.NumGC)},
	}}, nil
}

func hostRuntimeStats(ctx *langruntime.NativeContext, args []langruntime.Value) (langruntime.Value, error) {
	if ctx != nil && ctx.RuntimeStats != nil {
		return ctx.RuntimeStats(), nil
	}
	return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{
		"numCPU":        langruntime.IntValue{Value: int64(runtime.NumCPU())},
		"goroutines":    langruntime.IntValue{Value: int64(runtime.NumGoroutine())},
		"memory":        mustHostMemoryValue(),
		"goroutinePool": &langruntime.ObjectValue{Fields: map[string]langruntime.Value{}},
	}}, nil
}

func hostGC(ctx *langruntime.NativeContext, args []langruntime.Value) (langruntime.Value, error) {
	if ctx != nil && ctx.CollectGarbage != nil {
		return ctx.CollectGarbage(), nil
	}
	runtime.GC()
	return hostMemory(nil)
}

func mustHostMemoryValue() langruntime.Value {
	value, err := hostMemory(nil)
	if err != nil {
		return &langruntime.ObjectValue{Fields: map[string]langruntime.Value{}}
	}
	return value
}
