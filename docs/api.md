# Icoo Runtime API 文档

本文档描述当前仓库中 `pkg/api` 包已经公开的运行时与 bundle API，面向：

- CLI 调用链阅读者
- 测试编写者
- 需要从 Go 宿主程序调用 Icoo 的开发者

相关文档：

- AI 代码生成参考：`docs/ai-language-api.md`
- 语言说明：`docs/language-design.md`
- 当前状态：`docs/mvp-status.md`

相关实现：

- `pkg/api/runtime.go`
- `pkg/api/bundle.go`

## 设计定位

`pkg/api` 是 Icoo 的门面层。

它位于：

- `cmd/icoo/` 与 `internal/*` 之间
- 测试代码与底层 VM 之间

主要职责：

- 封装源码检查主链：`Lexer -> Parser -> Sema`
- 封装源码执行主链：`Lexer -> Parser -> Sema -> Compiler -> VM`
- 封装文件模块与标准库模块加载
- 封装 bundle 的读取、检查与执行
- 为 REPL、CLI、测试提供稳定入口

## 核心类型

### `Runtime`

定义位置：`pkg/api/runtime.go:17`

```go
type Runtime struct {
	vm               *vm.VM
	modules          map[string]*runtime.Module
	bundledSources   map[string]string
	projectRoot      string
	projectRootAlias string
}
```

字段含义：

- `vm`：当前运行时持有的底层虚拟机实例
- `modules`：已加载文件模块缓存，按解析后的路径缓存
- `bundledSources`：bundle 场景下的虚拟源码表
- `projectRoot`：项目根目录
- `projectRootAlias`：项目根别名，用于别名导入

### `BundleArchive`

定义位置：`pkg/api/bundle.go:14`

```go
type BundleArchive struct {
	Version       int               `json:"version"`
	Entry         string            `json:"entry"`
	EntryFunction string            `json:"entry_function,omitempty"`
	ProjectRoot   string            `json:"project_root,omitempty"`
	RootAlias     string            `json:"root_alias,omitempty"`
	Modules       map[string]string `json:"modules"`
}
```

字段说明：

- `Version`：bundle 格式版本，当前常量为 `BundleVersion = 1`
- `Entry`：入口模块相对路径
- `EntryFunction`：入口函数名，可选
- `ProjectRoot`：bundle 中项目根目录的相对位置，可选
- `RootAlias`：项目根别名，可选
- `Modules`：归档中所有模块源码，键为相对路径，值为源码文本

## 创建运行时

### `NewRuntime()`

定义位置：`pkg/api/runtime.go:25`

```go
func NewRuntime() *Runtime
```

作用：

- 创建底层 `vm.VM`
- 初始化模块缓存与 bundle 源码表
- 注册模块加载器
- 注册 builtins

当前注册的 builtins 可从 `internal/stdlib/builtin.go:12` 看到，包括：

- `print`
- `println`
- `len`
- `typeOf`
- `chan`
- `panic`
- `error`
- `satisfies`
- 若干运行时内部辅助 builtin，如 `__select`、`__buildClass`、`__superGet`

示例：

```go
rt := api.NewRuntime()
```

## 运行时配置 API

### `VM()`

定义位置：`pkg/api/runtime.go:79`

```go
func (r *Runtime) VM() *vm.VM
```

作用：返回底层 VM 指针。

适用场景：

- 你需要使用 `pkg/api` 没有直接暴露的底层能力
- 你要查看 globals、frames 或做更底层调试

注意：一旦直接操作底层 VM，就需要自己理解 `internal/vm` 的状态约束。

### `ConfigureGoPool(workers, queueSize)`

定义位置：`pkg/api/runtime.go`

```go
func (r *Runtime) ConfigureGoPool(workers, queueSize int) error
```

作用：

- 配置脚本 `go` 语句使用的 goroutine pool
- 为并发任务设置 worker 数和排队上限
- 避免队列打满后继续无限制创建宿主 goroutine

行为说明：

- `workers <= 0` 时会回退到默认值
- `queueSize <= 0` 时会按 `workers * 16` 计算默认队列长度
- 如果池中仍有活动或排队任务，拒绝热重配

### `Stats()`

定义位置：`pkg/api/runtime.go`

```go
func (r *Runtime) Stats() vm.RuntimeStats
```

作用：

- 读取当前运行时资源快照
- 返回 CPU、goroutine、内存以及 goroutine pool 统计

可用于：

- 宿主程序监控
- 压测时观测队列积压
- 排查内存或协程异常增长

### `CollectGarbage()`

定义位置：`pkg/api/runtime.go`

```go
func (r *Runtime) CollectGarbage() vm.RuntimeStats
```

作用：

- 主动触发一次 Go GC
- 返回 GC 后的最新资源快照

### `Shutdown(ctx)` / `Close()`

定义位置：`pkg/api/runtime.go`

```go
func (r *Runtime) Shutdown(ctx context.Context) error
func (r *Runtime) Close() error
```

作用：

- 停止接收新的脚本 goroutine 任务
- 关闭 goroutine pool，减少运行时残留 worker

行为说明：

- `Shutdown(ctx)` 支持调用方控制等待时长
- `Close()` 提供一个带默认超时的便捷关闭入口
- CLI 已在 REPL / run / bundle 执行链路里主动调用关闭逻辑

### `SetProjectRoot(root, alias)`

定义位置：`pkg/api/runtime.go:37`

```go
func (r *Runtime) SetProjectRoot(root string, alias string)
```

作用：

- 配置项目根目录
- 配置项目根别名导入

当脚本导入：

```icoo
import app/lib/util.ic
```

若 `app` 被配置为 `root_alias`，运行时会把它映射到项目根目录下的真实路径。

实现行为：

- `root` 会被 `filepath.Clean` 规范化
- `alias` 会被 `TrimSpace`
- 别名导入不允许逃逸出项目根目录

### `SetBundledSources(sources)`

定义位置：`pkg/api/bundle.go:51`

```go
func (r *Runtime) SetBundledSources(sources map[string]string)
```

作用：

- 为 bundle 执行预装虚拟源码表
- 让模块加载时优先从内存中的 bundle 源读取，而不是磁盘读取

通常不需要手动调用，`RunBundleArchive()` 会自行准备它。

## 检查 API

### `CheckSource(src)`

定义位置：`pkg/api/runtime.go:42`

```go
func (r *Runtime) CheckSource(src string) []error
```

执行链：

```text
LexAll
-> ParseProgram
-> sema.Analyze
```

特点：

- 只做词法、语法、语义检查
- 不进入编译与执行阶段
- 返回 `[]error`，可能包含多个错误

适合：

- 编辑器诊断
- 静态检查
- 测试 parser/sema 行为

### `CheckFile(path)`

定义位置：`pkg/api/runtime.go:59`

```go
func (r *Runtime) CheckFile(path string) []error
```

行为：

1. 读取文件
2. 调用 `CheckSource`

读取失败时会返回单元素错误切片。

示例：

```go
errs := rt.CheckFile("examples/03_control_flow.ic")
if len(errs) > 0 {
	// handle errors
}
```

## 执行 API

### `RunSource(src)`

定义位置：`pkg/api/runtime.go:55`

```go
func (r *Runtime) RunSource(src string) (runtime.Value, error)
```

执行链：

```text
LexAll
-> ParseProgram
-> sema.Analyze
-> compiler.Compile
-> vm.RunModule
```

特点：

- 直接运行源码字符串
- 不附带真实文件路径
- 模块内相对导入通常应优先使用 `RunFile`

适合：

- 单元测试
- 小段脚本执行
- REPL 外的动态片段执行

### `RunFile(path)`

定义位置：`pkg/api/runtime.go:67`

```go
func (r *Runtime) RunFile(path string) (runtime.Value, error)
```

行为：

1. 解析绝对路径
2. 读取源码
3. 调用内部 `runModuleSource(absPath, src)`

相比 `RunSource`，`RunFile` 会带着真实模块路径进入执行链，因此更适合有导入关系的项目脚本。

### `InvokeGlobal(name)`

定义位置：`pkg/api/runtime.go:83`

```go
func (r *Runtime) InvokeGlobal(name string) (runtime.Value, error)
```

作用：调用当前 VM 里已存在的全局可调用值。

当前可调用值类型：

- `*runtime.Closure`
- `*runtime.NativeFunction`

错误情况：

- 全局不存在：`undefined global`
- 全局存在但不可调用：`global is not callable`

典型用途：

- 先运行入口模块
- 再调用约定好的导出入口函数

### `RunReplLine(line)`

定义位置：`pkg/api/runtime.go:97`

```go
func (r *Runtime) RunReplLine(line string) (runtime.Value, error)
```

特点：

- 如果输入看起来是纯表达式，会自动包装成 `return <expr>`
- REPL 模式下会跳过 `sema`
- 解析、编译、执行单行程序

这样设计的结果是：

- 输入 `1 + 2` 可以直接得到返回值
- 输入声明语句则按语句执行
- REPL 更偏轻量交互入口，而不是完整增量编译环境

## Bundle API

### `LoadBundle(data)`

定义位置：`pkg/api/bundle.go:23`

```go
func LoadBundle(data []byte) (*BundleArchive, error)
```

作用：从 JSON 字节流解析 bundle。

校验内容：

- 版本必须等于 `BundleVersion`
- `Entry` 不能为空
- `Modules` 不能为空
- `Entry` 对应源码必须存在于 `Modules`

### `LoadBundleFile(path)`

定义位置：`pkg/api/bundle.go:43`

```go
func LoadBundleFile(path string) (*BundleArchive, error)
```

行为：

1. 读取 bundle 文件
2. 调用 `LoadBundle`

### `CheckBundleFile(path)`

定义位置：`pkg/api/bundle.go:58`

```go
func (r *Runtime) CheckBundleFile(path string) []error
```

行为：

1. 读取并解析 bundle 文件
2. 调用 `CheckBundleArchive`

### `CheckBundleArchive(archive)`

定义位置：`pkg/api/bundle.go:66`

```go
func (r *Runtime) CheckBundleArchive(archive *BundleArchive) []error
```

行为：

- 遍历归档中每个模块源码
- 逐个调用 `CheckSource`
- 返回带模块相对路径前缀的错误集合

### `RunBundleFile(path)`

定义位置：`pkg/api/bundle.go:77`

```go
func (r *Runtime) RunBundleFile(path string) (runtime.Value, error)
```

行为：

1. 读取并解析 bundle 文件
2. 调用 `RunBundleArchive`

### `RunBundleArchive(path, archive)`

定义位置：`pkg/api/bundle.go:85`

```go
func (r *Runtime) RunBundleArchive(path string, archive *BundleArchive) (runtime.Value, error)
```

执行步骤：

1. 生成 bundle 的虚拟根目录 `__bundle__`
2. 把归档模块映射为虚拟绝对路径
3. 写入 `bundledSources`
4. 如果 bundle 含 `RootAlias`，同时配置虚拟项目根
5. 运行入口模块
6. 如果配置了 `EntryFunction`，再调用该全局函数

这也是 `build` / `extract` / `inspect` 工具链能和运行时衔接起来的关键 API。

## 模块加载语义

虽然 `loadModule()` 没有作为公开 API 暴露，但理解它有助于正确使用 `RunFile` 和 bundle API。

相关实现：`pkg/api/runtime.go:171`

加载顺序：

1. 先尝试标准库 `stdlib.LoadModule(spec)`
2. 再解析文件模块路径
3. 查 `Runtime.modules` 缓存
4. 优先从 `bundledSources` 读源码，否则从磁盘读取
5. 为子模块创建新的 `vm.VM`
6. 重新注册 builtins 和 module loader
7. parse -> sema -> compile -> run
8. 收集模块导出表并缓存

实现特征：

- 是递归模块加载模型
- 文件模块按 resolved path 缓存
- 运行失败会删除缓存项
- 支持 project root alias

当前标准库导入除了早期的 `std.io` / `std.time` / `std.json` 之外，也已经覆盖：

- `std.math`
- `std.object`
- `std.observe`
- `std.service`
- `std.db`
- `std.orm`
- `std.redis`
- `std.fs`
- `std.exec`
- `std.os`
- `std.host`
- `std.http.client`
- `std.http.server`
- `std.net.websocket.client`
- `std.net.websocket.server`
- `std.net.sse.client`
- `std.net.sse.server`
- `std.net.socket.client`
- `std.net.socket.server`
- `std.express`
- `std.crypto`
- `std.uuid`
- `std.compress`
- `std.yaml`
- `std.toml`
- `std.xml`

## 返回值与错误约定

### 返回值

大多数执行 API 返回：

```go
(runtime.Value, error)
```

其中：

- `runtime.Value` 是语言级返回值
- `error` 是宿主层错误

两者语义不同：

- 语言级 `error(...)` 产生的是 `runtime.Value` 中的错误值
- 宿主层 `error` 表示读取文件失败、编译失败、运行时未捕获异常等

### 检查错误

检查 API 返回 `[]error` 而不是单个 `error`，因为：

- parser 可能收集多个错误
- sema 也可能产出多个诊断

而执行 API 当前通常在主链中遇到第一个错误就返回。

## 常见使用模式

### 1. 检查单个文件

```go
rt := api.NewRuntime()
errs := rt.CheckFile("script.ic")
if len(errs) > 0 {
	for _, err := range errs {
		fmt.Println(err)
	}
}
```

### 2. 执行单个文件

```go
rt := api.NewRuntime()
result, err := rt.RunFile("script.ic")
if err != nil {
	panic(err)
}
if result != nil {
	fmt.Println(result.String())
}
```

### 3. 运行源码片段

```go
rt := api.NewRuntime()
result, err := rt.RunSource(`
fn add(a, b) {
  return a + b
}
add(1, 2)
`)
```

### 4. 运行模块后调用入口函数

```go
rt := api.NewRuntime()
if _, err := rt.RunFile("main.ic"); err != nil {
	panic(err)
}
value, err := rt.InvokeGlobal("main")
```

### 5. 执行 bundle

```go
rt := api.NewRuntime()
result, err := rt.RunBundleFile("app.icb")
```

## 与 CLI 的关系

CLI 入口位于 `cmd/icoo/main.go:12`，其大部分语言相关工作最终都会委托给 `pkg/api`。

可以简单理解为：

- `cmd/icoo` 负责命令行交互与参数处理
- `pkg/api` 负责运行时门面
- `internal/*` 负责真正的语言实现

## 当前边界

当前 `pkg/api` 已经足够支撑：

- CLI
- REPL
- 测试
- bundle 运行
- 基础宿主嵌入

但它还不是一个“完整嵌入式 SDK”。

目前没有公开暴露的高层 API 包括：

- 直接注册宿主函数到 `Runtime` 的便捷方法
- 直接注册宿主模块的便捷方法
- 通用的 `runtime.Value` 与 Go 值双向转换 helper

如果后续要增强嵌入能力，建议优先围绕这些点扩展。
