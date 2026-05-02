# Icoo AI 代码生成参考

本文档面向会生成、解释或审查 Icoo 代码的 AI 系统。

目标不是介绍项目历史，而是提供一份**按当前实现对齐**的语言与原生 API 速查表，帮助 AI：

- 生成可运行的 Icoo 代码
- 选择正确语法
- 正确调用 builtin 与标准库
- 避免生成编译器内部辅助 API

相关文档：

- 语言总览：`docs/language-design.md`
- Runtime API：`docs/api.md`
- 当前状态：`docs/mvp-status.md`
- 架构分析：`docs/architecture-analysis-report.md`

---

## 1. AI 生成代码时的基本判断

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

### 1.2 AI 应优先遵守的规则

1. 变量声明优先用 `let` 或 `const`，不要生成 `var`。
2. 类方法定义**不写 `fn`**。
3. 继承写法是 `class Dog < Animal`，不是 `extends`。
4. `for-in` 可用于 array / string / object / module / iterator。
5. 需要错误传播时，优先使用 `expr?` 或 `try/catch/finally`。
6. 需要并发通道时，优先生成 `chan()` 与对象方法 `send/recv/...`。
7. 不要直接生成编译器内部 builtin，例如 `__select`、`__buildClass`。

---

## 2. 语法速查

## 2.1 变量与常量

```icoo
let count = 1
const name = "icoo"
```

## 2.2 函数

```icoo
fn add(a, b) {
  return a + b
}

let mul = fn(a, b) {
  return a * b
}
```

## 2.3 条件与循环

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

## 2.4 for-in

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

## 2.5 表达式

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

## 2.6 数组与对象

```icoo
let arr = [1, 2, 3]
let obj = {
  name: "icoo",
  version: 1,
}
```

## 2.7 模块导入导出

```icoo
import std.io as io
import std.fs as fs
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

## 2.8 错误与异常

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

## 2.9 类型与接口

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

## 2.10 类与继承

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

关键规则：

- 方法定义不写 `fn`
- 构造入口通常写 `init(...)`
- 通过 `this.xxx` 访问实例字段
- 通过 `super.xxx(...)` 调父类实现

## 2.11 装饰器

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

## 2.12 并发与 channel

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

以下 builtin 是**用户可直接调用**的公开能力。

### 3.1 `print(...args)`

输出参数到标准输出，不自动换行。

```icoo
print("hello", 1)
```

### 3.2 `println(...args)`

输出参数到标准输出，并在末尾换行。

```icoo
println("hello", 1)
```

### 3.3 `len(x)`

返回长度，支持：

- `string`
- `array`
- `object`

```icoo
println(len("abc"))
println(len([1, 2, 3]))
println(len({a: 1, b: 2}))
```

### 3.4 `typeOf(x)`

返回运行时类型名字符串。

```icoo
println(typeOf(1))
println(typeOf("hi"))
```

### 3.5 `chan(size?)`

创建 channel。

- 0 参数：无缓冲 channel
- 1 参数：缓冲区大小，必须是整数

```icoo
let ch = chan()
let ch2 = chan(16)
```

### 3.6 `panic(message)`

直接抛出运行时错误。

```icoo
panic("unexpected state")
```

适合：

- 明确不可恢复的内部错误
- 示例或测试中的硬失败

### 3.7 `error(message[, cause])`

构造错误值，返回的是语言级 `error` 值，不会自动抛出。

```icoo
return error("invalid config")
return error("open failed", err)
```

适合：

- 与 `expr?` 配合
- 作为函数返回值显式传播错误

### 3.8 `satisfies(obj, iface)`

检查对象是否满足接口要求。

当前实现按接口方法名检查对象上是否存在对应可调用字段。

```icoo
if satisfies(service, Greeter) {
  println("ok")
}
```

---

## 4. 不要直接生成的内部 builtin

这些 builtin 由编译器自动插入，**AI 不应手写生成**：

- `__select`
- `_tryCheck`
- `__buildClass`
- `__methodDef`
- `__methodProxy`
- `__superGet`

它们分别服务于：

- `select` 编译后运行时分发
- `expr?` 错误检查
- 类/方法/继承内部构造

结论：

- 用户代码请生成高级语法
- 不要把高级语法展开成这些底层 builtin

---

## 5. 标准库模块速查

## 5.1 core

### `std.io`

常用导出：

- `print`
- `println`
- `copy`
- `openReader`
- `openWriter`
- `readAll`

典型用途：控制台输出、Reader/Writer 风格 I/O。

```icoo
import std.io as io
io.println("hello")
```

说明：

- `openReader/openWriter` 返回对象
- reader/writer 对象包含 `close()` 方法

### `std.time`

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

```icoo
import std.time as time
let start = time.now()
time.sleep(100)
let text = time.format(start, "YYYY-MM-DD HH:mm:ss", "UTC")
```

### `std.math`

常用导出：

- `abs`
- `max`
- `min`
- `floor`
- `ceil`

```icoo
import std.math as math
println(math.max(3, 9))
```

## 5.2 format

### `std.json`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

```icoo
import std.json as json
let text = json.encode({name: "icoo"})
let obj = json.decode(text)
```

### `std.yaml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.toml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

### `std.xml`

导出：

- `encode`
- `decode`
- `fromFile`
- `saveToFile`

说明：

- `std.xml` 处理的节点通常使用对象结构表达
- 常见形态为 `{name, attrs?, text?, children?}`

## 5.3 system

### `std.fs`

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

```icoo
import std.fs as fs
if fs.exists("app.json") {
  let text = fs.readFile("app.json")
}
```

### `std.exec`

常用导出：

- `run(cmd[, argv])`

返回对象通常包含：

- `ok`
- `code`
- `stdout`
- `stderr`
- `command`
- `exitCode`

```icoo
import std.exec as exec
let result = exec.run("git", ["status"])
println(result.stdout)
```

### `std.os`

常用导出：

- `args`
- `cwd`
- `tempDir`
- `getEnv`
- `setEnv`
- `mkdirAll`
- `remove`
- `removeAll`

### `std.host`

常用导出：

- `arch`
- `goos`
- `hostname`
- `numCPU`
- `pid`

适合读取宿主环境信息。

## 5.4 database

### `std.db`

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

`table(name)` 返回一个轻量 ORM / 查询对象，常用方法：

- `select(columns)`
- `where(filters)`
- `whereRaw(sql, args?)`
- `orderBy(sql)`
- `limit(n)`
- `offset(n)`
- `all()` / `get()`
- `first()`
- `count()`
- `insert(data)`
- `update(data)`
- `delete()`

其中：

- `where({field: value})` 默认生成等值条件
- `where({field: null})` 会生成 `IS NULL`
- `where({field: [1, 2, 3]})` 会生成 `IN (...)`
- `update()` / `delete()` 默认要求先调用 `where(...)` 或 `whereRaw(...)`，避免整表误修改

```icoo
import std.db as db
let conn = db.sqlite("demo.db")
let rows = conn.query("select 1 as n")
conn.close()
```

```icoo
import std.db as db

let conn = db.sqlite(":memory:")
conn.exec("create table users (id integer primary key, name text, score integer)")

let users = conn.table("users")
users.insert({name: "Ada", score: 10})
users.insert({name: "Linus", score: 12})

let top = users.orderBy("score desc").first()
let rows = users.whereRaw("score >= ?", [10]).all()
users.where({name: "Ada"}).update({score: 15})

conn.close()
```

### `std.redis`

常用导出：

- `open(url)`
- `connect(options)`

返回 redis 对象方法：

- `close()`
- `ping()`
- `get(key)`
- `set(key, value[, ttlMs])`
- `del(key)`
- `exists(key)`
- `expire(key, ttlMs)`
- `ttl(key)`
- `incr(key)`
- `incrBy(key, delta)`
- `hSet(key, object)`
- `hGet(key, field)`
- `hGetAll(key)`

说明：

- `set` 对字符串按原样写入；数组/对象会先编码为 JSON 字符串
- `ttl` 返回剩余毫秒数；无过期时间或 key 不存在时返回 `null`
- `hGetAll` 返回字段值均为字符串的对象

```icoo
import std.redis as redis

let client = redis.open("redis://127.0.0.1:6379/0")
client.set("session:1", {user: "icoo"}, 60000)
let raw = client.get("session:1")
let count = client.incr("counter")
client.close()
```

## 5.5 data

### `std.crypto`

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

### `std.uuid`

常用导出：

- `v4`
- `isValid`

### `std.compress`

常用导出：

- `gzipCompress`
- `gzipDecompress`
- `zlibCompress`
- `zlibDecompress`

## 5.6 net

### `std.http.client`

常用导出：

- `get`
- `getJSON`
- `post`
- `put`
- `delete`
- `request`
- `requestJSON`
- `download`

```icoo
import std.http.client as http
let resp = http.get("https://example.com")
```

### `std.http.server`

常用导出：

- `listen`
- `forward`

说明：

- `listen(...)` 返回 server 对象
- server 对象含 `close()` 方法

### `std.net.websocket.client`

导出：

- `connect`

连接对象方法：

- `read()`
- `write(x)`
- `close()`

### `std.net.websocket.server`

导出：

- `listen`

handler 通常会拿到：

- websocket 连接对象
- 请求对象

连接对象包含：

- `read()`
- `write(x)`
- `close()`

### `std.net.sse.client`

导出：

- `connect`

client 对象方法：

- `read()`
- `close()`

### `std.net.sse.server`

导出：

- `listen`

连接对象方法：

- `send(x)`
- `close()`

### `std.net.socket.client`

导出：

- `connectTCP`
- `dialUDP`

连接对象方法：

- `read(size)`
- `write(x)`
- `close()`

### `std.net.socket.server`

导出：

- `listenTCP`
- `listenUDP`

说明：

- TCP handler 通常拿到 conn 对象
- UDP handler 通常拿到 `{data, addr, reply}` 形态对象

## 5.7 express

### `std.express`

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

server 对象方法：

- `close()`

适合生成 Web 服务、路由、中间件示例。

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

而不是把所有失败都写成 `panic(...)`。

### 6.2 面向模块别名生成代码

更推荐：

```icoo
import std.json as json
import std.fs as fs
```

而不是在代码里反复写完整模块名。

### 6.3 channel 使用对象方法

更推荐：

```icoo
let ch = chan()
ch.send(1)
let v, ok = ch.recv()
```

不要生成不存在的自由函数：

```icoo
send(ch, 1)
recv(ch)
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

错误：

```icoo
class User {
  fn greet() {
    return this.name
  }
}
```

---

## 7. AI 常见易错点

1. 把类继承写成 `extends`。
2. 在类体里给方法加 `fn`。
3. 直接生成内部 builtin，如 `__superGet(...)`。
4. 把 channel API 写成自由函数而不是对象方法。
5. 忘记 `expr?` 只对返回 `error` 值的函数有意义。
6. 把 `error(...)` 当成自动抛异常；实际上它是构造错误值。
7. 误以为 `satisfies` 是编译期关键字；它是 builtin。
8. 为简单场景生成过于复杂的 `try/catch/finally`，本语言也支持更轻量的 `expr?`。

---

## 8. 推荐给 AI 的生成策略

### 8.1 简单脚本

优先使用：

- `let` / `const`
- 顶层函数
- `if` / `for` / `while`
- `std.io` / `std.fs` / `std.json`

### 8.2 可恢复错误

优先使用：

- `error(...)`
- `expr?`
- 必要时 `try/catch/finally`

### 8.3 Web / 网络示例

优先从这些模块选：

- `std.http.client`
- `std.http.server`
- `std.express`
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

不要手工展开成底层运行时 helper。
