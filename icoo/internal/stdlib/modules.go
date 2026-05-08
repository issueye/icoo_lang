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

var moduleLoaders = map[string]func() *runtime.Module{
	// std.core - 核心基础
	"std.core.object":  stdcore.LoadStdCoreObjectModule,
	"std.core.observe": stdcore.LoadStdCoreObserveModule,
	"std.core.log":     stdcore.LoadStdCoreLogModule,
	"std.core.string":  stdcore.LoadStdCoreStringModule,
	"std.core.service": stdcore.LoadStdCoreServiceModule,
	"std.core.cache":   stdcore.LoadStdCoreCacheModule,

	// std.io - 输入输出
	"std.io.console":  stdio.LoadStdIOConsoleModule,
	"std.io.fs":       stdio.LoadStdIOFSModule,
	"std.io.template": stdio.LoadStdIOTemplateModule,

	// std.time - 时间日期
	"std.time.basic": stdcore.LoadStdTimeBasicModule,

	// std.math - 数学计算
	"std.math.basic": stdcore.LoadStdMathBasicModule,

	// std.net - 网络通信
	"std.net.http.client":      stdnet.LoadStdNetHTTPClientModule,
	"std.net.http.server":      stdnet.LoadStdNetHTTPServerModule,
	"std.net.websocket.client": stdnet.LoadStdNetWebSocketClientModule,
	"std.net.websocket.server": stdnet.LoadStdNetWebSocketServerModule,
	"std.net.sse.client":       stdnet.LoadStdNetSSEClientModule,
	"std.net.sse.server":       stdnet.LoadStdNetSSEServerModule,
	"std.net.socket.client":    stdnet.LoadStdNetSocketClientModule,
	"std.net.socket.server":    stdnet.LoadStdNetSocketServerModule,

	// std.web - Web开发
	"std.web.express": stdweb.LoadStdWebExpressModule,

	// std.db - 数据库
	"std.db.sql":   stddb.LoadStdDBSQLModule,
	"std.db.orm":   stddb.LoadStdDBORMModule,
	"std.db.redis": stddb.LoadStdDBRedisModule,

	// std.data - 数据处理
	"std.data.json":     stdformat.LoadStdJSONModule,
	"std.data.yaml":     stdformat.LoadStdYAMLModule,
	"std.data.toml":     stdformat.LoadStdTOMLModule,
	"std.data.xml":      stdformat.LoadStdXMLModule,
	"std.data.csv":      stdformat.LoadStdCSVModule,
	"std.data.compress": stddata.LoadStdDataCompressModule,

	// std.crypto - 加密安全
	"std.crypto.hash": stddata.LoadStdCryptoHashModule,
	"std.crypto.uuid": stddata.LoadStdCryptoUUIDModule,

	// std.sys - 系统信息
	"std.sys.os":   stdsystem.LoadStdSysOSModule,
	"std.sys.host": stdsystem.LoadStdSysHostModule,
	"std.sys.exec": stdsystem.LoadStdSysExecModule,
	"std.sys.cli":  stdsystem.LoadStdSysCLIModule,
}

// LoadModule 根据模块路径加载标准库模块
// 采用三级命名规范：std.<领域>.<子领域>.<功能>
func LoadModule(spec string) (*runtime.Module, bool) {
	loader, ok := moduleLoaders[spec]
	if !ok {
		return nil, false
	}
	return loader(), true
}
