# Icoo AI 代码生成参考

本文档面向会生成、解释或审查 Icoo 代码的 AI 系统。

目标不是介绍项目历史，而是提供一份按当前实现对齐的语言与原生 API 速查表，帮助 AI：

- 生成可运行的 Icoo 代码
- 选择正确语法
- 正确调用 builtin 与标准库
- 避免生成编译器内部辅助 API

本文档还有一个明确边界：

- 当前不要假定 Icoo 语言内置 AI / Agent 能力
- 当前不要生成不存在的 `std.ai.*`、`std.agent.*`、`agent(...)` 这类假想 API
- AI / Agent 相关能力当前应视为应用层工程能力，而不是语言内置能力或标准库能力
- `examples/icooa` 仅应视为 legacy example，不应再作为未来 Agent 实现基线

相关文档：

- 语言总览：`docs/language-design.md`
- Runtime API：`docs/api.md`
- 当前状态：`docs/mvp-status.md`
- 架构分析：`docs/architecture-analysis-report.md`

---

## 1. AI 生成代码时的基本判断

### 1.0 先判断“这是语言能力问题，还是应用层 Agent 问题”

生成代码前，AI 应先做这个判断：

- 如果需求是语法、运行时、标准库、模块系统、并发、Web、数据库、文件 I/O，则按本文档生成 Icoo 代码
- 如果需求是会话管理、工具调用、模型请求、审批策略、patch 应用、agent loop，则应按应用工程生成，不要伪造成语言内置能力

换句话说：

- 可以生成 `apps/agent/...` 这类独立应用代码
- 不要生成 `std.ai.chat(...)`、`std.agent.run(...)`、`builtin.agentCall(...)` 这类当前并不存在的 API

### 1.1 当前语言已实现的主要能力

Icoo 当前已经实现：

- `let` / `const`
- `fn`
- `if / else`
- `while`
- `for`
- `for-in`
- `break` / `continue`
- `return`
- `match`
- 匿名函数与闭包
- `throw`
- `try / catch / finally`
- 后缀 try 表达式 `expr?`
- `type` / `interface`
- `class` / `this` / `super`
- 函数、类、方法装饰器
- `chan` / `go` / `select`
- 文件模块与标准库模块导入
- 原生命令行参数 `argv()`
- 原生命令行框架 `std.sys.cli`

### 1.2 AI 应优先遵守的规则

1. 变量声明优先用 `let` 或 `const`，不要生成 `var`
2. 类方法定义不写 `fn`
3. 继承写法是 `class Dog <- Animal`，不是 `extends`，也不是旧写法 `<`
4. `for-in` 可用于 `array / string / object / module / iterator`
5. 需要错误传播时，优先使用 `expr?` 或 `try/catch/finally`
6. 需要并发通道时，优先生成 `chan()` 与对象方法 `send/recv/...`
7. 不要直接生成编译器内部 builtin，例如 `__select`、`__buildClass`
8. 不要假设存在内置 AI / Agent 标准库
9. 需要 Agent 时，优先生成独立 app / CLI 工程，而不是往 `std.*` 中虚构模块

---

## 2. 语法速查

### 2.1 变量与常量

```icoo
let count = 1
const name = "icoo"
```

### 2.2 函数

```icoo
fn add(a, b) {
  return a + b
}

let mul = fn(a, b) {
  return a * b
}
```

### 2.3 条件与循环

```icoo
if score > 90 {
  println("A")
} else {
  println("B")
}

while n > 0 {
  n = n - 1
}

for let i = 0; i < 10; i = i + 1 {
  println(i)
}
```

### 2.4 for-in

单绑定：

```icoo
for item in items {
  println(item)
}
```

双绑定：

```icoo
for key, value in obj {
  println(key, value)
}
```

忽略绑定：

```icoo
for _, value in obj {
  println(value)
}
```

### 2.5 表达式

支持：

- 一元：`!x`、`-x`
- 二元：`+ - * / % == != < <= > >= && ||`
- 赋值：`a = b`
- 三元：`cond ? a : b`
- 成员访问：`obj.name`
- 下标访问：`arr[i]`
- 调用：`fn(...)`

示例：

```icoo
let title = ok ? "ready" : "waiting"
let name = user.profile.name
let first = list[0]
```

### 2.6 数组与对象

```icoo
let arr = [1, 2, 3]
let obj = {
  name: "icoo",
  version: 1,
}
```

### 2.7 模块导入导出

```icoo
import std.io.console as io
import std.io.fs as fs
import app/lib/util.ic as util

export let version = "1.0.0"

export fn main() {
  io.println(version)
}
```

说明：

- 标准库使用 `std.*`
- 文件模块使用路径导入
- 项目根别名导入是否可用，取决于项目配置的 `root_alias`

### 2.8 错误与异常

创建错误值：

```icoo
return error("invalid port")
```

抛出异常：

```icoo
throw "boom"
```

try/catch/finally：

```icoo
try {
  dangerous()
} catch err {
  println(err)
} finally {
  println("done")
}
```

后缀 try 表达式：

```icoo
fn parsePort(text) {
  if text == "3000" {
    return 3000
  }
  return error("invalid port")
}

fn loadPort(text) {
  let port = parsePort(text)?
  return port + 1
}
```

`expr?` 语义：

- 表达式结果不是 `error` 时，返回原值
- 表达式结果是 `error` 时，提前向外传播
- 若外层有 `finally`，会先执行 `finally`

### 2.9 类型与接口

类型别名：

```icoo
type UserID = int
```

接口：

```icoo
interface Greeter {
  greet(name string) string
}
```

运行时检查：

```icoo
let ok = satisfies(service, Greeter)
```

### 2.10 类与继承

```icoo
class Animal {
  init(name) {
    this.name = name
  }

  speak() {
    return this.name + " makes a sound"
  }
}

class Dog <- Animal {
  init(name, breed) {
    super.init(name)
    this.breed = breed
  }

  speak() {
    return super.speak() + " and barks"
  }
}
```

关键规则：

- 方法定义不写 `fn`
- 构造入口通常是 `init(...)`
- 通过 `this.xxx` 访问实例字段
- 通过 `super.xxx(...)` 调父类实现

### 2.11 装饰器

函数装饰器：

```icoo
@prefix("hello ")
fn greet(name) {
  return name
}
```

类装饰器：

```icoo
@mark
class Box {
  init(value) {
    this.value = value
  }
}
```

方法装饰器：

```icoo
class Greeter {
  @excited
  hello() {
    return "welcome"
  }
}
```

### 2.12 并发与 channel

```icoo
let ch = chan()
let buffered = chan(8)

ch.send(1)
let value, ok = ch.recv()
let ok1 = ch.trySend(2)
let value2, ok2 = ch.tryRecv()
ch.close()
```

启动并发任务：

```icoo
go worker(ch)
```

`select`：

```icoo
select {
  recv ch1 as msg {
    println(msg)
  }

  send ch2, 1 {
    println("sent")
  }

  else {
    println("idle")
  }
}
```

---

## 3. builtin 速查

以下 builtin 是用户可直接调用的公开能力。

### 3.1 `print(...args)`

输出参数到标准输出，不自动换行。

### 3.2 `println(...args)`

输出参数到标准输出，并在末尾换行。

### 3.3 `len(x)`

返回长度，支持：

- `string`
- `array`
- `object`

### 3.4 `typeOf(x)`

返回运行时类型名字符串。

### 3.5 `argv()`

返回脚本级命令行参数数组，不包含可执行文件名。

这项能力与 `icoo run app -- ...` 配套，`--` 后的参数会原样进入 `argv()`。

```icoo
let args = argv()
println(args)
```

语义上：

- `argv()` 适合业务代码解析脚本参数
- `std.sys.os.args()` 返回的是进程级参数视图，适合更底层的系统信息读取

### 3.6 `chan(size?)`

创建 channel：

- 0 参数：无缓冲 channel
- 1 参数：缓冲区大小，必须是整数

### 3.7 `panic(message)`

直接抛出运行时错误。

### 3.8 `error(message[, cause])`

构造错误值，返回的是语言级 `error` 值，不会自动抛出。

### 3.9 `satisfies(obj, iface)`

检查对象是否满足接口要求。

当前实现按接口方法名检查对象上是否存在对应可调用字段。

---

## 4. 不要直接生成的内部 builtin

这些 builtin 由编译器自动插入，AI 不应手写生成：

- `__select`
- `_tryCheck`
- `__buildClass`
- `__methodDef`
- `__methodProxy`
- `__superGet`

结论：

- 用户代码请生成高级语法
- 不要把高级语法展开成这些底层 builtin

---

## 5. 标准库模块速查

在阅读本节前，AI 需要记住：

- 本节列出的 `std.*` 才是当前可依赖的标准库范围
- 如果这里没有 `std.ai`、`std.agent`、`std.llm`，就表示当前不要生成这些模块
- 涉及模型调用、会话编排、工具调度时，应使用 `std.net.http.*`、`std.io.fs`、`std.sys.exec` 等通用能力自行搭建应用层实现

## 5.1 core

### `std.io.console`

常用导出：

- `print`
- `println`
- `copy`
- `openReader`
- `openWriter`
- `readAll`

### `std.time.basic`

常用导出：

- `now()`
- `sleep(ms)`
- `format(ts[, layout[, timezone]])`
- `parse(text[, layout[, timezone]])`
- `parts(ts[, timezone])`
- `add(ts, deltaMs)`
- `diff(left, right)`
- `unix(ts)`
- `fromUnix(seconds)`

### `std.math.basic`

常用导出：

- `abs`
- `max`
- `min`
- `floor`
- `ceil`
- `parseInt`

## 5.2 data / format

### `std.data.json`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.data.yaml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.data.toml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.data.xml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.data.csv`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

## 5.3 system

### `std.io.fs`

常用导出：

- `readFile`
- `writeFile`
- `exists`
- `mkdir`
- `remove`
- `readDir`
- `stat`
- `rename`
- `copyFile`
- `join`
- `base`
- `dir`

### `std.sys.exec`

常用导出：

- `run(cmd[, argv])`
- `run({ command, args?, cwd?, shell? })`

### `std.sys.os`

常用导出：

- `args`
- `cwd`
- `tempDir`
- `getEnv`
- `setEnv`
- `mkdirAll`
- `remove`
- `removeAll`

### `std.sys.host`

常用导出：

- `arch`
- `goos`
- `hostname`
- `numCPU`
- `pid`
- `goroutines`
- `memory`
- `runtime`
- `gc`

### `std.sys.cli`

这是最小原生命令行框架，适合应用入口参数解析，不需要引入新语法。

模块入口：

- `create(options)`

`options` 常用字段：

- `name`
- `description`

返回 app 对象后可继续定义：

- `flagString(options)`
- `flagBool(options)`
- `flagInt(options)`
- `command(options[, handler])`
- `action(handler)`
- `help()`
- `run()`

flag `options` 常用字段：

- `name`
- `aliases`
- `short`
- `description`
- `default`
- `required`

`run()` 返回上下文对象，常用字段：

- `command`
- `flags`
- `args`
- `raw`
- `unknown`
- `help`
- `helpText`

示例：

```icoo
import std.io.console as io
import std.sys.cli as cli

let app = cli.create({
  name: "demo",
  description: "demo cli"
})

app.flagBool({
  name: "verbose",
  short: "v",
  description: "Enable verbose output"
})

app.flagString({
  name: "workspace",
  aliases: ["root"],
  required: true
})

let greet = app.command({
  name: "greet",
  description: "Print greeting"
})

greet.flagString({
  name: "name",
  short: "n",
  default: "world"
})

greet.action(fn(ctx) {
  io.println("hello", ctx.flags.name)
  if len(ctx.unknown) > 0 {
    io.println("unknown:", ctx.unknown)
  }
  if ctx.flags.verbose {
    io.println("args:", ctx.args)
  }
})

let result = app.run()
if result.help {
  io.println(result.helpText)
}
```

当前最小能力边界：

- 支持全局 flag
- 支持子命令
- 支持 `bool / string / int` 三种 flag
- 支持帮助文本生成
- 支持 `aliases` 长别名
- 支持 `required` 必填校验
- 支持 unknown args passthrough 到 `ctx.unknown`
- 支持 `--` 后透传为剩余位置参数

当前还没有内置：

- 自动短别名推导
- 更细粒度的子命令级 passthrough 策略
- 自动子命令嵌套树

## 5.4 database

### `std.db.sql`

常用导出：

- `open`
- `sqlite`
- `postgres`
- `mysql`

返回 db 对象方法：

- `close()`
- `ping()`
- `exec(sql, ...args)`
- `query(sql, ...args)`
- `queryOne(sql, ...args)`
- `table(name)`

### `std.db.redis`

常用导出：

- `open(url)`
- `connect(options)`

## 5.5 crypto / data

### `std.crypto.hash`

常用导出：

- `sha256`
- `sha512`
- `hmacSHA256`
- `hmacSHA512`
- `base64Encode`
- `base64Decode`
- `hexEncode`
- `hexDecode`
- `randomBytes`
- `aesGCMEncrypt`
- `aesGCMDecrypt`

### `std.crypto.uuid`

常用导出：

- `v4`
- `isValid`

### `std.data.compress`

常用导出：

- `gzipCompress`
- `gzipDecompress`
- `zlibCompress`
- `zlibDecompress`

## 5.6 net

### `std.net.http.client`

常用导出：

- `get`
- `getJSON`
- `post`
- `put`
- `delete`
- `request`
- `requestJSON`
- `download`

### `std.net.http.server`

常用导出：

- `listen`
- `forward`

### `std.net.websocket.client`

导出：

- `connect`

### `std.net.websocket.server`

导出：

- `listen`

### `std.net.sse.client`

导出：

- `connect`

### `std.net.sse.server`

导出：

- `listen`

### `std.net.socket.client`

导出：

- `connectTCP`
- `dialUDP`

### `std.net.socket.server`

导出：

- `listenTCP`
- `listenUDP`

## 5.7 express

### `std.web.express`

顶层导出：

- `create` / `new`
- `json`
- `text`
- `redirect`
- `next`

app 对象常用方法：

- `use`
- `all`
- `get`
- `post`
- `put`
- `delete`
- `patch`
- `head`
- `options`
- `listen`

---

## 6. AI 生成代码时的推荐模式

### 6.1 返回错误优先模式

更推荐：

```icoo
fn loadConfig(path) {
  if !fs.exists(path) {
    return error("config not found")
  }
  return json.decode(fs.readFile(path))
}

fn main() {
  let cfg = loadConfig("app.json")?
  println(cfg)
}
```

### 6.2 面向模块别名生成代码

更推荐：

```icoo
import std.data.json as json
import std.io.fs as fs
```

### 6.3 channel 使用对象方法

更推荐：

```icoo
let ch = chan()
ch.send(1)
let v, ok = ch.recv()
```

### 6.4 类方法不要写 `fn`

正确：

```icoo
class User {
  init(name) {
    this.name = name
  }

  greet() {
    return this.name
  }
}
```

### 6.5 命令行参数优先用 `argv()` 或 `std.sys.cli`

简单脚本参数读取：

```icoo
let args = argv()
```

复杂 CLI 应用：

```icoo
import std.sys.cli as cli
let app = cli.create({name: "tool"})
```

不要让业务代码直接依赖宿主进程的 `os.Args` 语义。

### 6.6 Agent 能力放在应用层

更推荐：

```icoo
import std.io.fs as fs
import std.data.json as json
import std.net.http.client as http

fn callModel(baseUrl, apiKey, payload) {
  return http.requestJSON({
    url: baseUrl + "/chat/completions",
    method: "POST",
    headers: {
      Authorization: "Bearer " + apiKey
    },
    json: payload
  })
}
```

而不是生成不存在的接口：

```icoo
import std.ai as ai

let result = ai.chat({
  model: "gpt-4.1-mini",
  prompt: "..."
})
```

---

## 7. AI 常见易错点

1. 把类继承写成 `extends`
2. 在类体里给方法加 `fn`
3. 继续使用旧继承语法 `<`
4. 直接生成内部 builtin，如 `__superGet(...)`
5. 把 channel API 写成自由函数而不是对象方法
6. 忘记 `expr?` 只对返回 `error` 值的函数有意义
7. 把 `error(...)` 当成自动抛异常；实际上它是构造错误值
8. 误以为 `satisfies` 是编译期关键字；它是 builtin
9. 为简单场景生成过于复杂的 `try/catch/finally`
10. 虚构 `std.ai`、`std.agent`、`std.llm` 或其他不存在的 AI 标准库
11. 把应用层 Agent 编排误写成语言内置能力
12. 继续把 `examples/icooa` 当成未来 Agent 主线结构
13. 复杂 CLI 仍然手写字符串遍历，而不使用 `std.sys.cli`

---

## 8. 推荐给 AI 的生成策略

### 8.1 简单脚本

优先使用：

- `let` / `const`
- 顶层函数
- `if` / `for` / `while`
- `argv()`
- `std.io.console` / `std.io.fs` / `std.data.json`

### 8.2 可恢复错误

优先使用：

- `error(...)`
- `expr?`
- 必要时 `try/catch/finally`

### 8.3 Web / 网络示例

优先从这些模块选：

- `std.net.http.client`
- `std.net.http.server`
- `std.web.express`
- `std.net.websocket.*`
- `std.net.sse.*`
- `std.net.socket.*`

### 8.4 需要接口约束时

优先写：

- `interface Xxx { ... }`
- `satisfies(obj, Xxx)`

### 8.5 需要面向对象示例时

优先写：

- `class`
- `init(...)`
- `this`
- `super`

### 8.6 需要 CLI / Agent / LLM / Tooling 时

优先写：

- 独立应用目录，例如 `apps/agent`
- 普通模块分层，例如 `config` / `session` / `context` / `tools` / `model` / `runtime`
- 用 `std.sys.cli` 解析 CLI 参数
- 用 `std.net.http.client` 做模型调用
- 用 `std.io.fs` / `std.sys.exec` / `std.io` 做工具实现

不要写：

- 假想的 `std.ai.*`
- 假想的 `std.agent.*`
- 编译器 builtin 风格的 AI 调用接口
- 把 Agent 能力直接写成语言内置特性
