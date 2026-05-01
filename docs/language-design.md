# Icoo 语言说明

相关文档：

- AI 代码生成参考：`docs/ai-language-api.md`
- Runtime API：`docs/api.md`
- 当前状态：`docs/mvp-status.md`
- 架构分析：`docs/architecture-analysis-report.md`

---

## 目标

Icoo 是一门用 Golang 实现的编译型脚本语言，目标是结合：

- Go 的简洁：少量关键字、清晰控制流、工程化友好
- JavaScript 的灵活：动态值、一等函数、对象/数组友好、脚本式开发体验

推荐执行模型：

```text
源码 -> Lexer -> Parser -> AST -> 语义分析 -> 字节码 -> VM 执行
```

---

## 当前实现概览

结合当前代码与 `docs/architecture-analysis-report.md`，Icoo 已经不是只覆盖最小表达式求值的原型，而是一套包含语言前端、字节码 VM、模块系统、标准库和 CLI 工具链的完整实现。

当前主链如下：

```text
源码
  -> Lexer
  -> Parser
  -> AST
  -> Sema
  -> Compiler
  -> Bytecode
  -> VM
  -> 标准库 / 模块系统 / CLI
```

从代码组织看，项目主要分为几层：

- `cmd/icoo/`
  - CLI 入口与命令分发
  - 提供 `init`、`check`、`run`、`repl`、`bundle`、`build`、`extract`、`inspect`
- `pkg/api/`
  - 面向 CLI 与测试的运行时门面
  - 封装检查、执行、模块加载、bundle 执行等主能力
- `internal/token` / `lexer` / `ast` / `parser` / `sema`
  - 语言前端与基础语义分析
- `internal/compiler` / `internal/bytecode`
  - AST 到字节码的单遍编译
- `internal/runtime` / `internal/vm`
  - 值系统、闭包、类、异常、并发、模块执行
- `internal/stdlib`
  - 标准库模块与内建函数注册

如果你是第一次阅读项目，建议按这个顺序建立整体心智模型：

1. `cmd/icoo/main.go`
2. `pkg/api/runtime.go`
3. `internal/parser/`
4. `internal/sema/`
5. `internal/compiler/`
6. `internal/vm/`
7. `internal/stdlib/modules.go`
8. `cmd/icoo/bundle.go` / `cmd/icoo/build.go`

---

## 当前语言能力

下面这部分描述的是**当前实现中已经可用或已经落到代码中的能力**，不是远期设想。

### 基础数据与表达式

已实现：

- `null` / `bool` / `int` / `float` / `string`
- `array` / `object`
- 一元与二元表达式
- 赋值表达式
- 逻辑短路：`&&` / `||`
- 三元表达式：`cond ? a : b`
- 成员访问：`obj.name`
- 下标访问：`arr[i]`
- 函数调用：`fn(...)`

### 变量、函数与闭包

已实现：

- `let` / `const`
- 命名函数与匿名函数
- 闭包捕获
- 嵌套函数
- 顶层表达式语句

### 控制流

已实现：

- `if / else`
- `while`
- `for`
- `for-in`
- `break` / `continue`
- `return`
- `match`

### 模块系统

已实现：

- `import`
- `export`
- 文件模块加载
- 标准库模块加载
- 模块缓存
- 项目根别名导入（通过 `project.toml` 的 `root_alias` 配置）

### 错误与异常

已实现：

- `throw`
- `try / catch / finally`
- `error(...)` 内建错误值
- 后缀 try 表达式：`expr?`

`expr?` 的语义是：

- 如果表达式结果不是 `error`，直接返回该值
- 如果结果是 `error`，则向外提前传播该错误值
- 如果外层有 `finally`，会先执行 `finally`

示例：

```icoo
fn parsePort(text) {
  if text == "3000" {
    return 3000
  }
  return error("invalid port: " + text)
}

fn loadPort(text) {
  let port = parsePort(text)?
  return port + 1
}
```

### 类型与接口

已实现：

- 类型别名：`type UserID = int`
- 接口声明：`interface Greeter { ... }`
- `satisfies(value, InterfaceName)` 运行时检查

当前类型系统更偏**声明与约束表达**，而不是完整静态类型推导系统。

### 类、继承与装饰器

已实现：

- `class`
- `this`
- `init(...)`
- 实例方法
- 单继承：`class Dog < Animal`
- `super.init(...)`
- `super.method(...)`
- 函数装饰器
- 类装饰器
- 方法装饰器

示例：

```icoo
class Animal {
  init(name) {
    this.name = name
  }

  speak() {
    return this.name + " makes a sound"
  }
}

class Dog < Animal {
  init(name, breed) {
    super.init(name)
    this.breed = breed
  }

  speak() {
    return super.speak() + " and barks"
  }
}
```

### 并发

已实现并采用对象风格 channel API：

```icoo
let ch = chan()
let ch = chan(8)

ch.send(v)
let v, ok = ch.recv()
let ok = ch.trySend(v)
let v, ok = ch.tryRecv()
ch.close()
```

并发启动：

```icoo
go worker(ch)
```

`select` 语法：

```icoo
select {
  recv ch1 as msg {
    print(msg)
  }

  send ch2, 1 {
    print("sent")
  }

  else {
    print("idle")
  }
}
```

运行时实现上，`go` 任务不是共享同一个活动 VM，而是由主 VM 调度、在子 VM 中隔离执行闭包或原生函数。这能减少解释器内部共享状态带来的并发复杂度。

### 标准库与宿主能力

当前标准库已按能力域拆分，已注册的模块包括：

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

示例可参考 `examples/README.md`。

---

## 工具链与执行入口

当前 CLI 入口提供以下命令：

- `icoo`
- `icoo repl`
- `icoo init [dir] [--entry path] [--entry-fn name] [--root-alias name]`
- `icoo check <file|dir>`
- `icoo run <file|dir>`
- `icoo bundle <file|dir> [output]`
- `icoo build <file|dir> [output] [--metadata file]`
- `icoo extract <bundle|executable> [output]`
- `icoo inspect <bundle|executable>`

其中：

- `check` 走词法、语法、语义分析，不进入编译执行
- `run` 走完整主链：词法 -> 语法 -> 语义 -> 编译 -> VM 执行
- `repl` 会对纯表达式自动包装 `return ...`，以便直接回显结果
- `bundle` 打的是**源码级归档**，不是字节码级归档
- `build` 会把 bundle 追加到 CLI stub，生成可分发执行文件

---

## 并发风格

采用对象风格 channel API，而不是 Go 的 `<-` 操作符。

```icoo
let ch = chan()
ch.send(1)
let x, ok = ch.recv()
```

推荐 API：

```icoo
let ch = chan()
let ch = chan(8)

ch.send(v)
let v, ok = ch.recv()
let ok = ch.trySend(v)
let v, ok = ch.tryRecv()
ch.close()
```

并发启动：

```icoo
go worker(ch)
```

select 建议语法：

```icoo
select {
  recv ch1 as msg {
    print(msg)
  }

  send ch2, 1 {
    print("sent")
  }

  else {
    print("idle")
  }
}
```

---

## EBNF 草案

### 程序结构

```ebnf
Program         = { TopLevelDecl } EOF ;
TopLevelDecl    = ImportDecl
                | ExportDecl
                | FnDecl
                | VarDecl
                | TypeDecl
                | InterfaceDecl
                | ClassDecl
                | DecoratedDecl
                | Statement ;
```

### 导入导出

```ebnf
ImportDecl      = "import" ImportPath [ "as" Identifier ] ;
ImportPath      = Identifier { "." Identifier } | String ;

ExportDecl      = "export" ( FnDecl | VarDecl | TypeDecl | InterfaceDecl ) ;
```

### 变量与函数声明

```ebnf
VarDecl         = ( "const" | "let" ) Identifier [ TypeAnnotation ] "=" Expression ;

FnDecl          = "fn" Identifier "(" [ ParamList ] ")" [ ReturnType ] Block ;
FnExpr          = "fn" "(" [ ParamList ] ")" [ ReturnType ] Block ;

ParamList       = Param { "," Param } ;
Param           = Identifier [ TypeAnnotation ] ;

TypeAnnotation  = ":" TypeExpr ;
ReturnType      = ":" TypeExpr ;
```

### 类型声明、类与装饰器

```ebnf
TypeDecl        = "type" Identifier "=" TypeExpr ;

InterfaceDecl   = "interface" Identifier "{" { InterfaceMethod } "}" ;
InterfaceMethod = Identifier "(" [ ParamTypeList ] ")" [ ReturnType ] ;

ClassDecl       = "class" Identifier [ "<" Expression ] "{" { ClassMethod } "}" ;
ClassMethod     = { Decorator } Identifier "(" [ ParamList ] ")" Block ;

DecoratedDecl   = Decorator { Decorator } ( FnDecl | ClassDecl ) ;
Decorator       = "@" Expression ;

ParamTypeList   = ParamType { "," ParamType } ;
ParamType       = Identifier TypeExpr ;
```

### 语句

```ebnf
Statement       = Block
                | IfStmt
                | WhileStmt
                | ForStmt
                | MatchStmt
                | TryCatchStmt
                | GoStmt
                | SelectStmt
                | ReturnStmt
                | ThrowStmt
                | BreakStmt
                | ContinueStmt
                | VarDecl
                | ExprStmt ;

Block           = "{" { Statement } "}" ;

ExprStmt        = Expression ;
ReturnStmt      = "return" [ Expression ] ;
ThrowStmt       = "throw" Expression ;
BreakStmt       = "break" ;
ContinueStmt    = "continue" ;
GoStmt          = "go" Expression ;
```

### if / while / for

```ebnf
IfStmt          = "if" Expression Block [ "else" ( Block | IfStmt ) ] ;

WhileStmt       = "while" Expression Block ;

ForStmt         = "for" (
                    [ ForInBinding ] "in" Expression
                  | Expression
                  )? Block ;

ForInBinding    = BindingName [ "," BindingName ] ;
BindingName     = Identifier | "_" ;
```

### 迭代器协议与 `for-in` 语义

更多使用示例见 `docs/iterators.md`。

Icoo 当前的 `for-in` 基于统一迭代器协议，而不是为数组单独做语法特判。

任何可迭代值都需要暴露：

- `iter()`：返回一个迭代器对象
- `next()`：返回一步迭代结果对象

内建迭代器的 `next()` 返回值统一为：

```icoo
{
  key: <当前键或索引，结束时为 null>,
  value: <当前值，结束时为 null>,
  item: <单绑定 for-in 使用的值，结束时为 null>,
  done: <是否结束>
}
```

`for-in` 支持两种绑定形式：

```icoo
for item in iterable {
  // 绑定 step.item
}

for key, value in iterable {
  // 分别绑定 step.key 和 step.value
}
```

其中 `_` 表示忽略该绑定：

```icoo
for _, value in arr {
  println(value)
}
```

当前内建可迭代对象的行为如下：

- `array`
  - `key` 为从 `0` 开始的索引
  - `value` 为数组元素
  - `item == value`
- `string`
  - 按 Unicode rune 逐个迭代
  - `key` 为从 `0` 开始的 rune 索引
  - `value` 为单个字符组成的字符串
  - `item == value`
- `object`
  - 默认按字段名排序后的稳定顺序迭代
  - `key` 为字段名
  - `value` 为字段值
  - `item` 为 `{ key, value }`
- `module`
  - 默认按导出名排序后的稳定顺序迭代
  - `key` 为导出名
  - `value` 为导出值
  - `item` 为 `{ key, value }`
- `iterator`
  - 迭代器本身也可再次参与 `for-in`
  - `iter()` 直接返回自身

对象还支持覆盖默认迭代行为。如果对象自身存在 `iter` 字段，优先使用该字段：

```icoo
let obj = {
  label: "fallback",
  iter: fn() {
    return ["x", "y"].iter()
  }
}

for item in obj {
  println(item)
}
```

示例：

```icoo
let arr = [4, 5, 6]
for idx, value in arr {
  println(idx)
  println(value)
}

let text = "ab"
for idx, ch in text {
  println(idx)
  println(ch)
}

let obj = {b: 2, a: 1}
for key, value in obj {
  println(key)
  println(value)
}

let iter = [7, 8].iter()
for idx, value in iter {
  println(idx)
  println(value)
}
```

### match

```ebnf
MatchStmt       = "match" Expression "{" { MatchCase } "}" ;

MatchCase       = Pattern [ Guard ] "=>" MatchBody ;
Guard           = "if" Expression ;

MatchBody       = Block | Expression ;

Pattern         = "_"
                | Literal
                | Identifier
                | ArrayPattern
                | ObjectPattern ;

ArrayPattern    = "[" [ Pattern { "," Pattern } ] "]" ;
ObjectPattern   = "{" [ ObjectPatternField { "," ObjectPatternField } ] "}" ;
ObjectPatternField = Identifier [ ":" Pattern ] ;
```

### try / catch / finally

```ebnf
TryCatchStmt    = "try" Block [ "catch" Identifier Block ] [ "finally" Block ] ;
```

### select

```ebnf
SelectStmt      = "select" "{" { SelectCase } "}" ;

SelectCase      = RecvCase | SendCase | ElseCase ;

RecvCase        = "recv" Expression "as" Identifier [ "," Identifier ] Block ;
SendCase        = "send" Expression "," Expression Block ;
ElseCase        = "else" Block ;
```

### 表达式

```ebnf
Expression      = Assignment ;

Assignment      = Ternary [ AssignOp Assignment ] ;
AssignOp        = "=" | "+=" | "-=" | "*=" | "/=" ;

Ternary         = LogicOr [ "?" Expression ":" Ternary ] ;

LogicOr         = LogicAnd { "||" LogicAnd } ;
LogicAnd        = Equality { "&&" Equality } ;
Equality        = Comparison { ( "==" | "!=" ) Comparison } ;
Comparison      = Term { ( ">" | ">=" | "<" | "<=" ) Term } ;
Term            = Factor { ( "+" | "-" ) Factor } ;
Factor          = Unary { ( "*" | "/" | "%" ) Unary } ;

Unary           = ( "!" | "-" ) Unary | Postfix ;

Postfix         = Primary { PostfixOp } ;
PostfixOp       = CallOp | MemberOp | IndexOp | TryOp ;

CallOp          = "(" [ ArgumentList ] ")" ;
MemberOp        = "." Identifier ;
IndexOp         = "[" Expression "]" ;
TryOp           = "?" ;

ArgumentList    = Expression { "," Expression } ;
```

### 字面量

```ebnf
Primary         = Literal
                | Identifier
                | FnExpr
                | ArrayLiteral
                | ObjectLiteral
                | "(" Expression ")" ;

Literal         = IntLit
                | FloatLit
                | StringLit
                | "true"
                | "false"
                | "null" ;

ArrayLiteral    = "[" [ Expression { "," Expression } ] "]" ;

ObjectLiteral   = "{" [ ObjectField { "," ObjectField } ] "}" ;
ObjectField     = Identifier ":" Expression
                | StringLit ":" Expression ;
```

### 类型表达式

```ebnf
TypeExpr        = SimpleType
                | ArrayType
                | ObjectType
                | FuncType
                | ChanType ;

SimpleType      = Identifier ;
ArrayType       = "[" TypeExpr "]" ;
ObjectType      = "{" [ TypeField { "," TypeField } ] "}" ;
TypeField       = Identifier ":" TypeExpr ;
FuncType        = "fn" "(" [ TypeExprList ] ")" [ ":" TypeExpr ] ;
TypeExprList    = TypeExpr { "," TypeExpr } ;
ChanType        = "chan" "[" TypeExpr "]" ;
```

### 语法与实现对齐说明

上面的 EBNF 仍然是“靠近实现的草案”，不是形式化规范。当前仓库里有几处实现细节值得额外说明：

- `?` 同时承担两种角色：
  - 三元表达式：`cond ? thenExpr : elseExpr`
  - 后缀 try 表达式：`expr?`
- `try` 语句当前支持：
  - `try { ... } catch err { ... }`
  - `try { ... } finally { ... }`
  - `try { ... } catch err { ... } finally { ... }`
- `class` 的继承写法是 `class Dog < Animal { ... }`
- 类方法目前不写 `fn` 关键字，直接写方法名
- 装饰器既可修饰函数，也可修饰类；类内部的方法也可单独带装饰器
- `interface` 方法参数在当前实现里记录的是“参数名 + 类型表达式”序列，文档中的 EBNF 只表达结构，不试图精确反映内部 AST 细节

如果要了解最准确的当前行为，应以这些文件为准：

- `internal/parser/parser_decl.go`
- `internal/parser/parser_stmt.go`
- `internal/parser/parser_expr.go`
- `pkg/api/*_test.go`

---

## Golang 实现结构设计

### 推荐目录

```text
internal/
  token/
    token.go
  lexer/
    lexer.go
  ast/
    ast.go
    expr.go
    stmt.go
    decl.go
  parser/
    parser.go
    parser_decl.go
    parser_stmt.go
    parser_expr.go
    precedence.go
  diag/
    diagnostic.go
```

---

## Token 设计

```go
type Type int
```

推荐包含：

- 基础：`Illegal`, `EOF`, `Ident`, `Int`, `Float`, `String`
- 运算符：`= + - * / % += -= *= /= == != < <= > >= && || !`
- 分隔符：`. , : ; => ( ) { } [ ]`
- 关键字：`fn return if else for while match break continue const let import export try catch go select interface type null true false in as recv send _`

位置结构：

```go
type Position struct {
	Offset int
	Line   int
	Column int
}

type Span struct {
	Start Position
	End   Position
}

type Token struct {
	Type   Type
	Lexeme string
	Span   Span
}
```

---

## Lexer 设计

```go
type Lexer struct {
	src    []rune
	pos    int
	line   int
	column int
}
```

对外接口：

```go
func New(src string) *Lexer
func (l *Lexer) NextToken() token.Token
func LexAll(src string) []token.Token
```

内部建议方法：

```go
func (l *Lexer) peek() rune
func (l *Lexer) peekNext() rune
func (l *Lexer) advance() rune
func (l *Lexer) skipWhitespace()
func (l *Lexer) skipComment()
func (l *Lexer) lexNumber() token.Token
func (l *Lexer) lexString() token.Token
func (l *Lexer) lexIdentifierOrKeyword() token.Token
```

首版支持：
- 空白符和换行
- 单行注释 `//`
- 标识符
- 数字
- 字符串
- 操作符
- 分隔符
- 关键字

---

## AST 设计

统一接口：

```go
type Node interface {
	node()
	Span() token.Span
}

type Expr interface {
	Node
	expr()
}

type Stmt interface {
	Node
	stmt()
}

type Decl interface {
	Node
	decl()
}
```

### Program

```go
type Program struct {
	Decls []Decl
	Span_ token.Span
}
```

### Declaration

- `ImportDecl`
- `ExportDecl`
- `VarDecl`
- `FnDecl`
- `TypeDecl`
- `InterfaceDecl`

### Statement

- `BlockStmt`
- `ExprStmt`
- `ReturnStmt`
- `IfStmt`
- `WhileStmt`
- `ForStmt`
- `ForInStmt`
- `BreakStmt`
- `ContinueStmt`
- `TryCatchStmt`
- `GoStmt`
- `SelectStmt`
- `MatchStmt`

### SelectCase

- `SelectRecvCase`
- `SelectSendCase`
- `SelectElseCase`

### Pattern

- `WildcardPattern`
- `LiteralPattern`
- `IdentPattern`
- `ArrayPattern`
- `ObjectPattern`

### Expression

- `IdentExpr`
- `IntLiteral`
- `FloatLiteral`
- `StringLiteral`
- `BoolLiteral`
- `NullLiteral`
- `UnaryExpr`
- `BinaryExpr`
- `AssignExpr`
- `CallExpr`
- `MemberExpr`
- `IndexExpr`
- `ArrayLiteral`
- `ObjectLiteral`
- `FnExpr`

### TypeExpr

- `NamedTypeExpr`
- `ArrayTypeExpr`
- `ObjectTypeExpr`
- `FuncTypeExpr`
- `ChanTypeExpr`

---

## Parser 设计

```go
type Parser struct {
	tokens []token.Token
	pos    int
	errors []error
}
```

对外接口：

```go
func New(tokens []token.Token) *Parser
func (p *Parser) ParseProgram() *ast.Program
func (p *Parser) Errors() []error
```

基础游标方法：

```go
func (p *Parser) current() token.Token
func (p *Parser) previous() token.Token
func (p *Parser) peek(offset int) token.Token
func (p *Parser) advance() token.Token
func (p *Parser) match(types ...token.Type) bool
func (p *Parser) check(tt token.Type) bool
func (p *Parser) expect(tt token.Type, msg string) token.Token
func (p *Parser) atEnd() bool
func (p *Parser) synchronize()
```

建议拆分：

- `parser_decl.go`
- `parser_stmt.go`
- `parser_expr.go`
- `precedence.go`

---

## 表达式优先级

```go
type Precedence int

const (
	PrecLowest Precedence = iota
	PrecAssign
	PrecOr
	PrecAnd
	PrecEquality
	PrecCompare
	PrecTerm
	PrecFactor
	PrecUnary
	PrecPostfix
)
```

并提供：

```go
func precedenceOf(tt token.Type) Precedence
```

表达式采用 Pratt Parser：

```go
func (p *Parser) parseExpression(precedence Precedence) ast.Expr
```

适合处理：
- `a + b * c`
- `foo(1, 2).bar[0]`
- `a = b + c`

---

## Diagnostic 设计

```go
type Severity int

const (
	Error Severity = iota
	Warning
)

type Diagnostic struct {
	Severity Severity
	Message  string
	Span     token.Span
}
```

用于统一：
- lexer 错误
- parser 错误
- sema 错误

---

## AST 示例

源码：

```icoo
fn add(a, b) {
  return a + b
}
```

对应 AST：

```text
Program
  FnDecl(name=add)
    Params: a, b
    Body:
      BlockStmt
        ReturnStmt
          BinaryExpr(+)
            IdentExpr(a)
            IdentExpr(b)
```

源码：

```icoo
go worker(ch)
```

对应 AST：

```text
GoStmt
  CallExpr
    IdentExpr(worker)
    IdentExpr(ch)
```

源码：

```icoo
select {
  recv ch as msg {
    print(msg)
  }
  else {
    print("idle")
  }
}
```

对应 AST：

```text
SelectStmt
  SelectRecvCase
    Channel: IdentExpr(ch)
    Value: "msg"
    Body: BlockStmt(...)
  SelectElseCase
    Body: BlockStmt(...)
```

---

## 字节码指令集初稿

推荐采用 **stack-based VM**。每个编译后的函数可表示为：

```go
type Chunk struct {
	Code      []byte
	Constants []Value
	Lines     []int
}
```

运行时基础组件建议包含：
- 操作数栈
- 调用帧栈
- 常量池
- 全局变量表
- 模块缓存
- 原生函数注册表

### Opcode 分类

建议分为：
1. 常量与字面量
2. 栈操作
3. 变量访问
4. 算术与比较
5. 控制流
6. 函数调用
7. 数组/对象
8. 模块系统
9. 并发
10. 异常处理

### 常量与字面量

```text
OpConstant <u16 constIndex>
OpNull
OpTrue
OpFalse
```

### 栈操作

```text
OpPop
OpDup
```

### 变量访问

局部变量：

```text
OpGetLocal <u16 slot>
OpSetLocal <u16 slot>
```

全局变量：

```text
OpDefineGlobal <u16 nameConstIndex>
OpGetGlobal <u16 nameConstIndex>
OpSetGlobal <u16 nameConstIndex>
```

闭包变量：

```text
OpGetUpvalue <u16 slot>
OpSetUpvalue <u16 slot>
OpCloseUpvalue
```

### 算术与逻辑

```text
OpAdd
OpSub
OpMul
OpDiv
OpMod
OpNegate
OpNot
OpEqual
OpNotEqual
OpGreater
OpGreaterEqual
OpLess
OpLessEqual
```

### 控制流

```text
OpJump <u16 offset>
OpJumpIfFalse <u16 offset>
OpLoop <u16 offset>
```

用于支撑：
- `if / else`
- `while`
- `for`
- `match` 降级后的条件跳转链

### 函数调用

```text
OpCall <u8 argc>
OpClosure <u16 constIndex>
OpReturn
```

### 数组与对象

```text
OpArray <u16 count>
OpObject <u16 pairCount>
OpGetProperty <u16 nameConstIndex>
OpSetProperty <u16 nameConstIndex>
OpGetIndex
OpSetIndex
```

### 模块系统

```text
OpImportModule <u16 pathConstIndex>
OpExport <u16 nameConstIndex>
```

### 并发

由于语言采用对象风格 channel API，首版建议：
- `chan()` 编译成 builtin function 调用
- `ch.send(v)` / `ch.recv()` 编译成方法调用

后续优化时再引入专用 opcode：

```text
OpChanSend
OpChanRecv
OpChanTrySend
OpChanTryRecv
OpChanClose
```

### select

`select` 建议首版编译成 runtime intrinsic 调用，而不是一开始就设计复杂 opcode。

即：
- 编译器先构造 select case 描述
- 调用内建 `__select(cases)`
- 按返回的 case index 跳转执行对应 block

### 异常处理

为 `try/catch` 预留：

```text
OpPushExceptionHandler <start> <end> <handler> <slot>
OpPopExceptionHandler
OpThrow
```

调用帧里可维护异常处理栈：

```go
type ExceptionHandler struct {
	StartIP   int
	EndIP     int
	HandlerIP int
	Slot      int
}
```

### match

MVP 不需要专用 pattern opcode，先将 `match` 编译成条件链。
复杂数组/对象模式匹配可在后续考虑：

```text
OpMatchArray
OpMatchObject
OpBindPattern
```

### 常量池

首版常量池建议支持：
- int
- float
- string
- function prototype

函数原型建议：

```go
type FunctionProto struct {
	Name         string
	Arity        int
	Chunk        *Chunk
	LocalCount   int
	UpvalueCount int
}
```

### 调用帧

```go
type CallFrame struct {
	Closure *Closure
	IP      int
	Base    int
}
```

### VM 基础结构

```go
type VM struct {
	stack    []Value
	frames   []CallFrame
	globals  map[string]Value
	modules  map[string]*Module
	builtins map[string]Value
}
```

### MVP 最小指令集

第一阶段建议优先实现：

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

第二阶段再补：

```text
OpImportModule
OpExport
OpPushExceptionHandler
OpPopExceptionHandler
OpThrow
OpChanSend
OpChanRecv
OpChanTrySend
OpChanTryRecv
OpChanClose
```

### 编译演进顺序建议

1. 表达式 + 变量 + `if/while` + 函数
2. 数组/对象 + 成员访问 + 模块
3. 闭包
4. `match`
5. `try/catch`
6. `go/channel/select`

---

## VM / Value / Closure / Module 核心结构设计

推荐运行时模型：
- 统一 `Value` 接口表示所有运行时值
- 编译产物用 `FunctionProto`
- 运行时可调用对象用 `Closure`
- 每个文件加载为 `Module`
- `VM` 负责栈、调用帧、模块缓存和内建函数

### Value 设计

推荐定义：

```go
type Value interface {
	Kind() ValueKind
	String() string
}
```

```go
type ValueKind uint8

const (
	NullKind ValueKind = iota
	BoolKind
	IntKind
	FloatKind
	StringKind
	ArrayKind
	ObjectKind
	FunctionKind
	NativeFunctionKind
	ClosureKind
	ModuleKind
	IteratorKind
	ChannelKind
	ErrorKind
)
```

基础值类型建议：
- `NullValue`
- `BoolValue`
- `IntValue`
- `FloatValue`
- `StringValue`

复合值类型建议：
- `ArrayValue { Elements []Value }`
- `ObjectValue { Fields map[string]Value }`
- `StringIterator { Runes []rune, Index int }`
- `ArrayIterator { Array *ArrayValue, Index int }`
- `ObjectIterator { Items []Value, Index int }`

其中数组、对象和迭代器建议使用引用语义。

### FunctionProto / Closure

编译后的函数原型：

```go
type FunctionProto struct {
	Name         string
	Arity        int
	Chunk        *Chunk
	LocalCount   int
	UpvalueCount int
}
```

运行时闭包对象：

```go
type Closure struct {
	Proto    *FunctionProto
	Upvalues []*Upvalue
}
```

闭包捕获结构：

```go
type Upvalue struct {
	Location *Value
	Closed   Value
	IsClosed bool
}
```

### NativeFunction

标准库和内建函数建议统一建模：

```go
type NativeFunc func(vm *VM, args []Value) (Value, error)

type NativeFunction struct {
	Name  string
	Arity int
	Fn    NativeFunc
}
```

适合承载：
- `print`
- `println`
- `chan`
- `len`
- `panic`

### Module

建议每个文件加载为一个模块对象：

```go
type Module struct {
	Name    string
	Path    string
	Exports map[string]Value
	Globals map[string]Value
	Init    *Closure
	Done    bool
}
```

字段含义：
- `Exports`：导出符号表
- `Globals`：模块内部全局作用域
- `Init`：模块初始化闭包
- `Done`：是否已完成初始化

### ChannelValue

并发运行时建议：

```go
type ChannelValue struct {
	ch     chan Value
	closed bool
	mu     sync.Mutex
}
```

建议方法：

```go
func NewChannelValue(size int) *ChannelValue
func (c *ChannelValue) Send(v Value) error
func (c *ChannelValue) Recv() (Value, bool)
func (c *ChannelValue) TrySend(v Value) bool
func (c *ChannelValue) TryRecv() (Value, bool)
func (c *ChannelValue) Close() error
```

### ErrorValue

为 `try/catch` 预留单独的语言级错误对象：

```go
type ErrorValue struct {
	Message string
	Data    Value
}
```

Go error 主要用于 VM/编译器/宿主失败；脚本运行时异常建议使用 `ErrorValue`。

### Chunk

```go
type Chunk struct {
	Code      []byte
	Constants []Value
	Lines     []int
}
```

建议方法：

```go
func (c *Chunk) Write(op byte, line int)
func (c *Chunk) WriteShort(v uint16, line int)
func (c *Chunk) AddConstant(v Value) uint16
```

### CallFrame

```go
type CallFrame struct {
	Closure *Closure
	IP      int
	Base    int
}
```

### VM

```go
type VM struct {
	stack        []Value
	frames       []CallFrame
	globals      map[string]Value
	builtins     map[string]Value
	modules      map[string]*Module
	openUpvalues []*Upvalue
}
```

建议基础方法：

```go
func NewVM() *VM
func (vm *VM) Push(v Value)
func (vm *VM) Pop() Value
func (vm *VM) Peek(distance int) Value
func (vm *VM) CallValue(callee Value, argc int) error
func (vm *VM) Run(closure *Closure) (Value, error)
func (vm *VM) DefineBuiltin(name string, v Value)
func (vm *VM) LoadModule(path string) (*Module, error)
```

### Truthiness 与相等性

建议统一封装：

```go
func IsTruthy(v Value) bool
func ValueEqual(a, b Value) bool
```

推荐 truthiness 规则：
- `null` -> false
- `false` -> false
- 其他 -> true

推荐相等性规则：
- `null == null`
- bool/string 按值比较
- array/object/function 默认按引用比较
- `int` 和 `float` 可做数值兼容比较

### 多返回值

为支持：

```icoo
let v, ok = ch.recv()
```

建议在 VM 调用边界引入：

```go
type MultiValue struct {
	Values []Value
}
```

它不必暴露给用户，只用于运行时展开多返回值。

### 编译期模块与运行期模块

建议区分：

```go
type CompiledModule struct {
	Path    string
	Proto   *FunctionProto
	Exports []string
}
```

运行期再加载成：

```go
type Module struct {
	Name    string
	Path    string
	Exports map[string]Value
	Globals map[string]Value
	Init    *Closure
	Done    bool
}
```

### MVP 运行时对象顺序建议

第一阶段优先：
- `NullValue`
- `BoolValue`
- `IntValue`
- `FloatValue`
- `StringValue`
- `ArrayValue`
- `ObjectValue`
- `Closure`
- `NativeFunction`
- `Module`

第二阶段再补：
- `ChannelValue`
- `ErrorValue`
- 完整 `Upvalue`

---

## Compiler 设计：AST -> Bytecode 与作用域管理

建议编译器分为两层：
- 顶层模块编译器：负责编译文件、`import/export`、模块初始化函数
- 函数编译器：负责编译函数体、作用域、局部变量、闭包捕获

### 推荐目录

```text
internal/
  compiler/
    compiler.go
    compile_decl.go
    compile_stmt.go
    compile_expr.go
    scope.go
    symbol.go
    module.go
```

### Compiler / FuncCompiler

```go
type Compiler struct {
	vmBuiltins map[string]struct{}
	modulePath string
	errors     []error
	current    *FuncCompiler
}
```

```go
type FuncCompiler struct {
	parent     *FuncCompiler
	proto      *FunctionProto
	chunk      *Chunk
	locals     []Local
	scopeDepth int
	upvalues   []UpvalueRef
	loopStack  []LoopContext
}
```

### Local

```go
type Local struct {
	Name       string
	Depth      int
	Slot       int
	IsCaptured bool
	IsConst    bool
}
```

### UpvalueRef

```go
type UpvalueRef struct {
	Index   int
	IsLocal bool
}
```

### LoopContext

```go
type LoopContext struct {
	BreakJumps     []int
	ContinueTarget int
	ScopeDepth     int
}
```

### CompiledModule

```go
type CompiledModule struct {
	Path    string
	Proto   *FunctionProto
	Exports []string
}
```

整个模块建议编译成一个隐式初始化函数，例如 `__module_init__`。

### 作用域管理

```go
func (fc *FuncCompiler) beginScope()
func (fc *FuncCompiler) endScope()
```

规则：
- `scopeDepth == 0` 表示函数顶层
- 更深层表示 block 作用域
- 退出作用域时：
  - 普通 local 发 `OpPop`
  - captured local 发 `OpCloseUpvalue`

### 变量解析顺序

读取变量时按顺序查找：
1. 当前函数 local
2. 当前函数 upvalue
3. 模块/全局变量
4. builtin

建议定义：

```go
type VarRefKind int

const (
	VarLocal VarRefKind = iota
	VarUpvalue
	VarGlobal
	VarBuiltin
)

type VarRef struct {
	Kind  VarRefKind
	Index int
	Name  string
}
```

```go
func (fc *FuncCompiler) resolve(name string) (VarRef, bool)
```

### 闭包捕获流程

内层函数访问外层变量时：
- 当前函数找不到 local
- 递归 parent 查找
- 若命中 parent local，则标记 `IsCaptured=true`
- 当前函数登记对应 upvalue

### 声明编译

建议统一入口：

```go
func (c *Compiler) compileDecl(d ast.Decl)
```

分派到：
- `compileImportDecl`
- `compileExportDecl`
- `compileVarDecl`
- `compileFnDecl`
- `compileTypeDecl`
- `compileInterfaceDecl`

其中 `type/interface` 在 MVP 可先只做语义记录。

### VarDecl 编译

模块级变量可编译成：

```text
OpConstant 1
OpDefineGlobal "a"
```

局部变量流程：
1. 编译 RHS
2. 分配 local slot
3. 发 `OpSetLocal`

### FnDecl / FnExpr 编译

流程：
1. 创建子 `FuncCompiler`
2. 参数注册为 local
3. 编译函数体
4. 自动补 `return null`
5. 生成 `FunctionProto`
6. 外层发 `OpClosure`
7. 若有名称则绑定名字

### BlockStmt 编译

```go
func (fc *FuncCompiler) compileBlockStmt(b *ast.BlockStmt)
```

进入作用域，依次编译语句，退出作用域。

### IfStmt 编译

基本流程：
1. 编译条件
2. 发 `OpJumpIfFalse`
3. 编译 then
4. 发 `OpJump`
5. patch false jump
6. 编译 else
7. patch end jump

### WhileStmt 编译

基本流程：
1. 记录 loopStart
2. 编译条件
3. `OpJumpIfFalse exit`
4. 编译 body
5. `OpLoop loopStart`
6. patch exit

并通过 `LoopContext` 支持 `break/continue`。

### ReturnStmt / ExprStmt

- `return expr` -> 编译 expr 后发 `OpReturn`
- `return` -> `OpNull` + `OpReturn`
- 表达式语句 -> 编译表达式后 `OpPop`

### 赋值表达式

支持：
- `x = 1`
- `obj.name = 1`
- `arr[i] = 1`

对应目标：
- local -> `OpSetLocal`
- upvalue -> `OpSetUpvalue`
- global -> `OpSetGlobal`
- property -> `OpSetProperty`
- index -> `OpSetIndex`

### 二元表达式

AST 已处理优先级，编译时按左右递归后发 opcode，例如：
- `+` -> `OpAdd`
- `-` -> `OpSub`
- `*` -> `OpMul`
- `/` -> `OpDiv`
- `%` -> `OpMod`
- `==` -> `OpEqual`
- `!=` -> `OpNotEqual`
- `>` -> `OpGreater`
- `>=` -> `OpGreaterEqual`
- `<` -> `OpLess`
- `<=` -> `OpLessEqual`

### 短路逻辑

建议支持：
- `&&`
- `||`

实现时建议补一个：

```text
OpJumpIfTrue
```

便于实现 `||` 的短路语义。

### 调用 / 成员 / 下标

- `CallExpr`：callee + args -> `OpCall argc`
- `MemberExpr`：object -> `OpGetProperty`
- `IndexExpr`：object + index -> `OpGetIndex`

### Array / Object 字面量

- 数组：逐个压栈后 `OpArray count`
- 对象：逐个压 key/value 后 `OpObject pairCount`

### Import / Export

`import` 建议编译为模块对象加载并绑定别名：

```text
OpImportModule "std.io"
```

`export` 建议编译期收集导出名，再在模块初始化结束时统一填充导出表。

### Match / TryCatch / Go / Select

建议：
- `match` MVP 降级成条件跳转链
- `try/catch` 第二阶段实现，依赖异常处理器栈
- `go` 第一版可降级为 builtin `__go`
- `ch.send/ch.recv` 第一版仍作为普通成员调用
- `select` 第二阶段实现为 runtime intrinsic

### 编译器辅助方法

```go
func (fc *FuncCompiler) emit(op byte)
func (fc *FuncCompiler) emitByte(b byte)
func (fc *FuncCompiler) emitShort(v uint16)
func (fc *FuncCompiler) emitConstant(v Value)
func (fc *FuncCompiler) emitJump(op byte) int
func (fc *FuncCompiler) patchJump(pos int)
func (fc *FuncCompiler) emitLoop(loopStart int)
func (fc *FuncCompiler) resolveLocal(name string) int
func (fc *FuncCompiler) resolveUpvalue(name string) int
func (fc *FuncCompiler) addUpvalue(index int, isLocal bool) int
```

### 第一阶段边界

建议第一阶段先只完成：
- 常量
- 局部/全局变量
- `const/let`
- 函数声明与匿名函数
- `return`
- `if/else`
- `while`
- 表达式
- 数组/对象
- 调用
- 成员/下标访问

暂缓：
- `match`
- `try/catch`
- `go`
- `select`
- 完整 import/export
- 完整闭包优化

---

## 标准库、内建函数与宿主 Go 绑定策略

建议把运行时能力分成三层：

1. **Builtins**：全局直接可用，如 `print`、`println`、`len`、`chan`
2. **Stdlib Modules**：通过 `import` 使用，如 `std.io`、`std.time`、`std.json`
3. **Host Binding**：由 Go 宿主程序动态注入函数、值和模块

### Builtins 设计原则

建议 builtin 保持精简，首版保留：
- `print(...args)`
- `println(...args)`
- `len(x)`
- `typeOf(x)`
- `chan(size?)`
- `panic(message)`

统一建模：

```go
type NativeFunc func(vm *VM, args []Value) (Value, error)

type NativeFunction struct {
	Name  string
	Arity int // -1 表示变参
	Fn    NativeFunc
}
```

### Builtin 细节建议

#### `print(...args)`
- 直接输出
- 不换行
- 建议直接拼接参数的字符串表示

#### `println(...args)`
- 输出后换行
- 建议参数之间以空格连接

#### `len(x)`
建议支持：
- `string`
- `array`
- `object`

返回对应长度或字段数。

#### `typeOf(x)`
返回字符串类型名，例如：
- `int`
- `float`
- `string`
- `array`
- `object`
- `null`

#### `chan(size?)`
- 0 参数：无缓冲 channel
- 1 参数：缓冲大小
- 参数必须为 int

#### `panic(message)`
MVP 先作为 runtime error 返回；后续再接入 `try/catch` 的可捕获异常体系。

### Builtin 注册

建议在 `NewVM()` 中统一注册：

```go
func NewVM() *VM {
	vm := &VM{
		globals:  map[string]Value{},
		builtins: map[string]Value{},
		modules:  map[string]*Module{},
	}

	vm.registerBuiltins()
	return vm
}
```

```go
func (vm *VM) registerBuiltins() {
	vm.DefineBuiltin("print", newPrintBuiltin())
	vm.DefineBuiltin("println", newPrintlnBuiltin())
	vm.DefineBuiltin("len", newLenBuiltin())
	vm.DefineBuiltin("typeOf", newTypeOfBuiltin())
	vm.DefineBuiltin("chan", newChanBuiltin())
	vm.DefineBuiltin("panic", newPanicBuiltin())
}
```

### 标准库模块建议

建议保留这些首版模块：
- `std.io`
- `std.time`
- `std.json`
- `std.fs`
- `std.math`

模块路径建议使用点路径风格，例如：

```icoo
import std.io as io
import std.time as time
```

### `std.io`
可导出：
- `print`
- `println`

即使全局已有 builtin，也允许模块方式使用。

### `std.time`
建议首版只包含：
- `now()`
- `sleep(ms)`

建议：
- `now()` 返回毫秒时间戳 `int`
- `sleep(ms)` 使用毫秒整数

### `std.json`
建议首版包含：
- `encode(value)`
- `decode(text)`

`decode` 返回 Icoo 运行时值：
- object -> `ObjectValue`
- array -> `ArrayValue`
- string -> `StringValue`
- number -> `IntValue` / `FloatValue`
- bool -> `BoolValue`
- null -> `NullValue`

### `std.fs`
建议首版只做：
- `readFile(path)`
- `writeFile(path, content)`
- `exists(path)`

### `std.math`
建议首版只做常见函数：
- `abs(x)`
- `max(a, b)`
- `min(a, b)`
- `floor(x)`
- `ceil(x)`

### 标准库实现方式

首版推荐使用 **Go 原生构造模块**，而不是先用 `.ic` 源码实现标准库。

例如：

```go
func loadStdMathModule(vm *VM) (*Module, error) {
	return &Module{
		Name: "std.math",
		Path: "std.math",
		Exports: map[string]Value{
			"abs": &NativeFunction{Name: "abs", Arity: 1, Fn: mathAbs},
			"max": &NativeFunction{Name: "max", Arity: 2, Fn: mathMax},
			"min": &NativeFunction{Name: "min", Arity: 2, Fn: mathMin},
		},
		Done: true,
	}, nil
}
```

### 标准库注册表

建议 VM 增加标准库注册表：

```go
type ModuleLoader func(vm *VM) (*Module, error)
```

```go
type VM struct {
	...
	stdlib map[string]ModuleLoader
}
```

注册方式：

```go
func (vm *VM) registerStdlib() {
	vm.stdlib["std.io"] = loadStdIOModule
	vm.stdlib["std.time"] = loadStdTimeModule
	vm.stdlib["std.json"] = loadStdJSONModule
	vm.stdlib["std.fs"] = loadStdFSModule
	vm.stdlib["std.math"] = loadStdMathModule
}
```

`LoadModule(path)` 时建议顺序：
1. 查模块缓存
2. 查标准库注册表
3. 查文件模块

### 宿主 Go 绑定策略

当前代码中的 `pkg/api` 已提供一个面向宿主程序与 CLI 的轻量运行时门面，核心对象如下：

```go
type Runtime struct {
	vm               *vm.VM
	modules          map[string]*runtime.Module
	bundledSources   map[string]string
	projectRoot      string
	projectRootAlias string
}
```

当前已经暴露的主要入口包括：

```go
func NewRuntime() *Runtime
func (r *Runtime) VM() *vm.VM
func (r *Runtime) SetProjectRoot(root string, alias string)
func (r *Runtime) SetBundledSources(sources map[string]string)

func (r *Runtime) CheckSource(src string) []error
func (r *Runtime) CheckFile(path string) []error
func (r *Runtime) RunSource(src string) (runtime.Value, error)
func (r *Runtime) RunFile(path string) (runtime.Value, error)
func (r *Runtime) InvokeGlobal(name string) (runtime.Value, error)
func (r *Runtime) RunReplLine(line string) (runtime.Value, error)

func LoadBundle(data []byte) (*BundleArchive, error)
func LoadBundleFile(path string) (*BundleArchive, error)
func (r *Runtime) CheckBundleFile(path string) []error
func (r *Runtime) CheckBundleArchive(archive *BundleArchive) []error
func (r *Runtime) RunBundleFile(path string) (runtime.Value, error)
func (r *Runtime) RunBundleArchive(path string, archive *BundleArchive) (runtime.Value, error)
```

也就是说，当前 `pkg/api` 更像一个**运行、检查、bundle 执行门面**，而不是一个完整的宿主嵌入 SDK。像 `DefineFunc`、`DefineValue`、`DefineModule` 这类直接注入接口，目前并没有作为 `pkg/api` 的公开方法提供。

如果宿主方需要更底层控制，可以通过 `Runtime.VM()` 取到底层 `*vm.VM` 继续扩展。

### Go 值与 Icoo 值转换

当前实现已经有清晰的运行时值模型，核心接口位于 `internal/runtime/value.go`：

```go
type Value interface {
	Kind() ValueKind
	String() string
}
```

当前值种类包括：

- `null`
- `bool`
- `int`
- `float`
- `string`
- `array`
- `object`
- `native function`
- `closure`
- `module`
- `channel`
- `error`
- `iterator`
- `interface`
- `class`
- `bound method`
- `method proxy`
- `method definition`

对宿主开发者来说，最重要的边界是：`pkg/api` 返回的是 `runtime.Value`，而不是 Go 原生类型。因此在宿主集成时，通常需要：

- 根据 `Kind()` 判断值类型
- 再断言到具体值结构，例如 `runtime.StringValue`、`*runtime.ArrayValue`、`*runtime.ObjectValue`

目前仓库里还没有对外导出的通用 `AsString` / `AsInt` 这类 helper API，因此这部分仍以底层值类型断言为主。

### Host Object 建议

从当前实现看，Icoo 的宿主集成策略仍然偏保守：

- 内建函数通过 `runtime.NativeFunction` 暴露
- 标准库模块通过 `runtime.Module` 暴露
- 运行时值通过显式的 `runtime.Value` 体系传递

这意味着当前更适合：

- 暴露函数
- 暴露模块
- 传递基础值、数组、对象

而不适合直接把任意 Go struct 通过反射无约束暴露到脚本层。

### 首版实施顺序建议

如果后续继续扩展宿主嵌入能力，更合适的增量顺序通常是：

1. 补齐 `pkg/api` 的嵌入式 API 文档
2. 为常用 `runtime.Value` 提供安全的 helper 转换函数
3. 在不引入反射泛滥的前提下补最小注入接口
4. 最后再考虑更完整的宿主对象桥接
