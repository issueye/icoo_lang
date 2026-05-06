# Icoo Proxy 示例

这个目录不是普通语法示例，而是 `icoo_lang` 在 `v0.1` 阶段最关键的服务端可用交付样例。

它承担两件事：

1. 证明 `icoo` 语言和标准库已经能支撑一个可运行的代理服务
2. 证明这轮从 `proxy/icoo_server` 反哺出来的能力已经真正落进语言侧，而不是停留在分析文档里

相关文档：

- `v0.1` 可用交付说明：`examples/proxy/v0.1-delivery.md`
- 总交付规划：`docs/v0.1-delivery-plan.md`
- proxy 重构反馈：`docs/proxy_refactor_feedback.md`

## 目录说明

- `app.ic`
  - 项目入口，直接调用 `src/main.ic` 的 `main()`
- `src/main.ic`
  - 对外暴露 `start` / `stop` / `main`
- `src/controllers/`
  - HTTP 请求控制器，只负责把请求转发到服务层
- `src/models/`
  - 配置、运行时状态、catalog、history、协议转换与持久化 store
- `src/routes/`
  - 路由声明层，负责把 URL/HTTP 方法映射到控制器
- `src/services/`
  - 控制面、catalog 装配与代理请求管线等服务层
- `src/views/`
  - HTTP/JSON 响应包装、traffic/catalog/proxy 输出等视图层输出
- `smoke.ic`
  - proxy 可用交付的冒烟验收脚本
- `project.toml`
  - 项目入口与 root alias 配置

## 运行方式

### 1. 直接运行 proxy

```powershell
go run ./cmd/icoo run examples/proxy/app.ic
```

这会启动一个长期运行的 proxy 进程。

### 2. 运行 proxy 冒烟验收

```powershell
go run ./cmd/icoo run examples/proxy/smoke.ic
```

这个脚本会：

- 启动一个临时 upstream 服务
- 启动 proxy
- 发起授权与未授权请求
- 验证路由改写、模型改写、请求 ID 透传
- 验证 `/healthz`、`/readyz`、`/admin/routes`、`/admin/requests`、`/admin/monitor`
- 验证 recent requests、counters、latency 监控数据

## 当前依赖的语言能力

这个示例当前直接依赖以下本轮反哺出来的能力：

- `std.object`
- `req.header(name)` / `req.hasHeader(name)` / `req.json`
- 双参 handler `fn(req, res)`
- `res.status(...)` / `res.setHeader(...)` / `res.write(...)` / `res.json(...)` / `res.end(...)`
- `res.sse(...)`
- `res.proxy(req, options)`
- `req.requestId`
- `std.net.sse.client.request(...)`
- `std.observe.recent(limit)`
- `std.service.create(...)`
- `std.orm.model(...)`

因此这个 example 不是“附属演示”，而是本轮语言演进是否真正落地的回归样例。

## 管理接口

当前 proxy 暴露以下接口：

- `/healthz`
- `/readyz`
- `/admin/routes`
- `/admin/models`
- `/admin/requests`
- `/admin/monitor`
- `/admin/suppliers/health`
- `/admin/catalog`
- `/admin/suppliers`
- `/admin/endpoints`
- `/admin/model-aliases`
- `/admin/auth-keys`
- `/admin/route-policies`
- `/admin/history`
- `/admin/history/clear`
- `/api/overview`
- `/api/state`
- `/api/proxy/reload`
- `/api/traffic`
- `/api/traffic/clear`
- `/api/suppliers`
- `/api/endpoints`
- `/api/model-aliases`
- `/api/auth-keys`
- `/api/route-policies`
- `/api/settings`
- `/api/ui-prefs`

其中：

- `/admin/requests` 用于查看 recent requests、计数器和延迟摘要
- `/admin/monitor` 用于查看完整服务监控快照
- `/admin/suppliers/health` 用于查看当前 upstream 供应方健康状态
- `/admin/catalog` 用于查看持久化目录模型快照
- `/admin/endpoints`、`/admin/model-aliases`、`/admin/auth-keys`、`/admin/route-policies` 用于查看当前 catalog store 的只读数据
- `/admin/history` 用于查看持久化请求历史
- `/api/*` 当前提供最小控制面，可对 suppliers、endpoints、model aliases、auth keys、route policies、settings、ui prefs 做最小 list/upsert/delete，并支持 `reload`

## v0.1 可用交付标准

要把这个 proxy 作为 `v0.1` 的“可用交付”看待，至少要满足下面条件：

1. `go run ./cmd/icoo run examples/proxy/smoke.ic` 稳定通过
2. proxy 成功路径可完成鉴权、路由改写、上游转发和响应回写
3. 失败路径可正确返回未授权、找不到路由、上游错误等结果
4. 请求 ID 能在下游响应和上游透传中保持一致
5. 服务监控数据可通过管理接口读取
6. 主路径不再依赖旧的 store 监控实现
7. 请求 history 可通过持久化接口跨重启读取

## 当前边界

这个 proxy 的定位仍然是：

- 可运行的服务端样例
- 语言能力回归样例
- 标准库设计反馈样例

它还不是：

- 完整产品化网关
- 完整流式协议桥接框架
- 完整持久化流量记录系统
- 生产级运维平台

这些复杂度仍然应优先由 Go 宿主层承接。

## 当前新增边界

截至 `2026-05-03`，这个 proxy 还额外具备了两条很重要但仍然刻意收敛的流式能力：

- 可把上游 `OpenAI Responses` 的 SSE 响应聚合回 JSON
- 可把上游 `OpenAI Responses` 的文本 SSE 响应直接翻译成 `OpenAI Chat` SSE 响应
- 可把上游 `OpenAI Responses` 的最小 tool call SSE 事件翻译成 `OpenAI Chat` `tool_calls` delta

当前仍然没有直接完成：

- 通用化的下游流式跨协议翻译
- `Responses -> Chat` 的 reasoning 等复杂事件翻译
- `Responses -> Anthropic` 的逐事件 SSE 翻译

也就是说，当前已经有了“流式底座 + 聚合回退路径 + 最小直译路径”，但还没有进入完整的“流式协议桥”阶段。

## 当前协议转换边界

截至 `2026-05-03`，这个 proxy 在非流式 JSON 路径上已经具备最小三协议转换能力：

- `OpenAI Chat <-> OpenAI Responses`
- `Anthropic Messages -> OpenAI Responses`
- `OpenAI Responses -> Anthropic Messages`

当前这层能力刻意只覆盖主路径字段：

- `system` / `instructions`
- text message / content
- `max_tokens` / `max_output_tokens`
- usage 的最小映射

仍然明确没有一次性补齐：

- Anthropic tool use / tool result 的完整双向映射
- reasoning / thinking block 的完整跨协议语义保持
- Anthropic 逐事件 SSE 翻译

这也是当前这轮反哺的重要设计结论：先把主路径 JSON 转换薄化进语言侧，再继续决定哪些复杂协议细节值得进入标准库，哪些仍应留在宿主层。

## 当前持久化边界

截至 `2026-05-03`，这个 proxy 还额外具备了最小持久化 history 闭环：

- 基于 `std.db + std.orm` 的 SQLite 请求历史存储
- 基于 `std.db + std.orm` 的 SQLite catalog 存储
- `/admin/history` 的只读查询接口
- `/admin/history/clear` 的清理入口
- `/admin/catalog` 及目录只读接口
- `/api/*` 最小控制面与 runtime reload
- 跨重启 history 保留验证
- 跨重启 catalog / alias / auth key 保留验证

这一步反哺出的语言结论同样明确：

- `std.db` / `std.orm` 已经足以支撑轻量服务级持久化
- 服务控制面要真正可用，还需要薄的 query 参数数值解析原语
- 因此本轮同步补上了 `std.math.parseInt(...)`
