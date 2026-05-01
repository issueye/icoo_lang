# Icoo 架构分析报告（面向开发者）

## 1. 项目概述

Icoo 是一个使用 Go 实现的脚本语言运行时与工具链项目。  
从当前代码结构看，它采用典型的分层式语言实现架构：

```text
源码
  ↓
Lexer
  ↓
Parser
  ↓
AST
  ↓
Sema
  ↓
Compiler
  ↓
Bytecode
  ↓
VM
  ↓
标准库 / 模块系统 / CLI
```

核心目标不是只做一个“能跑 demo 的解释器”，而是构建一套可扩展的语言基础设施，包括：

- 脚本语言前端
- 字节码编译执行引擎
- 模块与标准库系统
- REPL 与 CLI
- 打包与分发能力

---

## 2. 总体分层

### 2.1 CLI 与用户入口层
目录：
- `cmd/icoo/`

职责：
- 解析命令行参数
- 调用运行时 API
- 提供项目初始化、打包、构建、提取、检查、运行、REPL 等入口

入口文件：
- `cmd/icoo/main.go:12`

命令分发：
- `build` / `bundle` / `extract` / `inspect`
- `check` / `run`
- `init` / `repl`

这一层尽量薄，不直接承载语言逻辑，而是把工作委托给 `pkg/api` 或辅助函数。

---

### 2.2 API 门面层
目录：
- `pkg/api/`

核心文件：
- `pkg/api/runtime.go:17`

职责：
- 封装“词法→语法→语义→编译→执行”的完整主链
- 为 CLI 和测试提供稳定调用接口
- 持有运行时对象 `Runtime`

主要接口：
- `NewRuntime()`：创建运行环境
- `CheckSource()` / `CheckFile()`
- `RunSource()` / `RunFile()`
- `InvokeGlobal()`
- `RunReplLine()`

这一层相当于整个语言引擎的应用服务层，屏蔽了 `internal/*` 的实现细节。

---

### 2.3 语言前端
目录：
- `internal/token/`
- `internal/lexer/`
- `internal/ast/`
- `internal/parser/`
- `internal/sema/`

职责拆分：

#### `internal/token`
定义 token、位置、span、关键字表。

#### `internal/lexer`
将源码切分为 token 流。

#### `internal/ast`
定义语言的抽象语法树节点：
- 声明
- 语句
- 表达式
- 类型表达式

#### `internal/parser`
把 token 流解析成 AST：
- 顶层声明
- 表达式优先级
- 错误恢复

入口：
- `internal/parser/parser.go:19`

#### `internal/sema`
做基础语义分析：
- 作用域
- 标识符定义与引用
- 声明冲突
- 一些结构性合法性检查

这部分整体上是经典编译器前端结构，边界相对清晰。

---

### 2.4 编译层
目录：
- `internal/compiler/`
- `internal/bytecode/`

职责：

#### `internal/bytecode`
定义 VM 执行协议：
- opcode
- chunk
- 常量池

#### `internal/compiler`
把 AST 编译为字节码模块。

入口：
- `internal/compiler/compiler.go:79`

编译器内部维护：
- 当前函数编译上下文
- 局部变量槽位
- 作用域深度
- 循环上下文
- try/finally 上下文
- upvalue 捕获

这是一个“有状态单遍生成字节码”的编译器，而不是 IR 多阶段编译器。

---

### 2.5 运行时与虚拟机
目录：
- `internal/runtime/`
- `internal/vm/`

职责：

#### `internal/runtime`
定义运行时值系统和辅助逻辑：
- 基础值
- 对象、数组、函数、闭包、类等
- 值比较、truthy 判断、字符串化等

#### `internal/vm`
执行字节码：
- 栈
- 调用帧
- 异常处理器栈
- 全局变量
- 模块缓存
- 内建函数注册
- goroutine pool

核心对象：
- `internal/vm/vm.go:29` `type VM struct`

这部分是整个项目的核心执行引擎。

---

### 2.6 标准库与宿主能力层
目录：
- `internal/stdlib/`

模块注册入口：
- `internal/stdlib/modules.go:13`

按功能拆为多个子域：
- `core`
- `data`
- `database`
- `format`
- `net`
- `system`
- `express`

职责：
- 把 Go 宿主能力封装为 Icoo 模块
- 通过 `LoadModule(spec)` 暴露为标准库导入

---

## 3. 运行主链分析

### 3.1 文件执行主链

`pkg/api/runtime.go:67` `RunFile()`：

```text
读取源码
  → runModuleSource()
    → lexer.LexAll
    → parser.New(...).ParseProgram
    → sema.Analyze
    → compiler.Compile
    → vm.RunModule
```

关键位置：
- `pkg/api/runtime.go:141`

这条链是项目最核心的执行路径。

---

### 3.2 文件检查主链

`pkg/api/runtime.go:59` `CheckFile()`：

```text
读取文件
  → CheckSource()
    → LexAll
    → ParseProgram
    → sema.Analyze
```

特点：
- `check` 不进入编译/执行阶段
- 当前主要用于静态错误发现

---

### 3.3 REPL 主链

`pkg/api/runtime.go:97` `RunReplLine()`

REPL 与文件执行的差异：
- 纯表达式会自动包装成 `return ...`
- 跳过 sema
- 直接编译并执行单行 AST

这说明 REPL 现在更偏“轻量交互入口”，而不是完整增量编译环境。

---

## 4. 模块系统架构

### 4.1 模块来源

模块分两类：

#### 1）标准库模块
通过：
- `internal/stdlib/modules.go:13`
- `pkg/api/runtime.go:172`

处理

如果 `spec` 命中 `std.*`，直接从标准库返回模块对象。

#### 2）文件模块
未命中标准库时，走：
- `pkg/api/runtime.go:176`

解析真实文件路径并加载源码。

---

### 4.2 模块加载流程

`pkg/api/runtime.go:171` `loadModule()`

流程：
1. 先查 stdlib
2. 解析模块路径
3. 查模块缓存 `r.modules`
4. 读取源码
5. 为子模块创建新的 VM
6. 重新注册 builtins 和 module loader
7. parse → sema → compile → run
8. 收集模块导出表

这是一个 **递归模块加载模型**。

---

### 4.3 模块缓存策略

缓存存放于：
- `pkg/api/runtime.go:19` `modules map[string]*runtime.Module`

特点：
- 按 resolved path 缓存
- 在执行前先写入缓存，再运行模块
- 运行失败时删除缓存项：`pkg/api/runtime.go:220`

优点：
- 能避免重复加载
- 能为循环依赖留出基础结构

潜在点：
- 当前是否完整支持复杂循环依赖，还需要更多系统性验证

---

### 4.4 项目根别名导入

相关逻辑：
- `pkg/api/runtime.go:242`
- `cmd/icoo/project.go:325`

设计：
- `project.toml` 可配置 `root_alias`
- 导入时若匹配该别名，则映射到项目根内路径
- 明确禁止越出项目根目录

这是对大型项目导入体验很实用的设计。

---

## 5. 编译器架构分析

### 5.1 编译器模型

编译器入口：
- `internal/compiler/compiler.go:79`

整体模型：
- 以模块为单位编译
- 顶层生成 `__module_init__`
- 每个函数生成对应 `FunctionProto`
- 编译时维护嵌套函数上下文

关键结构：
- `Compiler`
- `FuncCompiler`
- `Local`
- `LoopContext`
- `TryContext`

这是典型的基于上下文栈的字节码编译器。

---

### 5.2 局部变量与作用域

局部变量结构：
- `internal/compiler/compiler.go:10`

特征：
- 局部变量有 `Slot`
- 用 `Depth` 跟踪作用域层级
- 用 `IsConst` 跟踪是否常量

作用域退出时会做清理：
- `emitScopeCleanup()`：`internal/compiler/compiler.go:308`

说明编译器对局部槽位和栈平衡是显式控制的。

---

### 5.3 控制流编译

支持：
- jump
- conditional jump
- loop back edge
- break/continue patching

关键方法：
- `emitJump()`：`internal/compiler/compiler.go:156`
- `patchJump()`：`internal/compiler/compiler.go:169`
- `emitLoop()`：`internal/compiler/compiler.go:175`
- `patchBreakJumps()`：`internal/compiler/compiler.go:196`

控制流处理思路是直接字节码回填，没有额外 CFG 抽象层。

---

### 5.4 异常与 finally 编译

编译器中有专门的：
- `TryContext`：`internal/compiler/compiler.go:45`
- `FinallyAction`：`internal/compiler/compiler.go:37`

这表明 `try/catch/finally` 并不是“语法解析了但没落地”，而是已经有专门的控制流建模。

这是当前项目中一个复杂度较高的点，因为 finally 需要和：
- return
- break
- continue
- exception

交互。

---

## 6. VM 架构分析

### 6.1 VM 核心状态

`internal/vm/vm.go:29`

核心字段：
- `stack []runtime.Value`
- `frames []CallFrame`
- `handlers []ExceptionHandler`
- `globals map[string]runtime.Value`
- `builtins map[string]runtime.Value`
- `modules map[string]*runtime.Module`
- `openUpvalues map[int]*runtime.Upvalue`

这是标准的栈式虚拟机设计。

---

### 6.2 调用帧模型

`CallFrame`：
- 当前 closure
- 当前 module
- receiver / super
- 指令指针 IP
- 栈基址 Base

说明 VM 已支持：
- 普通函数调用
- 方法调用
- 类继承上下文
- 模块执行上下文

---

### 6.3 闭包实现

相关逻辑：
- `captureUpvalue()`：`internal/vm/vm.go:179`
- `closeUpvalues()`：`internal/vm/vm.go:188`

说明闭包通过“开放上值 + 关闭时复制”的经典方案实现。

优点：
- 模型成熟
- 对嵌套函数友好

需要关注：
- 更复杂捕获场景下的边界测试是否足够

---

### 6.4 异常处理模型

VM 中显式维护：
- `handlers []ExceptionHandler`

说明异常处理不是靠 Go panic 直接穿透，而是语言级 handler 栈管理。  
这对 `try/catch/finally` 的语义一致性是好事。

---

### 6.5 并发执行模型

`internal/vm/vm.go:61` `Pool()`  
`internal/vm/vm.go:68` `goExecutor()`

设计特点：
- 使用 `internal/concurrency` 提供 goroutine pool
- `go` 任务会复制 globals/modules 快照到子 VM
- 在子 VM 中运行 closure / native function / bound method

这说明并发模型不是共享一个活动 VM，而是更接近：
- 主 VM 调度
- 子任务隔离执行

这是一个相对稳妥的设计方向，因为能减少并发读写共享解释器状态的复杂度。

---

## 7. 标准库架构分析

### 7.1 模块注册模式

入口：
- `internal/stdlib/modules.go:13`

采用集中分发表：

```go
func LoadModule(spec string) (*runtime.Module, bool)
```

优点：
- 入口清晰
- 标准库模块枚举显式
- 易于控制暴露面

不足：
- 模块数继续增长时，单个 switch 会膨胀

---

### 7.2 标准库分域

当前拆分方式比较合理：

- `core`：基础内建能力
- `format`：JSON/YAML/TOML/XML
- `system`：fs/exec/os/host
- `net`：http/websocket/sse/socket
- `database`：db
- `data`：crypto/uuid/compress
- `express`：web 框架封装

这种按宿主能力域拆分的方式便于后续维护。

---

### 7.3 标准库的架构角色

标准库在这个项目里不只是“附带示例”，而是语言能力的一部分。  
原因是很多语言价值来自宿主集成：

- 文件系统
- 网络协议
- 数据库
- 系统命令
- 服务端开发

换句话说，Icoo 的可用性高度依赖 stdlib 质量。

---

## 8. 工具链架构分析

### 8.1 项目初始化

`cmd/icoo/project.go:48` `runInit()`

生成：
- `project.toml`
- 默认 entry 脚本

作用：
- 引导项目目录结构
- 定义入口和 root alias

---

### 8.2 Bundle

`cmd/icoo/bundle.go:47` `buildBundleArchive()`

流程：
1. 解析入口
2. 遍历 import 图
3. 收集源码
4. 计算相对路径
5. 生成 `BundleArchive`

这是源码级 bundle，不是字节码级 bundle。

优点：
- 可调试
- 格式直观
- 跨平台简单

代价：
- 运行时仍要重新 parse/compile

---

### 8.3 Build

`cmd/icoo/build.go:48` `runBuild()`

机制：
- 先产出 bundle archive
- 再将 bundle 追加到当前 CLI 可执行 stub
- 最终产出可分发 exe

Windows 下还支持资源写入：
- icon
- version
- product name
- file description 等

这部分体现出较强的产品化导向。

---

### 8.4 Extract / Inspect

这两个命令补足了 build/bundle 的可观察性：
- `extract`：从 bundle/exe 中提取归档
- `inspect`：查看 bundle 内容和元数据

从架构上讲，这是很好的运维/调试辅助工具。

---

## 9. 代码组织优点

### 9.1 分层边界清晰
`cmd`、`pkg/api`、`internal/*` 的分工比较明确，便于理解与演进。

### 9.2 语言主链干净
`pkg/api/runtime.go:141` 的执行链结构非常直接，定位问题相对容易。

### 9.3 功能增长没有完全打乱目录结构
虽然语言特性增长很快，但整体仍维持了：
- 前端
- 编译器
- VM
- 标准库
- CLI

这种清晰分域。

### 9.4 工具链意识强
不是只停留在 `run`，而是覆盖了：
- init
- repl
- bundle
- build
- inspect
- extract

---

## 10. 当前架构风险

### 10.1 文档与实现脱节
文档仍停留在较早阶段，而代码已经实现了更多高级特性。  
这会影响新开发者对系统真实边界的判断。

---

### 10.2 语言特性交互复杂度快速上升
当前系统已包含：
- class/inheritance
- decorators
- exceptions/finally
- go/select
- type/interface

单项实现不算最难，难点在它们的组合语义。  
架构上需要更多“特性交互测试”来支撑。

---

### 10.3 前端与编译层白盒测试偏少
虽然端到端测试通过，但：
- lexer
- parser
- sema
- compiler

单元测试不足，长期会影响定位效率。

---

### 10.4 标准库增长快于统一约束
模块很多，但架构上还需要更统一地定义：
- 错误模型
- 返回值风格
- 资源生命周期
- 命名一致性

---

### 10.5 bundle 仍基于源码
这是当前阶段的合理选择，但未来如果项目继续产品化，可能会考虑：
- 字节码级 bundle
- 混淆/压缩策略
- 启动速度优化
- 版本兼容策略

---

## 11. 对开发者的建议

### 11.1 阅读顺序建议
建议按下面顺序理解项目：

1. `cmd/icoo/main.go`
2. `pkg/api/runtime.go`
3. `internal/parser/`
4. `internal/sema/`
5. `internal/compiler/`
6. `internal/vm/`
7. `internal/stdlib/modules.go`
8. `cmd/icoo/bundle.go` / `build.go`

这样最容易建立整体心智模型。

---

### 11.2 开发新语言特性时的落点
新增语法一般需要同步修改：

- `internal/token`
- `internal/lexer`
- `internal/ast`
- `internal/parser`
- `internal/sema`
- `internal/compiler`
- `internal/vm`
- `pkg/api/*_test.go`
- `examples/`
- `docs/`

这是一个典型的“全链路变更”项目。

---

### 11.3 更适合优先补强的方向
比起继续快速加语法，更建议优先做：

1. 文档同步
2. parser/sema/compiler 单元测试
3. 高级特性交互测试
4. 标准库 API 一致性整理
5. 错误信息和调试体验优化

---

## 12. 结论

Icoo 当前架构已经具备一个小型语言实现项目应有的主要骨架，并且在工具链和标准库上走得比很多同类项目更远。

从开发者视角看，它的核心特征是：

- **分层清楚**
- **主链完整**
- **可扩展性不错**
- **工具链意识强**
- **高级特性增长较快**

当前最值得投入的不是继续堆功能，而是把已有架构“固化”下来：
- 文档对齐
- 测试补强
- 语义收敛
- 标准库统一

这样后续继续扩展 class、并发、类型系统时，成本会低很多。
