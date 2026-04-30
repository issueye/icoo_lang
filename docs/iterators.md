# Icoo 迭代器与 `for-in` 指南

本文档说明 Icoo 当前实现中的统一迭代器协议，以及 `for-in` 的单绑定和双绑定语义。

## 概览

Icoo 的 `for-in` 不直接依赖某一种具体类型，而是依赖统一协议：

- 可迭代值需要提供 `iter()`
- 迭代器需要提供 `next()`

因此，数组、字符串、对象、模块，以及迭代器本身，都可以用同一套 `for-in` 语法消费。

## `for-in` 的两种形式

### 单绑定

```icoo
for item in iterable {
  println(item)
}
```

单绑定读取的是每一步返回结果中的 `item` 字段。

### 双绑定

```icoo
for key, value in iterable {
  println(key)
  println(value)
}
```

双绑定读取的是每一步返回结果中的：

- `key`
- `value`

### 忽略绑定

可以使用 `_` 忽略不需要的值：

```icoo
for _, value in [4, 5, 6] {
  println(value)
}
```

## `next()` 的统一返回结构

内建迭代器的 `next()` 总是返回一个对象：

```icoo
{
  key: <当前键或索引，结束时为 null>,
  value: <当前值，结束时为 null>,
  item: <单绑定使用的值，结束时为 null>,
  done: <是否结束>
}
```

结束时：

```icoo
{
  key: null,
  value: null,
  item: null,
  done: true
}
```

## 各内建类型的行为

### 数组

数组按索引顺序迭代。

```icoo
let arr = [4, 5, 6]

for idx, value in arr {
  println(idx)
  println(value)
}
```

等价的显式协议调用：

```icoo
let iter = [4, 5, 6].iter()
let step = iter.next()

if !step.done {
  println(step.key)
  println(step.value)
  println(step.item)
}
```

规则：

- `key` 是索引：`0`、`1`、`2`...
- `value` 是数组元素
- `item == value`

## 字符串

字符串按 Unicode rune 迭代，而不是按字节迭代。

```icoo
let text = "你好a"
let out = ""

for idx, ch in text {
  println(idx)
  out = out + ch
}

println(out)
```

规则：

- `key` 是 rune 索引
- `value` 是单个字符组成的字符串
- `item == value`

## 对象

对象默认按字段名排序后的稳定顺序迭代。

```icoo
let obj = {b: 2, a: 1, c: 3}

for key, value in obj {
  println(key)
  println(value)
}
```

如果使用单绑定：

```icoo
for pair in obj {
  println(pair.key)
  println(pair.value)
}
```

规则：

- `key` 是字段名
- `value` 是字段值
- `item` 是 `{ key, value }`

因此对象的单绑定和双绑定都成立，只是读取方式不同。

## 模块

模块默认按导出名排序后的稳定顺序迭代。

```icoo
import "./math.ic" as math

for key, value in math {
  println(key)
  println(value)
}
```

规则与对象类似：

- `key` 是导出名
- `value` 是导出值
- `item` 是 `{ key, value }`

## 直接迭代 iterator

迭代器本身也可以直接用于 `for-in`。

```icoo
let iter = [7, 8].iter()

for idx, value in iter {
  println(idx)
  println(value)
}
```

这依赖迭代器自己的 `iter()` 直接返回自身。

## 自定义对象迭代行为

对象如果自身定义了 `iter` 字段，会优先使用该字段，而不是默认的对象字段遍历逻辑。

```icoo
let obj = {
  label: "fallback",
  iter: fn() {
    return ["x", "y"].iter()
  }
}

let out = ""
for item in obj {
  out = out + item
}

println(out)
```

上面的输出应为：

```text
xy
```

这意味着对象可以把自己的迭代语义代理到其他可迭代值上。

## 何时使用单绑定或双绑定

建议：

- 只关心元素本身时，用单绑定
- 需要索引、字段名、导出名时，用双绑定

例如：

```icoo
for ch in "ab" {
  println(ch)
}

for idx, ch in "ab" {
  println(idx)
  println(ch)
}
```

## 端到端示例

下面这个例子覆盖了当前测试中的几种常见迭代方式：

```icoo
let text = "ab"
let out = ""
let textIndexSum = 0

for idx, ch in text {
  textIndexSum = textIndexSum + idx
  out = out + ch
}

println(out)
println(textIndexSum)

let obj = {b: 2, a: 1}
let keys = ""
let total = 0

for key, value in obj {
  keys = keys + key
  total = total + value
}

println(keys)
println(total)

let arr = [1, 2, 3]
let arrIndexSum = 0
let sum = 0

for idx, value in arr {
  arrIndexSum = arrIndexSum + idx
  sum = sum + value
}

println(arrIndexSum)
println(sum)
```

预期输出：

```text
ab
1
ab
3
3
6
```

## 当前实现约束

当前内建协议已覆盖：

- array
- string
- object
- module
- iterator

尚未在语言层引入独立的 `Iterator` 类型声明语法；当前协议主要通过运行时对象属性约定工作。

## 相关文档

- `docs/language-design.md`
- `docs/mvp-roadmap.md`
