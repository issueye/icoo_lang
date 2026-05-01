package stdlib

import (
	"icoo_lang/internal/runtime"
	express "icoo_lang/internal/stdlib/express"
	stdnet "icoo_lang/internal/stdlib/net"
)

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
	case "std.express":
		return express.LoadStdExpressModule(), true
	case "std.net.http.client":
		return stdnet.LoadStdNetHTTPClientModule(), true
	case "std.net.http.server":
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
		return loadStdCryptoModule(), true
	case "std.uuid":
		return loadStdUUIDModule(), true
	case "std.compress":
		return loadStdCompressModule(), true
	default:
		return nil, false
	}
}
