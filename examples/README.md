# Icoo 语言示例

这些示例是可以直接运行的 `.ic` 小程序，用来从基础语法一路介绍到常用标准库能力。

可以这样运行单个示例：

```powershell
go run ./cmd/icoo run examples/01_data_types.ic
```

建议按下面的顺序阅读：

1. `01_data_types.ic` - `null`、布尔、整数、浮点数、字符串、数组、对象
2. `02_functions_and_closures.ic` - 函数、匿名函数、闭包
3. `03_control_flow.ic` - `if`、三元表达式、`for`、`break`、`continue`、逻辑运算
4. `04_collections_and_iteration.ic` - 数组、对象、字符串、迭代器
5. `05_classes_and_methods.ic` - 类、`this`、实例方法、状态
6. `06_errors_and_try.ic` - `throw`、`try/catch/finally`、`error()`、后缀 `?`
7. `07_types_and_interfaces.ic` - 类型别名、接口、`satisfies`
8. `08_modules_and_formats.ic` - 本地模块、JSON/YAML/TOML/XML 编解码
9. `09_http_client_server.ic` - `std.http.client` 和 `std.http.server`
10. `10_concurrency.ic` - channel、`go`、`select`
11. `11_decorators.ic` - 类 Python 的 `@decorator` 包装器
12. `12_inheritance_and_super.ic` - 类继承、方法覆写、`super.init()`、`super.method()`
13. `13_system_modules.ic` - `std.os`、`std.host`、`std.exec`
14. `14_fs_and_files.ic` - `std.fs` 与临时目录下的文件操作
15. `15_sqlite_database.ic` - `std.db` + SQLite 内存库
16. `16_crypto_uuid_compress.ic` - `std.crypto`、`std.uuid`、`std.compress`
17. `17_express_app.ic` - `std.express` + `std.http.client`
18. `18_websocket_echo.ic` - `std.net.websocket.server` 与 `std.net.websocket.client`
19. `19_sse_stream.ic` - `std.net.sse.server` 与 `std.net.sse.client`
20. `20_socket_tcp_udp.ic` - `std.net.socket.server` 与 `std.net.socket.client`
21. `21_http_json_and_download.ic` - `std.http.client` 的 `post`、`requestJSON`、`download`
22. `22_format_files_roundtrip.ic` - JSON/YAML/TOML/XML 的 `saveToFile` 与 `fromFile`
23. `23_http_forward_proxy.ic` - `std.http.server.forward` 的反向代理转发
24. `24_crypto_helpers.ic` - `std.crypto` 的 hex、HMAC、随机字节
25. `25_http_put_and_delete.ic` - `std.http.client.put` 与 `std.http.client.delete`
26. `26_config_file_tool.ic` - `std.fs`、`std.json`、`std.yaml` 组合成配置文件工具
27. `27_db_open_driver.ic` - `std.db.open(...)` 的通用驱动入口
28. `28_db_orm.ic` - `std.db.table(...)` 的轻量 ORM / 查询构造器

示例使用到的辅助模块位于 `examples/lib`。
