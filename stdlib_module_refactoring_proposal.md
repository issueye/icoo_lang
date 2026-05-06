# Icoo Lang 标准库模块分类重构建议

> 参考 `std.net.xx` 的命名方式，将其他模块也进行层级化分类  
> **⚠️ 破坏性变更：不提供向后兼容，旧模块名直接废弃**

---

## 变更概览

| 项目 | 说明 |
|------|------|
| **变更类型** | Breaking Change（破坏性变更） |
| **向后兼容** | ❌ 不提供 |
| **旧模块名** | 直接废弃，返回 `nil, false` |
| **目标版本** | v0.3.0 或 v1.0.0 |
| **影响范围** | 25 个模块需修改，8 个模块保持不变 |

---

## 一、当前模块现状

### 1.1 现有模块列表（27个）

| 当前模块名 | 分类 | 说明 |
|-----------|------|------|
| std.io | core | 输入输出 |
| std.time | core | 时间操作 |
| std.math | core | 数学计算 |
| std.object | core | 对象操作 |
| std.observe | core | 观测能力 |
| std.service | core | 服务能力 |
| std.cache | core | 缓存 |
| std.template | core | 模板引擎 |
| std.db | database | 数据库操作 |
| std.orm | database | ORM能力 |
| std.redis | database | Redis客户端 |
| std.json | format | JSON编解码 |
| std.yaml | format | YAML编解码 |
| std.toml | format | TOML编解码 |
| std.xml | format | XML编解码 |
| std.csv | format | CSV编解码 |
| std.fs | system | 文件系统 |
| std.exec | system | 系统命令执行 |
| std.os | system | 操作系统信息 |
| std.host | system | 宿主信息 |
| std.express | web | Web框架 |
| std.net.http.client | net | HTTP客户端 |
| std.net.http.server | net | HTTP服务端 |
| std.net.websocket.client | net | WebSocket客户端 |
| std.net.websocket.server | net | WebSocket服务端 |
| std.net.sse.client | net | SSE客户端 |
| std.net.sse.server | net | SSE服务端 |
| std.net.socket.client | net | Socket客户端 |
| std.net.socket.server | net | Socket服务端 |
| std.crypto | data | 加密解密 |
| std.uuid | data | UUID生成 |
| std.compress | data | 压缩解压 |

### 1.2 当前命名问题

1. **命名不一致**：`std.net.xx` 采用三级命名，其他模块多为二级命名
2. **分类粒度不均**：`core` 包含 8 个模块，`data` 只有 3 个
3. **职责边界模糊**：`std.express` 作为 Web 框架，与 `std.net.http` 关系紧密但命名不统一

---

## 二、重构方案

### 2.1 命名规范

采用统一的三级命名规范：`std.<领域>.<子领域>.<功能>`

- **一级**：`std` - 标准库前缀
- **二级**：`<领域>` - 功能大分类（core, io, net, db, web, data, sys等）
- **三级**：`<子领域>.<功能>` - 细分功能（如 http.client, fs.file等）

### 2.2 重构后的模块分类

#### 🔹 std.core - 核心基础（4个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.core.object | std.object | 对象操作 |
| std.core.observe | std.observe | 观测能力 |
| std.core.service | std.service | 服务能力 |
| std.core.cache | std.cache | 缓存 |

> **说明**：将最基础的运行时能力保留在 core，其他移至更具体的分类

#### 🔹 std.io - 输入输出（3个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.io.console | std.io | 控制台输入输出（print/println等） |
| std.io.fs | std.fs | 文件系统操作 |
| std.io.template | std.template | 模板引擎 |

> **说明**：将 IO 相关功能统一归类，fs 从 system 移至 io

#### 🔹 std.net - 网络通信（9个模块）【保持现状】

| 模块名 | 说明 |
|--------|------|
| std.net.http.client | HTTP客户端 |
| std.net.http.server | HTTP服务端 |
| std.net.websocket.client | WebSocket客户端 |
| std.net.websocket.server | WebSocket服务端 |
| std.net.sse.client | SSE客户端 |
| std.net.sse.server | SSE服务端 |
| std.net.socket.client | Socket客户端 |
| std.net.socket.server | Socket服务端 |

> **说明**：net 分类已是最佳实践，保持不变

#### 🔹 std.web - Web开发（1个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.web.express | std.express | Web框架 |

> **说明**：express 作为 Web 框架，与 net 区分，单独归类为 web

#### 🔹 std.db - 数据库（3个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.db.sql | std.db | SQL数据库操作 |
| std.db.orm | std.orm | ORM能力 |
| std.db.redis | std.redis | Redis客户端 |

> **说明**：统一为三级命名，std.db.sql 替代 std.db

#### 🔹 std.data - 数据处理（6个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.data.json | std.json | JSON编解码 |
| std.data.yaml | std.yaml | YAML编解码 |
| std.data.toml | std.toml | TOML编解码 |
| std.data.xml | std.xml | XML编解码 |
| std.data.csv | std.csv | CSV编解码 |
| std.data.compress | std.compress | 压缩解压 |

> **说明**：将所有数据序列化和处理功能统一归类

#### 🔹 std.crypto - 加密安全（1个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.crypto.hash | std.crypto | 哈希/加密/解密 |
| std.crypto.uuid | std.uuid | UUID生成 |

> **说明**：加密相关功能单独归类，uuid 作为安全相关功能移至 crypto

#### 🔹 std.sys - 系统信息（4个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.sys.os | std.os | 操作系统信息 |
| std.sys.host | std.host | 宿主信息 |
| std.sys.exec | std.exec | 系统命令执行 |
| std.sys.process | (新增) | 进程管理 |

> **说明**：fs 移至 io，sys 专注于系统信息和进程管理

#### 🔹 std.math - 数学计算（1个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.math.basic | std.math | 基础数学计算 |
| std.math.random | (新增) | 随机数生成 |
| std.math.stat | (新增) | 统计计算 |

> **说明**：为未来扩展预留空间

#### 🔹 std.time - 时间日期（1个模块）

| 新模块名 | 原模块名 | 说明 |
|---------|---------|------|
| std.time.basic | std.time | 基础时间操作 |
| std.time.format | (新增) | 时间格式化 |
| std.time.zone | (新增) | 时区处理 |

> **说明**：为未来扩展预留空间

---

## 三、重构对照表

### 3.1 完整映射关系

| 新模块名 | 原模块名 | 变更类型 |
|---------|---------|---------|
| std.core.object | std.object | 重命名 |
| std.core.observe | std.observe | 重命名 |
| std.core.service | std.service | 重命名 |
| std.core.cache | std.cache | 重命名 |
| std.io.console | std.io | 重命名 |
| std.io.fs | std.fs | 分类调整+重命名 |
| std.io.template | std.template | 分类调整+重命名 |
| std.net.http.client | std.net.http.client | 保持不变 |
| std.net.http.server | std.net.http.server | 保持不变 |
| std.net.websocket.client | std.net.websocket.client | 保持不变 |
| std.net.websocket.server | std.net.websocket.server | 保持不变 |
| std.net.sse.client | std.net.sse.client | 保持不变 |
| std.net.sse.server | std.net.sse.server | 保持不变 |
| std.net.socket.client | std.net.socket.client | 保持不变 |
| std.net.socket.server | std.net.socket.server | 保持不变 |
| std.web.express | std.express | 重命名 |
| std.db.sql | std.db | 重命名 |
| std.db.orm | std.orm | 重命名 |
| std.db.redis | std.redis | 重命名 |
| std.data.json | std.json | 重命名 |
| std.data.yaml | std.yaml | 重命名 |
| std.data.toml | std.toml | 重命名 |
| std.data.xml | std.xml | 重命名 |
| std.data.csv | std.csv | 重命名 |
| std.data.compress | std.compress | 重命名 |
| std.crypto.hash | std.crypto | 重命名 |
| std.crypto.uuid | std.uuid | 分类调整+重命名 |
| std.sys.os | std.os | 重命名 |
| std.sys.host | std.host | 重命名 |
| std.sys.exec | std.exec | 重命名 |
| std.math.basic | std.math | 重命名 |
| std.time.basic | std.time | 重命名 |

### 3.2 统计对比

| 分类 | 原模块数 | 新模块数 | 变化 |
|------|---------|---------|------|
| core | 8 | 4 | -4 |
| io | 1 | 3 | +2 |
| net | 8 | 8 | 0 |
| web | 0 | 1 | +1 |
| db | 3 | 3 | 0 |
| data | 3 | 6 | +3 |
| crypto | 0 | 2 | +2 |
| sys | 4 | 3 | -1 |
| math | 1 | 1 | 0 |
| time | 1 | 1 | 0 |
| **合计** | **29** | **32** | **+3** |

---

## 四、目录结构调整

### 4.1 建议的目录结构

```
internal/stdlib/
├── core/              # 核心运行时能力
│   ├── object.go
│   ├── observe.go
│   ├── service.go
│   └── cache.go
├── io/                # 输入输出
│   ├── console.go     # 原 io.go
│   ├── fs.go          # 从 system 移入
│   └── template.go    # 从 core 移入
├── net/               # 网络通信（保持现状）
│   ├── http.go
│   ├── websocket.go
│   ├── sse.go
│   └── socket.go
├── web/               # Web框架
│   └── express.go     # 从 express 移入
├── db/                # 数据库
│   ├── sql.go         # 原 db.go
│   ├── orm.go
│   └── redis.go
├── data/              # 数据处理
│   ├── json.go        # 从 format 移入
│   ├── yaml.go
│   ├── toml.go
│   ├── xml.go
│   ├── csv.go
│   └── compress.go    # 从 data 保留
├── crypto/            # 加密安全
│   ├── hash.go        # 原 crypto.go
│   └── uuid.go        # 从 data 移入
├── sys/               # 系统信息
│   ├── os.go
│   ├── host.go
│   └── exec.go
├── math/              # 数学计算
│   └── basic.go       # 原 math.go
├── time/              # 时间日期
│   └── basic.go       # 原 time.go
└── modules.go         # 模块注册入口
```

### 4.2 废弃的目录

- `format/` → 合并到 `data/`
- `system/` → 重命名为 `sys/`，fs 移至 `io/`
- `express/` → 合并到 `web/`
- `database/` → 重命名为 `db/`

---

## 五、代码变更示例

### 5.1 modules.go 变更

```go
// 重构前
switch spec {
case "std.io":
    return stdcore.LoadStdIOModule(), true
case "std.fs":
    return stdsystem.LoadStdFSModule(), true
// ...
}

// 重构后
switch spec {
case "std.io.console":
    return stdio.LoadStdIOConsoleModule(), true
case "std.io.fs":
    return stdio.LoadStdIOFSModule(), true
// ...
}
```

---

## 六、破坏性变更说明

> **⚠️ 重要提示：本次重构为破坏性变更（Breaking Change），不提供向后兼容。**

### 6.1 变更原则

- **无别名映射**：旧模块名不再可用，直接返回 `nil, false`
- **无 deprecation 周期**：直接替换，不提供过渡期
- **版本标记**：作为重大版本变更（如 v0.3.0 或 v1.0.0）发布

### 6.2 对用户代码的影响

| 旧导入语句 | 新导入语句 | 状态 |
|-----------|-----------|------|
| `import std.io` | `import std.io.console` | ❌ 必须修改 |
| `import std.fs` | `import std.io.fs` | ❌ 必须修改 |
| `import std.json` | `import std.data.json` | ❌ 必须修改 |
| `import std.yaml` | `import std.data.yaml` | ❌ 必须修改 |
| `import std.toml` | `import std.data.toml` | ❌ 必须修改 |
| `import std.xml` | `import std.data.xml` | ❌ 必须修改 |
| `import std.csv` | `import std.data.csv` | ❌ 必须修改 |
| `import std.template` | `import std.io.template` | ❌ 必须修改 |
| `import std.object` | `import std.core.object` | ❌ 必须修改 |
| `import std.observe` | `import std.core.observe` | ❌ 必须修改 |
| `import std.service` | `import std.core.service` | ❌ 必须修改 |
| `import std.cache` | `import std.core.cache` | ❌ 必须修改 |
| `import std.db` | `import std.db.sql` | ❌ 必须修改 |
| `import std.orm` | `import std.db.orm` | ❌ 必须修改 |
| `import std.redis` | `import std.db.redis` | ❌ 必须修改 |
| `import std.crypto` | `import std.crypto.hash` | ❌ 必须修改 |
| `import std.uuid` | `import std.crypto.uuid` | ❌ 必须修改 |
| `import std.compress` | `import std.data.compress` | ❌ 必须修改 |
| `import std.fs` | `import std.io.fs` | ❌ 必须修改 |
| `import std.exec` | `import std.sys.exec` | ❌ 必须修改 |
| `import std.os` | `import std.sys.os` | ❌ 必须修改 |
| `import std.host` | `import std.sys.host` | ❌ 必须修改 |
| `import std.express` | `import std.web.express` | ❌ 必须修改 |
| `import std.math` | `import std.math.basic` | ❌ 必须修改 |
| `import std.time` | `import std.time.basic` | ❌ 必须修改 |
| `import std.net.http.client` | `import std.net.http.client` | ✅ 保持不变 |
| `import std.net.http.server` | `import std.net.http.server` | ✅ 保持不变 |
| `import std.net.websocket.client` | `import std.net.websocket.client` | ✅ 保持不变 |
| `import std.net.websocket.server` | `import std.net.websocket.server` | ✅ 保持不变 |
| `import std.net.sse.client` | `import std.net.sse.client` | ✅ 保持不变 |
| `import std.net.sse.server` | `import std.net.sse.server` | ✅ 保持不变 |
| `import std.net.socket.client` | `import std.net.socket.client` | ✅ 保持不变 |
| `import std.net.socket.server` | `import std.net.socket.server` | ✅ 保持不变 |

### 6.3 迁移工具

提供迁移脚本，自动替换项目中的导入语句：

```bash
# 自动迁移脚本使用示例
icoo migrate --from 0.2.0 --to 0.3.0 ./my-project

# 或手动执行替换
sed -i 's/import std\.io/import std.io.console/g' *.ic
sed -i 's/import std\.fs/import std.io.fs/g' *.ic
sed -i 's/import std\.json/import std.data.json/g' *.ic
# ... 其他替换规则
```

### 6.4 版本发布说明

**Release Note 示例：**

```markdown
## v0.3.0 - 标准库模块命名重构

### ⚠️ Breaking Changes

本次版本对标准库模块命名进行了全面重构，采用统一的三级命名规范。
所有旧模块名不再可用，请在升级前更新您的代码。

#### 变更列表
- `std.io` → `std.io.console`
- `std.fs` → `std.io.fs`
- `std.json` → `std.data.json`
- ... (完整列表见文档)

#### 迁移步骤
1. 使用迁移脚本：`icoo migrate --from 0.2.0 --to 0.3.0 .`
2. 手动检查并修复未自动替换的导入
3. 运行测试验证功能正常

#### 未变更的模块
以下模块保持原名称，无需修改：
- `std.net.http.client/server`
- `std.net.websocket.client/server`
- `std.net.sse.client/server`
- `std.net.socket.client/server`
```

---

## 七、总结

### 7.1 重构收益

1. **命名一致性**：所有模块采用统一的三级命名规范
2. **职责清晰**：每个分类的职责更加明确
3. **可扩展性**：为未来新增模块预留了清晰的分类位置
4. **易于理解**：用户可以根据模块名快速判断功能领域

### 7.2 关键变更

- **core**：精简为 4 个最基础的运行时能力
- **io**：整合所有输入输出相关功能（console, fs, template）
- **net**：保持不变，作为最佳实践参考
- **web**：express 单独归类，与 net 区分
- **data**：整合所有数据序列化格式（json, yaml, toml, xml, csv, compress）
- **crypto**：新增分类，整合加密和 UUID
- **sys**：精简为系统信息和进程管理
- **db, math, time**：保持功能，统一命名规范

### 7.3 实施建议（破坏性变更）

1. **版本规划**：
   - 作为 **v0.3.0** 或 **v1.0.0** 发布（重大版本变更）
   - 在版本号中明确标识 breaking changes

2. **代码实施**：
   - 直接替换 `modules.go` 中的 switch case，不保留旧命名
   - 重构目录结构，删除旧目录
   - 更新所有内部引用和示例代码

3. **文档更新**：
   - 同步更新所有文档中的导入示例
   - 更新 `docs/api.md` 中的模块列表
   - 更新 `docs/language-design.md` 中的标准库章节
   - 更新 `examples/` 目录下的所有示例

4. **迁移工具**：
   - 提供 `icoo migrate` 命令，自动替换项目中的导入语句
   - 提供详细的迁移指南和替换对照表

5. **IDE 支持**：
   - 更新 VSCode 插件的代码片段
   - 更新自动补全列表
   - 添加旧模块名的错误提示（"该模块已移至 std.xxx，请更新导入"）

6. **发布流程**：
   ```
   1. 代码重构完成
   2. 内部示例全部更新
   3. 文档全部更新
   4. 迁移工具发布
   5. 发布 Release Note（明确标注 Breaking Changes）
   6. 社区公告
   ```
