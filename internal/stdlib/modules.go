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
	case "std.yaml":
		return loadStdYAMLModule(), true
	case "std.toml":
		return loadStdTOMLModule(), true
	case "std.xml":
		return loadStdXMLModule(), true
	case "std.fs":
		return loadStdFSModule(), true
	case "std.exec":
		return loadStdExecModule(), true
	case "std.os":
		return loadStdOSModule(), true
	case "std.host":
		return loadStdHostModule(), true
	case "std.http":
		return loadStdHTTPModule(), true
	case "std.crypto":
		return loadStdCryptoModule(), true
	case "std.uuid":
		return loadStdUUIDModule(), true
	case "std.compress":
		return loadStdCompressModule(), true
	default:
		return nil, false
	}
}
