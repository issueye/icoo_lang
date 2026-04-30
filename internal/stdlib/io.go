package stdlib

import "icoo_lang/internal/runtime"

func loadStdIOModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io",
		Path: "std.io",
		Exports: map[string]runtime.Value{
			"print":   &runtime.NativeFunction{Name: "print", Arity: -1, Fn: builtinPrint},
			"println": &runtime.NativeFunction{Name: "println", Arity: -1, Fn: builtinPrintln},
		},
		Done: true,
	}
}
