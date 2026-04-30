# Icoo MVP 状态总览

本文档用于收口当前 MVP 的实际状态，明确：

- 已实现能力
- 尚未实现或仍受限的能力
- 当前验证方式
- 建议的下一阶段方向

## 当前结论

截至目前，Icoo 的 MVP 主链已经打通：

```text
源码 -> Lexer -> Parser -> AST -> 语义分析 -> 字节码 -> VM -> CLI
```

已经具备：

- `icoo check <file>`
- `icoo run <file>`
- 多文件 `import/export`
- `for` / `for-in` / `match`
- 统一 iterator 协议
- 基础标准库模块

## 已实现能力

### 1. CLI 与执行主链

已实现：

- `icoo check <file>`
- `icoo run <file>`

入口文件：

- `cmd/icoo/main.go`
- `pkg/api/runtime.go`

### 2. 前端：词法 / 语法 / AST

已实现：

- 基础 token
- 关键字识别
- 表达式优先级解析
- 顶层声明
- 顶层表达式语句
- block / `if` / `while` / `for`
- `for-in`
- `match`
- 函数声明 / 匿名函数
- `import` / `export`

关键目录：

- `internal/token/`
- `internal/lexer/`
- `internal/ast/`
- `internal/parser/`

### 3. 语义分析

已实现：

- 基础作用域分析
- 未定义标识符检查
- 重复声明检查
- `return` 合法性检查
- `for-in` 绑定检查

关键目录：

- `internal/sema/`

### 4. 编译器与 VM

已实现：

- 基础表达式编译
- 函数调用
- 条件与循环
- 数组 / 对象
- 属性访问 / 下标访问
- 顶层模块执行
- 文件模块加载
- 导出表

关键目录：

- `internal/bytecode/`
- `internal/compiler/`
- `internal/runtime/`
- `internal/vm/`

### 5. `for-in` 与 iterator 协议

已实现：

- `iter()` / `next()` 协议
- 单绑定：`for item in iterable`
- 双绑定：`for key, value in iterable`
- `_` 忽略绑定
- array / string / object / module / iterator 的默认迭代行为
- 对象自定义 `iter` 覆盖默认行为

相关文档：

- `docs/language-design.md`
- `docs/iterators.md`

### 6. 标准库模块

当前已实现：

- `std.io`
  - `print`
  - `println`
- `std.time`
  - `now`
  - `sleep`
- `std.math`
  - `abs`
  - `max`
  - `min`
  - `floor`
  - `ceil`
- `std.json`
  - `encode`
  - `decode`
- `std.fs`
  - `readFile`
  - `writeFile`
  - `exists`

当前结构已经按“每个原生库一个单元”拆分：

- `internal/stdlib/modules.go`
- `internal/stdlib/io.go`
- `internal/stdlib/time.go`
- `internal/stdlib/math.go`
- `internal/stdlib/json.go`
- `internal/stdlib/fs.go`

## 当前仍未实现 / 不完整部分

以下能力仍不应视为当前 MVP 已完成：

### 1. 并发能力

未实现：

- `go`
- `select`
- channel 语言级能力

虽然设计文档已有方向，但当前运行时还没有这一批特性。

### 2. 异常处理

未实现：

- `try`
- `catch`
- 统一脚本级异常模型

当前 `panic(...)` 仍主要通过宿主错误回传。

### 3. 类型系统

未实现：

- `type`
- `interface`
- 更完整的静态类型约束

### 4. 更完整的闭包/高级运行时能力

仍受限：

- 完整闭包优化未完成
- 更强的模块缓存/循环依赖细节未系统化验证
- 还没有 REPL / build / debugger

### 5. 标准库仍是最小集

当前标准库偏 MVP，用于验证运行时闭环，尚未形成稳定的长期 API 面。

## 当前验证方式

### 单元与运行时测试

通过：

```bash
go test ./...
```

重点覆盖：

- 运行时调用
- import/export
- `for` / `for-in`
- iterator 协议
- `match`
- 标准库模块

### 集成脚本

当前已存在：

- `testdata/integration/basic.ic`
- `testdata/integration/import_main.ic`
- `testdata/integration/iterators.ic`
- `testdata/integration/stdlib.ic`

其中 `stdlib.ic` 会覆盖：

- `std.io`
- `std.time`
- `std.math`
- `std.json`
- `std.fs`
- `icoo run` 端到端执行链路

## 对 MVP 的判断

如果把 MVP 的定义限定为：

- 有完整主链
- 能执行脚本
- 有模块能力
- 有最小标准库
- 有基本测试和 integration 覆盖

那么当前状态已经可以视为 **MVP 基本完成**。

如果把 MVP 的定义扩大到：

- 并发
- 异常
- 类型系统
- 更完整工具链

那么这些属于 **MVP 之后的下一阶段**，不应继续混入 MVP 范围。

## 建议的下一阶段

建议从这里开始，不再继续扩大 MVP，而是进入“阶段 2”开发。

优先顺序建议：

1. `try/catch`
2. 更完整的模块系统边界与错误处理
3. `go/select` 与 channel
4. `type/interface`
5. CLI 扩展（如 `build` / `repl`）

## 相关文档

- `docs/mvp-roadmap.md`
- `docs/language-design.md`
- `docs/iterators.md`
