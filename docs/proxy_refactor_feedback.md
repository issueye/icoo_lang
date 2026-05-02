# `icoo_proxy` 重构对语言演进的反馈

## 结论先行

`examples/proxy` 到 `proxy/icoo_server` 的演进，不是单纯的“把脚本翻译成 Go”，而是一次明确的职责迁移：

1. `.ic` 版本适合做可运行样例和轻量业务编排。
2. Go 版本承接了协议转换、流式聚合、链路日志、持久化流量记录、服务分层和长期维护成本。
3. 这说明当前语言已经能表达“业务流程”，但在“产品级服务工程”上仍有明显摩擦。

这次对照的价值，不是要求语言去复制 Go，而是识别出哪些复杂度应该继续留给宿主语言，哪些复杂度值得回流到 `icoo_lang` 本身。

## 一、重构里暴露出的语言摩擦

### 1. 对象访问成本偏高

`examples/proxy` 里反复出现这些模式：

- 手写 `lookup(obj, "field")`
- 手写 `headerValue(req, "Authorization")`
- 为了判断字段是否存在，循环遍历对象
- 为了做配置覆盖，手写一长串字段拷贝

根因不是业务复杂，而是对象索引在 key 缺失时会报错，所以真实服务代码不得不绕开 `obj[key]` 直写。

这类摩擦已经直接影响示例代码形态，说明它不是“语法洁癖问题”，而是会把业务代码推向样板化。

### 2. HTTP 抽象只覆盖了同步 JSON 场景

`.ic` 版本的代理能力主要停留在：

- 读取 JSON body
- 改写 model
- 发一个同步 HTTP 请求
- 返回一个同步 HTTP 响应

而 Go 版本新增的大量复杂度都围绕这些缺口展开：

- 事件流透传与聚合
- 跨协议流式转换
- 上下游请求/响应镜像保存
- 响应头过滤和细粒度控制
- 统一错误格式与链路观测

这说明当前 `std.http.server` / `std.express` 更像“轻量 Web API 工具”，还不是“代理/网关工具”。

### 3. 缺少面向服务代码的观测与中间件骨架

Go 重构版里最膨胀的模块不是业务路由，而是：

- request context
- request id
- structured logging
- recent request snapshot
- persistent traffic recorder
- health/ready/admin state

这些能力如果完全靠业务层重复实现，会让 `.ic` 服务一旦走出 demo 规模，就迅速失去可维护性。

### 4. 配置与数据变换仍然过度手工

`examples/proxy/lib/config.ic` 和 `json_helpers.ic` 里可以看到很多“浅层对象搬运”：

- 环境变量覆盖
- 请求体字段改写
- route 结构克隆
- 管理接口响应拼装

这些不是高级抽象需求，而是服务脚本最常见的日常操作。当前如果没有内建帮助，代码就会不断退化成“对象字段搬运层”。

## 二、哪些复杂度不必强行回流到语言

以下部分更适合继续由 Go 宿主或原生扩展承接，不建议短期内为了“纯语言化”强塞回脚本层：

- 高性能 HTTP 细节控制
- 协议级别的流式桥接与聚合
- 大量 typed model 的精确 JSON 编解码
- LevelDB / GORM 这类强工程化依赖

原因很简单：它们的价值主要来自成熟生态、静态类型和调试工具，而不是脚本层表达力本身。

## 三、值得优先回流到语言的能力

### P0：补齐安全对象访问与浅合并

这是当前最确定、投入最小、收益最直接的回流项。

本次已先落地 `std.object`：

- `object.get(obj, key[, fallback])`
- `object.has(obj, key)`
- `object.keys(obj)`
- `object.merge(...objects)`

它解决的是 `.ic` 服务代码里最密集的一层样板，尤其适合配置覆盖、headers 读取、JSON payload 改写这类场景。

### P1：补齐代理/网关导向的 HTTP 标准库

建议在标准库而非语法层扩展：

- 原始 body 流访问
- 流式响应转发
- 更完整的 response writer 控制
- 中间件链与统一错误处理
- 请求上下文对象

这会比继续扩展一套“更像 Express 的表面 API”更有价值。

### P1：补齐服务观测标准件

适合以 `std.service` 或 `std.observe` 一类模块提供：

- request id
- structured log helper
- ring buffer recent requests
- health/readiness helper
- redact sensitive fields helper

这些能力一旦沉到标准库，示例才能稳定长成“可维护的服务”，而不是“能跑的脚本”。

### P2：考虑语法层面的对象便捷写法

如果后续继续观察到大量同类样板，可以考虑：

- 可空对象访问
- 缺省值访问
- 对象展开 / patch 语法

但这类改动应该排在标准库改进之后。当前问题首先是“没有顺手工具”，还不是“必须新增语法”。

## 四、这次已经落实的最小改进

为了让这份分析不是停留在文档，本次同步做了一个小而明确的回流：

1. 新增 `std.object` 标准库模块。
2. 补齐顶层 `import` 在闭包/IIFE 中的可捕获性。
3. 补齐字符串键对象字面量，以及服务端请求对象的 `req.header(name)` / `req.hasHeader(name)` / `req.json`。
4. 为 `std.http.server` 和 `std.express` 增加双参 handler 形态 `fn(req, res)`，支持 `res.status(...)` / `res.statusCode()` / `res.setHeader(...)` / `res.write(...)` / `res.json(...)` / `res.end(...)`，让脚本层第一次具备“直接控制响应”的能力。
5. 继续补上 `res.proxy(req, options)`，让代理型服务可以把上游响应直接流式写回下游，而不是像 `forward()` 一样先整包读入内存再返回对象。
6. 用这些能力重写了 `examples/proxy` 中最典型的对象访问、header 读取、JSON 解码、模块缓存样板，以及成功代理路径上的响应回写方式。
7. 增加对应运行时测试，确保模块可用。

这一步不能替代更大的 HTTP/服务化能力建设，但它验证了一件事：

> `icoo_proxy` 的重构确实能反向指导语言演进，而且其中一部分问题可以通过“小而准”的标准库补强立刻改善。

## 五、建议的下一步顺序

1. 继续把 `examples/proxy` 当作语言回归样例，专门观察服务端样板代码密度。
2. 继续沿 `std.http.server` / `std.express` 补代理友好的低层 HTTP 能力，尤其是请求流和真正的流式上游桥接，而不是直接追求更多语法。
3. 等服务端样例稳定后，再判断是否需要对象展开、可空访问等语法级增强。
