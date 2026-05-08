# Icoo 包说明

`icoo` 支持类似 Java `jar` 的可复用包机制，用来做两类事情：

- 把一组 Icoo 源码打成一个可分发的 `.icpkg`
- 在别的项目里通过 `pkg:` 或本地文件方式导入这个包

这套机制同时支持“库包”和“可运行包”。

## 1. 包和 bundle 的区别

Icoo 目前有两种归档格式：

- `.icb`
  应用 bundle，主要给 `icoo run`、`icoo build`、`icoo inspect`、`icoo extract` 使用
- `.icpkg`
  可复用包，既可以被别的项目导入，也可以带有自己的可运行入口

两者底层结构接近，都会记录：

- `entry`
  运行入口模块
- `entry_function`
  入口模块加载后调用的函数
- `export`
  被别的项目导入时暴露的模块
- `modules`
  打进归档的源码文件
- `packages`
  一并打入的嵌套依赖包

简单理解：

- `.icb` 更偏“应用产物”
- `.icpkg` 更偏“复用产物”

## 2. 包的两种角色

### 2.1 库包

库包的重点是“对外导出什么”。

例如：

```text
pkg/config/
  pkg.toml
  lib.ic
  src/
    main.ic
    args.ic
    defaults.ic
```

常见做法是：

- `src/main.ic`
  作为包内部主入口，负责聚合实际实现
- `lib.ic` 或 `src/main.ic`
  作为 `export` 暴露给外部

如果这个包主要是给别人 `import` 用，建议把导出面收敛成一个稳定模块，而不是把所有内部文件直接暴露出去。

### 2.2 可运行包

一个 `.icpkg` 也可以自带运行入口：

```powershell
icoo run .\dist\demo.icpkg
```

这意味着一个包可以同时具备：

- “像库一样被导入”
- “像应用一样能直接运行”

如果既要导入又要运行，建议把：

- `entry`
  指向运行入口
- `export`
  指向稳定导出模块

分开配置，不要强行共用一个文件。

## 3. `pkg.toml`

包目录通过 `pkg.toml` 描述自身：

```toml
[package]
name = "issueye/agent/pkg/config"
version = "0.1.0"
entry = "src/main.ic"
entry_function = "main"
export = "lib.ic"
root_alias = "@"
```

字段说明：

- `name`
  包名。用于 `pkg:` 导入时的逻辑名称
- `version`
  包版本
- `entry`
  包运行时加载的入口模块
- `entry_function`
  包运行时调用的入口函数
- `export`
  作为依赖被导入时暴露的模块
- `root_alias`
  包内部源码使用的项目根别名，默认通常是 `@`

当前默认值来自 `icoo init-pkg`：

- `version = "0.1.0"`
- `entry = "src/main.ic"`
- `entry_function = "main"`
- `export = "lib.ic"`
- `root_alias = "@"`

## 4. 导入方式

Icoo 支持两种包导入方式。

### 4.1 本地文件导入

```icoo
import "./libs/greeter.icpkg" as greeter

if greeter.hello("icoo") != "hello icoo" {
  panic("unexpected package result")
}
```

适合：

- 临时联调
- 没有固定包名的本地包
- 显式依赖某个归档文件

### 4.2 命名包导入

```icoo
import "pkg:acme/greeter" as greeter
```

适合：

- 项目内复用
- 稳定模块名导入
- 多个子包协作

命名包的解析位置是：

- `<projectRoot>/.icoo/packages/<name>.icpkg`
- `<projectRoot>/packages/<name>.icpkg`

例如：

```icoo
import "pkg:issueye/agent/pkg/config" as agentConfig
import "pkg:issueye/agent/pkg/tools" as agentTools
```

会查找：

```text
.icoo/packages/issueye/agent/pkg/config.icpkg
.icoo/packages/issueye/agent/pkg/tools.icpkg
```

## 5. 推荐目录结构

### 5.1 独立库包

```text
demo/
  pkg.toml
  lib.ic
  build.ps1
  src/
    main.ic
    util.ic
  examples/
```

建议职责：

- `pkg.toml`
  包元数据
- `lib.ic`
  对外导出面
- `src/main.ic`
  包内部聚合入口
- `build.ps1`
  当前包自己的打包脚本
- `examples/`
  演示该包的使用方式

### 5.2 项目内子包

如果一个应用项目内部要拆多个复用包，推荐：

```text
apps/agent/
  project.toml
  src/
  pkg/
    config/
      pkg.toml
      lib.ic
      build.ps1
      src/
    session/
      pkg.toml
      lib.ic
      build.ps1
      src/
    tools/
      pkg.toml
      lib.ic
      build.ps1
      src/
```

然后应用主项目通过 `pkg:` 导入这些子包：

```icoo
import "pkg:issueye/agent/pkg/config" as agentConfig
import "pkg:issueye/agent/pkg/session" as agentSession
import "pkg:issueye/agent/pkg/tools" as agentTools
```

## 6. `lib.ic` 和 `src/main.ic` 的关系

推荐模式是：

- `src/main.ic`
  做实际实现聚合
- `lib.ic`
  做对外导出

例如：

```icoo
import "./src/main.ic" as mainModule

export {
  ArgsConfig: mainModule.ArgsConfig,
  Defaults: mainModule.Defaults,
  MergeConfig: mainModule.MergeConfig
}
```

如果你希望包的导出入口更直接，也可以把 `export` 指向 `src/main.ic`。

何时用哪种方式：

- 想给外部一个稳定薄封装层，用 `lib.ic`
- 想减少一层中转，直接导出 `src/main.ic`

## 7. 初始化命令

### 7.1 初始化独立包

```powershell
icoo init-pkg .\demo --name acme/demo
```

也可以指定更多参数：

```powershell
icoo init-pkg .\demo `
  --name acme/demo `
  --version 1.0.0 `
  --entry src/main.ic `
  --entry-fn main `
  --export lib.ic `
  --root-alias @
```

生成内容包括：

- `pkg.toml`
- `lib.ic`
- `src/main.ic`
- `examples/`
- `build.ps1`

### 7.2 初始化子包

```powershell
icoo init-subpkg .\pkg\config --parent issueye/agent
```

这会自动推导出包名：

```text
issueye/agent/pkg/config
```

适合在一个大项目里快速拆出 `pkg/config`、`pkg/tools` 这类复用单元。

## 8. 打包命令

### 8.1 使用 `icoo package`

```powershell
icoo package .\demo .\dist\demo.icpkg
```

也可以覆盖元数据：

```powershell
icoo package .\demo .\dist\demo.icpkg `
  --name acme/demo `
  --version 1.0.0 `
  --export lib.ic
```

如果目标目录本身有 `pkg.toml`，则：

- `--name` 不传时，读取 `pkg.toml` 的 `name`
- `--version` 不传时，读取 `pkg.toml` 的 `version`
- `--export` 不传时，读取 `pkg.toml` 的 `export`

### 8.2 使用生成的 `build.ps1`

`icoo init-pkg` / `icoo init-subpkg` 会自动生成当前包自己的 `build.ps1`。

例如：

```powershell
& .\build.ps1
```

脚本会：

- 自动定位仓库根
- 自动构建 `icoo.exe`（如果不存在）
- 执行 `icoo package`
- 把 `.icpkg` 输出到 `dist/`

## 9. 运行、分发和复用

### 9.1 运行源项目

```powershell
icoo run .\demo
```

### 9.2 运行包

```powershell
icoo run .\dist\demo.icpkg
```

### 9.3 构建二进制

如果目标是应用分发，通常不是直接分发 `.icpkg`，而是把项目构建成可执行文件：

```powershell
icoo build .\demo .\dist\demo.exe
```

更常见的做法是：

- 子包分发为 `.icpkg`
- 顶层应用分发为 `.exe`

## 10. 包设计建议

### 10.1 库包建议

- 导出面尽量小
- 不要让调用方依赖内部 `src/...` 文件路径
- 用 `lib.ic` 或聚合模块统一导出稳定 API

### 10.2 子包建议

- 子包内部尽量使用相对导入，如 `./args.ic`
- 不要让子包实现强依赖宿主项目的 `@/...` 目录结构
- 这样同一份子包源码既能在项目内使用，也能独立打包

### 10.3 应用包建议

- `entry` 用于运行
- `export` 用于导入
- 两者职责分开更清晰

例如顶层应用包：

- `entry = "src/main.ic"`
- `export = "src/app/app.ic"`

这样：

- `icoo run app.icpkg` 走应用入口
- `import "pkg:issueye/agent"` 则拿到导出 API

## 11. 常见问题

### 11.1 为什么 `pkg:` 导入找不到包？

先检查：

- 包文件是否存在于 `.icoo/packages/...` 或 `packages/...`
- 包名和目录是否一致
- 导入名是否写成 `pkg:scope/name`

### 11.2 为什么包能运行但不能导入？

通常是 `export` 没配好。

运行走的是：

- `entry`
- `entry_function`

导入走的是：

- `export`

这三者不是一回事。

### 11.3 为什么建议子包内部多用相对导入？

因为子包往往既要：

- 在宿主项目里开发
- 又要单独打成 `.icpkg`

相对导入更稳定，不会把宿主项目的目录结构泄漏进包内部。

### 11.4 为什么顶层应用有时也要配置 `export`？

因为有些应用本身也想被别的项目当库导入。

例如：

- 运行时是 CLI
- 导入时暴露 `App`、`Config`、`Runtime` 等 API

这种场景下，应用包既是“应用”，也是“库”。

## 12. 最小工作流示例

### 12.1 创建包

```powershell
icoo init-pkg .\demo --name acme/demo
```

### 12.2 编写实现

```icoo
// src/main.ic
export fn hello(name) {
  return "hello " + name
}

fn main() {
  return hello("icoo")
}
```

### 12.3 打包

```powershell
& .\demo\build.ps1
```

### 12.4 在另一个项目中导入

把产物放到：

```text
consumer/.icoo/packages/acme/demo.icpkg
```

然后：

```icoo
import "pkg:acme/demo" as demo

if demo.hello("world") != "hello world" {
  panic("unexpected result")
}
```
