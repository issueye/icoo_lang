package stdlib

import "icoo_lang/internal/runtime"

func LoadModule(spec string) (*runtime.Module, bool) {
	switch spec {
	case "std.io":
		return loadStdIOModule(), true
	case "std.time":
		return loadStdTimeModule(), true
	case "std.math":
		return loadStdMathModule(), true
	case "std.json":
		return loadStdJSONModule(), true
	case "std.fs":
		return loadStdFSModule(), true
	default:
	case "std.crypto":
		return loadStdCryptoModule(), true
	case "std.uuid":
		return loadStdUUIDModule(), true
	case "std.compress":
		return loadStdCompressModule(), true
		return nil, false
	}
}
