package stdlib

import (
	"icoo_lang/internal/runtime"
	stdcore "icoo_lang/internal/stdlib/core"
	stddata "icoo_lang/internal/stdlib/data"
	stddb "icoo_lang/internal/stdlib/database"
	stdformat "icoo_lang/internal/stdlib/format"
	stdio "icoo_lang/internal/stdlib/io"
	stdnet "icoo_lang/internal/stdlib/net"
	stdsystem "icoo_lang/internal/stdlib/system"
	stdweb "icoo_lang/internal/stdlib/web"
)

// LoadModule 根据模块路径加载标准库模块
// 采用三级命名规范：std.<领域>.<子领域>.<功能>
func LoadModule(spec string) (*runtime.Module, bool) {
	switch spec {
	// ==================== std.core - 核心基础 ====================
	case "std.core.object":
		return stdcore.LoadStdCoreObjectModule(), true
	case "std.core.observe":
		return stdcore.LoadStdCoreObserveModule(), true
	case "std.core.service":
		return stdcore.LoadStdCoreServiceModule(), true
	case "std.core.cache":
		return stdcore.LoadStdCoreCacheModule(), true

	// ==================== std.io - 输入输出 ====================
	case "std.io.console":
		return stdio.LoadStdIOConsoleModule(), true
	case "std.io.fs":
		return stdio.LoadStdIOFSModule(), true
	case "std.io.template":
		return stdio.LoadStdIOTemplateModule(), true

	// ==================== std.time - 时间日期 ====================
	case "std.time.basic":
		return stdcore.LoadStdTimeBasicModule(), true

	// ==================== std.math - 数学计算 ====================
	case "std.math.basic":
		return stdcore.LoadStdMathBasicModule(), true

	// ==================== std.net - 网络通信 ====================
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

	// ==================== std.web - Web开发 ====================
	case "std.web.express":
		return stdweb.LoadStdWebExpressModule(), true

	// ==================== std.db - 数据库 ====================
	case "std.db.sql":
		return stddb.LoadStdDBSQLModule(), true
	case "std.db.orm":
		return stddb.LoadStdDBORMModule(), true
	case "std.db.redis":
		return stddb.LoadStdDBRedisModule(), true

	// ==================== std.data - 数据处理 ====================
	case "std.data.json":
		return stdformat.LoadStdJSONModule(), true
	case "std.data.yaml":
		return stdformat.LoadStdYAMLModule(), true
	case "std.data.toml":
		return stdformat.LoadStdTOMLModule(), true
	case "std.data.xml":
		return stdformat.LoadStdXMLModule(), true
	case "std.data.csv":
		return stdformat.LoadStdCSVModule(), true
	case "std.data.compress":
		return stddata.LoadStdDataCompressModule(), true

	// ==================== std.crypto - 加密安全 ====================
	case "std.crypto.hash":
		return stddata.LoadStdCryptoHashModule(), true
	case "std.crypto.uuid":
		return stddata.LoadStdCryptoUUIDModule(), true

	// ==================== std.sys - 系统信息 ====================
	case "std.sys.os":
		return stdsystem.LoadStdSysOSModule(), true
	case "std.sys.host":
		return stdsystem.LoadStdSysHostModule(), true
	case "std.sys.exec":
		return stdsystem.LoadStdSysExecModule(), true

	default:
		return nil, false
	}
}
