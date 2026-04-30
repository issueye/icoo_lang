# Icoo MVP 实施路线

## 目标

本文档定义 Icoo 从空仓库启动到首个可运行 MVP 的实施路线。MVP 的目标不是一次性实现完整语言，而是尽快形成一个可验证的最小闭环：

- 可以解析源码
- 可以完成基础语义检查
- 可以编译到字节码
- 可以通过 VM 执行最小脚本
- 可以支持函数、变量、条件、循环、数组、对象、调用

首版重点是把编译链路和运行时架构打通，为后续 `match`、`try/catch`、`go/select`、`type/interface` 扩展留好边界。

---

## MVP 范围

### 首批必须支持的语言能力

- `const`
- `let`
- `fn`
- `return`
- `if / else`
- `while`
- `for`
- `for in`
- 基础表达式
- 函数调用
- 数组字面量
- 对象字面量
- 成员访问
- 下标访问
- 顶层函数与变量声明
- 基础 `import / export`
- 统一迭代器协议（`iter()` / `next()`）

### 暂不纳入 MVP

- `try / catch`
- `go`
- `select`
- `type / interface`
- 完整闭包优化

---

## 成功标准

当以下条件都满足时，视为 MVP 完成：

1. `icoo check demo.ic` 能完成词法、语法、基础语义检查
2. `icoo run demo.ic` 能运行最小脚本
3. 支持函数定义与调用
4. 支持 `if/else`、`for` 与 `while`
5. 支持数组、对象与 `for-in` 迭代
6. 支持基础 `import/export`
7. 支持 builtin：`print`、`println`、`len`、`typeOf`
8. `go test ./...` 能通过核心包测试

---

## 推荐目录结构

```text
icoo_lang/
  go.mod
  cmd/
    icoo/
      main.go
  internal/
    token/
      token.go
    lexer/
      lexer.go
    ast/
      ast.go
      decl.go
      stmt.go
      expr.go
    parser/
      parser.go
      parser_decl.go
      parser_stmt.go
      parser_expr.go
      precedence.go
    diag/
      diagnostic.go
    sema/
      sema.go
      scope.go
    bytecode/
      opcode.go
      chunk.go
    compiler/
      compiler.go
      compile_decl.go
      compile_stmt.go
      compile_expr.go
      scope.go
      symbol.go
    runtime/
      value.go
      helpers.go
    vm/
      vm.go
      vm_run.go
      vm_call.go
      vm_ops.go
    stdlib/
      builtin.go
      io.go
  pkg/
    api/
      runtime.go
  testdata/
    lexer/
    parser/
    integration/
  docs/
    language-design.md
    mvp-roadmap.md
```

---

## Phase 0：项目骨架

### 目标

初始化 Go 工程与 CLI 外壳。

### 需要创建

- `go.mod`
- `cmd/icoo/main.go`
- `internal/` 各核心目录
- `pkg/api/runtime.go`

### 完成标准

- `go test ./...` 能运行空测试
- `go run ./cmd/icoo --help` 或最小命令入口可执行

### CLI 首批命令

建议只做：
- `icoo check <file>`
- `icoo run <file>`

`build`、`repl` 暂缓。

---

## Phase 1：Token 与 Lexer

### 目标

把源码稳定切分为 token 流。

### 需要实现

#### `internal/token/token.go`
定义：
- `Type`
- `Position`
- `Span`
- `Token`
- `Keywords`

#### `internal/lexer/lexer.go`
实现：
- 空白处理
- 单行注释 `//`
- 标识符与关键字
- 数字
- 字符串
- 操作符与分隔符

### 首批支持 token

- 字面量：`Ident`, `Int`, `Float`, `String`
- 运算符：`= + - * / % == != < <= > >= && || !`
- 分隔符：`. , : ; ( ) { } [ ]`
- 关键字：`fn return if else while const let true false null`

### 测试

在 `testdata/lexer/` 放输入/输出用例，覆盖：
- 关键字识别
- 字符串
- 数字
- 注释
- 错误 token

### 完成标准

- 能将一段合法源码转成稳定 token 列表
- 错误位置准确

---

## Phase 2：AST 与 Parser

### 目标

把 token 流转成 AST。

### 需要实现

#### `internal/ast/`
定义：
- `Node`, `Expr`, `Stmt`, `Decl`
- `Program`
- `VarDecl`, `FnDecl`
- `BlockStmt`, `IfStmt`, `WhileStmt`, `ReturnStmt`, `ExprStmt`
- `IdentExpr`, `Literal`, `UnaryExpr`, `BinaryExpr`, `AssignExpr`
- `CallExpr`, `MemberExpr`, `IndexExpr`, `ArrayLiteral`, `ObjectLiteral`, `FnExpr`

#### `internal/parser/`
实现：
- 顶层声明解析
- 语句解析
- Pratt Parser 表达式解析
- 错误恢复 `synchronize()`

### MVP 支持的语法

- 顶层 `const/let/fn`
- block
- `if/else`
- `while`
- `return`
- 匿名函数
- 数组/对象字面量
- 成员/下标访问
- 赋值表达式

### 测试

在 `testdata/parser/` 放：
- precedence 用例
- AST 快照测试
- 错误恢复测试

### 完成标准

- 能把 MVP 子集源码解析为稳定 AST
- 对错误源码能继续同步解析并给出合理诊断

---

## Phase 3：基础语义分析

### 目标

在执行前发现最关键的静态错误。

### 需要实现

#### `internal/sema/scope.go`
- 词法作用域
- 符号注册
- 作用域入栈/出栈

#### `internal/sema/sema.go`
检查：
- 重名声明
- 未定义标识符
- `return` 是否在函数内部
- `const` 是否被重新赋值

### MVP 暂不做

- 完整类型推导
- interface/type 检查
- match/select/try 规则

### 测试

- 未定义变量
- 块级作用域覆盖
- const 重赋值
- 非函数体 return

### 完成标准

- `icoo check` 能对常见静态错误给出明确提示

---

## Phase 4：字节码定义与 Chunk

### 目标

定义 VM 执行协议。

### 需要实现

#### `internal/bytecode/opcode.go`
定义 MVP opcode：

```text
OpConstant
OpNull
OpTrue
OpFalse
OpPop
OpGetLocal
OpSetLocal
OpGetGlobal
OpDefineGlobal
OpSetGlobal
OpAdd
OpSub
OpMul
OpDiv
OpMod
OpNegate
OpNot
OpEqual
OpGreater
OpLess
OpJump
OpJumpIfFalse
OpLoop
OpCall
OpClosure
OpReturn
OpArray
OpObject
OpGetProperty
OpSetProperty
OpGetIndex
OpSetIndex
```

#### `internal/bytecode/chunk.go`
实现：
- `Chunk`
- `Write`
- `WriteShort`
- `AddConstant`

### 完成标准

- 编译器可向 `Chunk` 正确发码
- 常量池索引稳定

---

## Phase 5：运行时 Value 与 Builtins

### 目标

建立最小运行时值系统。

### 需要实现

#### `internal/runtime/value.go`
- `Value`
- `ValueKind`
- `NullValue`
- `BoolValue`
- `IntValue`
- `FloatValue`
- `StringValue`
- `ArrayValue`
- `ObjectValue`
- `NativeFunction`
- `Closure`

#### `internal/runtime/helpers.go`
- `IsTruthy`
- `ValueEqual`
- 字符串化辅助

#### `internal/stdlib/builtin.go`
首批 builtin：
- `print`
- `println`
- `len`
- `typeOf`

### 完成标准

- VM 已有可执行的值系统
- builtin 已可调用

---

## Phase 6：Compiler MVP

### 目标

把 AST 编译成字节码。

### 需要实现

#### `internal/compiler/compiler.go`
- `Compiler`
- `FuncCompiler`
- `CompiledModule`

#### `internal/compiler/scope.go`
- `Local`
- `beginScope/endScope`
- slot 管理

#### `internal/compiler/symbol.go`
- local/global 解析
- 可先不做完整 upvalue

#### `compile_decl.go`
- `const/let`
- `fn`

#### `compile_stmt.go`
- `block`
- `if/else`
- `while`
- `return`
- `expr stmt`

#### `compile_expr.go`
- literal
- identifier
- unary/binary
- assignment
- call
- member/index
- array/object
- anonymous function

### MVP 策略

闭包结构先预留，但首版可以：
- 先支持函数对象
- 暂不实现完整 upvalue 捕获

### 完成标准

- 能生成可执行字节码
- `if/while/function/call` 链路打通

---

## Phase 7：VM Run Loop MVP

### 目标

运行字节码。

### 需要实现

#### `internal/vm/vm.go`
- `VM`
- `CallFrame`
- `Push/Pop/Peek`
- `NewVM`

#### `internal/vm/vm_run.go`
- `Run`
- `runLoop`
- `readByte/readShort/readConstant`

#### `internal/vm/vm_call.go`
- `CallValue`
- `callClosure`
- `callNative`
- `returnFromFrame`

#### `internal/vm/vm_ops.go`
- 算术
- 比较
- truthiness
- 属性访问
- 下标访问

### 完成标准

- 能执行最小脚本
- 能调用 builtin
- 能调用用户函数
- 局部/全局变量读写稳定

---

## Phase 8：CLI 打通 check / run

### 目标

把前端、编译器、VM 串起来。

### `icoo check <file>`
执行：
1. 读文件
2. lexer
3. parser
4. sema
5. 输出诊断

### `icoo run <file>`
执行：
1. 读文件
2. lexer
3. parser
4. sema
5. compiler
6. VM run

### 完成标准

- `icoo check demo.ic` 可工作
- `icoo run demo.ic` 可工作

---

## Phase 9：集成测试与示例脚本

### 目标

验证端到端行为。

### 建议测试脚本

#### `testdata/integration/basic.ic`
验证：
- 变量
- 算术
- if/else
- while

#### `testdata/integration/function.ic`
验证：
- 函数定义
- 参数
- return
- 嵌套调用

#### `testdata/integration/object.ic`
验证：
- 数组
- 对象
- 成员访问
- 下标访问

### 完成标准

- 关键集成脚本都能稳定通过
- 输出与预期一致

---

## 推荐实施顺序总表

1. 项目骨架
2. token + lexer
3. AST + parser
4. sema
5. opcode + chunk
6. runtime values + builtins
7. compiler
8. VM
9. CLI check/run
10. integration tests

---

## 每阶段的“不要做”

为了保证 MVP 不失控，每阶段要明确克制范围。

### 在 MVP 阶段不要做这些

- 不做 `match`
- 不做 `try/catch`
- 不做 `go/select`
- 不做完整模块导入缓存
- 不做反射式 host object
- 不做复杂类型系统
- 不做泛型
- 不做 REPL
- 不做优化器
- 不做调试器

---

## MVP 完成后第一批扩展建议

当 MVP 稳定后，再按这个顺序扩展：

1. 完整模块系统：`import/export`
2. 闭包与 upvalue
3. `match`
4. `try/catch`
5. `chan`
6. `go`
7. `select`
8. `type/interface`
9. 标准库扩展
10. 宿主 API 强化

---

## 最终结论

Icoo 的 MVP 不应该追求“特性很多”，而应该优先完成一条稳定主链：

```text
源码 -> token -> AST -> sema -> bytecode -> VM -> CLI
```

只要这条主链稳定，后续高级特性都能按层叠加；如果这条主链不稳，越早加高级特性，后面返工越大。
