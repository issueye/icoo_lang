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
- `lib/`
  - proxy 的配置、鉴权、路由、请求改写、管理接口和监控逻辑
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

其中：

- `/admin/requests` 用于查看 recent requests、计数器和延迟摘要
- `/admin/monitor` 用于查看完整服务监控快照

## v0.1 可用交付标准

要把这个 proxy 作为 `v0.1` 的“可用交付”看待，至少要满足下面条件：

1. `go run ./cmd/icoo run examples/proxy/smoke.ic` 稳定通过
2. proxy 成功路径可完成鉴权、路由改写、上游转发和响应回写
3. 失败路径可正确返回未授权、找不到路由、上游错误等结果
4. 请求 ID 能在下游响应和上游透传中保持一致
5. 服务监控数据可通过管理接口读取
6. 主路径不再依赖旧的 store 监控实现

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

当前仍然没有直接完成：

- 通用化的下游流式跨协议翻译
- `Responses -> Chat` 的 tool call / reasoning 等复杂事件翻译
- `Responses -> Anthropic` 的逐事件 SSE 翻译

也就是说，当前已经有了“流式底座 + 聚合回退路径 + 最小直译路径”，但还没有进入完整的“流式协议桥”阶段。
