# Icoo 当前状态总览

本文档用于收口 Icoo 当前的实际状态，明确：

- 已实现能力
- 当前边界与风险
- 主要验证方式
- 更适合继续投入的方向

相关文档：

- 架构分析：`docs/architecture-analysis-report.md`
- 语言说明：`docs/language-design.md`
- AI 代码生成参考：`docs/ai-language-api.md`
- Runtime API：`docs/api.md`
- v0.1 交付规划：`docs/v0.1-delivery-plan.md`
- v0.1 发布说明：`docs/v0.1-release-notes.md`
- v0.1 验收清单：`docs/v0.1-acceptance-checklist.md`
- 迭代器协议：`docs/iterators.md`
- MVP 路线（历史规划）：`docs/mvp-roadmap.md`

## 当前结论

Icoo 的“最小可运行 MVP”阶段已经完成，而且当前实现已经明显超出最初 MVP 范围。

当前已经具备完整主链：

```text
源码 -> Lexer -> Parser -> AST -> 语义分析 -> Compiler -> Bytecode -> VM -> CLI
```

同时也已经具备：

- 项目级 `check` / `run`
- `repl`
- 源码级 `bundle`
- 可分发 `build`
- `extract` / `inspect`
- 文件模块与标准库模块系统
- 类、继承、装饰器
- `throw` / `try/catch/finally` / `expr?`
- `go` / `select` / channel
- `type` / `interface` / `satisfies`
- 覆盖多个能力域的标准库

因此如果还沿用“是否完成 MVP”的口径，答案已经不再是“正在接近”，而是：

- **MVP 早已完成**
- 当前更适合按“已有能力固化与质量补强”来评估项目状态

## 已实现能力

### 1. CLI 与工具链

当前 CLI 入口位于：

- `cmd/icoo/main.go`

已提供命令：

- `icoo`
- `icoo repl`
- `icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]`
- `icoo check <file|dir>`
- `icoo run <file|dir>`
- `icoo bundle <file|dir> [output]`
- `icoo build <file|dir> [output] [--metadata file]`
- `icoo extract <bundle|executable> [output]`
- `icoo inspect <bundle|executable>`

这说明项目已经不只是“解释器主链可跑”，而是具备了相对完整的工具链闭环。

### 2. API 门面层

当前对外运行时门面集中在：

- `pkg/api/runtime.go`
- `pkg/api/bundle.go`

已提供的主要能力：

- `NewRuntime()`
- `CheckSource()` / `CheckFile()`
- `RunSource()` / `RunFile()`
- `InvokeGlobal()`
- `RunReplLine()`
- `LoadBundle()` / `LoadBundleFile()`
- `CheckBundleFile()` / `CheckBundleArchive()`
- `RunBundleFile()` / `RunBundleArchive()`
- `SetProjectRoot()` / `SetBundledSources()`

这层已经足以支撑：

- CLI
- 测试
- REPL
- bundle 执行
- 基础宿主嵌入

### 3. 语言前端

关键目录：

- `internal/token/`
- `internal/lexer/`
- `internal/ast/`
- `internal/parser/`
- `internal/sema/`

已实现或已落地到主链的能力包括：

- token / span / 关键字表
- 词法分析
- 顶层声明解析
- Pratt 风格表达式解析
- 错误恢复
- 基础语义分析
- 作用域与标识符检查
- 重复声明检查
- 结构性合法性检查

### 4. 编译器与 VM

关键目录：

- `internal/bytecode/`
- `internal/compiler/`
- `internal/runtime/`
- `internal/vm/`

已具备：

- 单遍 AST -> 字节码编译
- 局部变量槽位管理
- 作用域深度跟踪
- break/continue 跳转回填
- 闭包捕获与 upvalue
- 异常处理器栈
- 模块执行上下文
- 类/方法调用上下文
- goroutine pool 支持的并发执行模型

### 5. 当前语言能力

#### 基础数据与表达式

- `null` / `bool` / `int` / `float` / `string`
- `array` / `object`
- 一元与二元表达式
- 赋值表达式
- `&&` / `||`
- 三元表达式 `cond ? a : b`
- 成员访问与下标访问
- 函数调用

#### 变量、函数与闭包

- `let` / `const`
- 命名函数
- 匿名函数
- 闭包捕获
- 嵌套函数

#### 控制流

- `if / else`
- `while`
- `for`
- `for-in`
- `break` / `continue`
- `return`
- `match`

#### 模块系统

- `import`
- `export`
- 文件模块加载
- 标准库模块加载
- 模块缓存
- 项目根别名导入

#### 错误与异常

- `throw`
- `try / catch / finally`
- `error(...)`
- 后缀 try 表达式 `expr?`

#### 类型与接口

- `type`
- `interface`
- `satisfies(value, InterfaceName)`

#### 类、继承与装饰器

- `class`
- `this`
- `init(...)`
- 实例方法
- 单继承 `class Dog < Animal`
- `super.init(...)`
- `super.method(...)`
- 函数装饰器
- 类装饰器
- 方法装饰器

#### 并发

- `chan()`
- `send` / `recv`
- `trySend` / `tryRecv`
- `close`
- `go`
- `select`

### 6. 迭代器协议

相关文档：

- `docs/iterators.md`
- `docs/language-design.md`

当前 `for-in` 已基于统一迭代器协议实现，支持：

- `iter()` / `next()`
- 单绑定：`for item in iterable`
- 双绑定：`for key, value in iterable`
- `_` 忽略绑定
- array / string / object / module / iterator 的默认迭代行为
- 对象通过自定义 `iter` 覆盖默认行为

### 7. 标准库

标准库注册入口：

- `internal/stdlib/modules.go`

当前已注册模块包括：

- `std.io`
- `std.time`
- `std.math`
- `std.db`
- `std.json`
- `std.yaml`
- `std.toml`
- `std.xml`
- `std.fs`
- `std.exec`
- `std.os`
- `std.host`
- `std.express`
- `std.http.client`
- `std.http.server`
- `std.net.websocket.client`
- `std.net.websocket.server`
- `std.net.sse.client`
- `std.net.sse.server`
- `std.net.socket.client`
- `std.net.socket.server`
- `std.crypto`
- `std.uuid`
- `std.compress`

从能力域看，标准库已经不再是“最小集验证”，而是覆盖了：

- core
- format
- system
- net
- database
- data
- express

## 当前边界与风险

结合当前代码和 `docs/architecture-analysis-report.md`，更值得关注的已经不是“哪些基础能力还没做”，而是下面这些风险点。

### 1. 文档与实现容易再次脱节

虽然本轮已经补充说明文档和 API 文档，但项目特性扩展很快：

- class / inheritance
- decorators
- exceptions / finally / try expr
- go / select
- type / interface
- build / bundle

后续如果继续快速迭代，文档仍然很容易再次落后于代码。

### 2. 特性交互复杂度高

当前难点已经从“单个特性是否存在”转为“多个特性组合时是否稳定”，例如：

- `finally` 与 `return` / `break` / `continue` / `throw`
- 闭包与 `go`
- `select` 与作用域绑定
- `super` 与闭包
- 装饰器与类初始化

### 3. 前端与编译层白盒测试仍可加强

`pkg/api/*_test.go` 已覆盖大量端到端行为，但从长期维护角度看，仍建议继续补强：

- lexer 单元测试
- parser 单元测试
- sema 单元测试
- compiler 单元测试

### 4. 标准库增长快于统一约束

当前标准库能力已经不少，但仍需要持续收敛：

- 错误模型
- 返回值风格
- 资源生命周期
- 命名一致性
- 跨模块 API 手感

### 5. bundle 仍是源码级归档

当前 bundle / build 的方向是合理的，但也带来已知边界：

- 运行时仍需重新 parse / compile
- 启动速度仍受源码主链影响
- 尚未进入字节码级打包阶段
- 版本兼容策略仍较轻量

## 当前验证方式

### 1. Go 测试

当前基础验证方式仍是：

```bash
go test ./...
```

重点覆盖区域包括：

- `pkg/api/*_test.go`
- `cmd/icoo/*_test.go`

从现有测试名可以看到，已覆盖的主题包含：

- 运行时主链
- import/export
- project root alias
- 闭包
- 逻辑短路
- 三元表达式
- `try` 表达式
- 类型与接口
- 类、继承、`super`
- 装饰器
- channel / `go` / `select`
- bundle / build / extract / inspect
- examples 批量运行

### 2. 示例脚本

示例入口：

- `examples/README.md`

当前 examples 已覆盖从基础语法到标准库、网络、数据库、并发、装饰器、继承的多种场景，可作为：

- 冒烟测试
- 回归测试素材
- 新读者示例

### 3. CLI 闭环验证

当前除了 `run` / `check` 外，还可以通过下列命令验证工具链行为：

- `icoo init`
- `icoo bundle`
- `icoo build`
- `icoo extract`
- `icoo inspect`
- `icoo repl`

## 对当前阶段的判断

如果问题是“项目现在是否还处在 MVP 收口阶段”，更准确的回答是：

- **不是**

更合适的表述是：

- 主链已经完整
- 工具链已经成型
- 高级语言特性已超过 MVP 范围
- 当前工作重心应从“继续证明能跑”切换到“固化现有能力”

## 更适合优先投入的方向

按当前状态，更值得优先推进的方向是：

1. 文档持续同步
2. parser / sema / compiler 白盒测试补强
3. 高级特性交互测试
4. 标准库 API 一致性整理
5. 错误信息与调试体验优化

如果继续往产品化方向推进，再考虑：

6. bundle / build 的版本与启动体验优化
7. 字节码级 bundle 方案
8. 更清晰的嵌入式宿主 API

## 相关文档

- `docs/architecture-analysis-report.md`
- `docs/language-design.md`
- `docs/api.md`
- `docs/iterators.md`
- `docs/mvp-roadmap.md`
