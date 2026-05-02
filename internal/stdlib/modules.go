package stdlib

import (
	"icoo_lang/internal/runtime"
	stdcore "icoo_lang/internal/stdlib/core"
	stddata "icoo_lang/internal/stdlib/data"
	stddb "icoo_lang/internal/stdlib/database"
	express "icoo_lang/internal/stdlib/express"
	stdformat "icoo_lang/internal/stdlib/format"
	stdnet "icoo_lang/internal/stdlib/net"
	stdsystem "icoo_lang/internal/stdlib/system"
)

func LoadModule(spec string) (*runtime.Module, bool) {
	switch spec {
	case "std.io":
		return stdcore.LoadStdIOModule(), true
	case "std.time":
		return stdcore.LoadStdTimeModule(), true
	case "std.math":
		return stdcore.LoadStdMathModule(), true
	case "std.object":
		return stdcore.LoadStdObjectModule(), true
	case "std.db":
		return stddb.LoadStdDBModule(), true
	case "std.orm":
		return stddb.LoadStdORMModule(), true
	case "std.json":
		return stdformat.LoadStdJSONModule(), true
	case "std.yaml":
		return stdformat.LoadStdYAMLModule(), true
	case "std.toml":
		return stdformat.LoadStdTOMLModule(), true
	case "std.xml":
		return stdformat.LoadStdXMLModule(), true
	case "std.fs":
		return stdsystem.LoadStdFSModule(), true
	case "std.exec":
		return stdsystem.LoadStdExecModule(), true
	case "std.os":
		return stdsystem.LoadStdOSModule(), true
	case "std.host":
		return stdsystem.LoadStdHostModule(), true
	case "std.express":
		return express.LoadStdExpressModule(), true
	case "std.http.client":
		return stdnet.LoadStdNetHTTPClientModule(), true
	case "std.http.server":
		return stdnet.LoadStdNetHTTPServerModule(), true
	case "std.net.websocket.client":
		return stdnet.LoadStdNetWebSocketClientModule(), true
	case "std.net.websocket.server":
		return stdnet.LoadStdNetWebSocketServerModule(), true
	case "std.net.sse.client":
		return stdnet.LoadStdNetSSEClientModule(), true
	case "std.net.sse.server":
		return stdnet.LoadStdNetSSEServerModule(), true
	case "std.net.socket.client":
		return stdnet.LoadStdNetSocketClientModule(), true
	case "std.net.socket.server":
		return stdnet.LoadStdNetSocketServerModule(), true
	case "std.crypto":
		return stddata.LoadStdCryptoModule(), true
	case "std.uuid":
		return stddata.LoadStdUUIDModule(), true
	case "std.compress":
		return stddata.LoadStdCompressModule(), true
	default:
		return nil, false
	}
}
