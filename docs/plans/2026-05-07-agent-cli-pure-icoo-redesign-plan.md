# Agent CLI Pure Icoo Redesign Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 以不兼容式重构的方式，用纯 `icoo_lang` 实现新的 `icoo agent` 主线工程；Agent 本身不采用 Go + Icoo 混合开发，而是作为完整的 Icoo 应用开发。开发过程中暴露出的能力缺口，优先通过增强 `icoo_lang` 的语法、标准库、runtime 与通用 CLI 机制来解决。

**Architecture:** 不在 `examples/icooa` 上继续兼容式扩展，也不把 AI / Agent 做成语言内置能力。新的主线是一个纯 Icoo 项目 `apps/agent`：会话、上下文、工具、模型、计划、patch、审批全部在 Icoo 层实现；Go 侧只保留现有通用执行能力，例如 `icoo run <project>`、`icoo check <project>`、`icoo build <project>`，不为 Agent 单独堆一个宿主编排层。若 Agent 需要更强能力，应反哺 `icoo_lang` 本身，而不是回退到写专用 Go 逻辑。

**Tech Stack:** `icoo_lang` project system, pure Icoo app under `apps/agent`, existing `icoo run/check/build`, OpenAI-compatible HTTP APIs, JSON session/event store, filesystem and process stdlib, targeted runtime/stdlib improvements when blocked.

---

## 1. 核心约束

- Agent 主体必须使用纯 `icoo_lang` 开发
- 不新增 Go 版 `cmd/icoo/agent.go` 作为主实现
- 不搞 Go 宿主编排 + Icoo 脚本业务的双栈分裂
- 不把 AI / Agent 直接塞进语言内置能力
- 遇到能力缺口，先判断是否应补到：
  - 通用 CLI 机制
  - 通用 runtime 能力
  - 通用标准库能力
  - 通用项目结构约定
- 不继续把 `examples/icooa` 当主线演进基础

## 2. 方向判断

这次主线要解决的不只是“Agent 怎么写”，而是：

- `icoo_lang` 是否足以支撑一个中等复杂度的真实应用
- 如果不够，缺的是哪类通用能力
- 这些缺口是否值得被沉淀为语言、运行时、标准库增强

因此，`apps/agent` 不只是一个功能项目，也是 `icoo_lang` 的压力测试项目。

如果用 Go 承接大量 Agent 编排逻辑，会掩盖真实能力缺口，最后得不出对语言本身有价值的结论。这条路不符合当前目标。

## 3. 新主线产物

重构完成后，应存在以下主线产物：

1. `apps/agent/`
   - 纯 Icoo Agent 工程
   - 具备自己的 `project.toml`、`app.ic`、`src/*`、`smoke.ic`

2. `icoo run apps/agent`
   - 作为 Agent 主启动方式
   - 通过参数和环境变量进入交互模式或批处理模式

3. `docs/agent/`
   - 使用说明
   - 架构说明
   - session/event 格式
   - 能力缺口与语言增强记录

4. `examples/icooa/`
   - 降级为 legacy example
   - 明确不再作为主线基础

## 4. 主架构

### 4.1 顶层分层

1. `App Entry Layer`
   - 启动配置
   - 参数读取
   - 模式选择
   - 输出渲染

2. `Runtime Layer`
   - 会话生命周期
   - 回合循环
   - 计划 / 执行状态机
   - 错误恢复

3. `Context Layer`
   - 工作区扫描
   - 文件选择
   - 截断预算
   - 任务相关性排序

4. `Tool Layer`
   - 文件工具
   - 搜索工具
   - 命令工具
   - patch 工具

5. `Model Layer`
   - 消息组装
   - 远端请求
   - 响应解析
   - 工具调用协议

6. `State Layer`
   - session
   - turns
   - events
   - artifacts
   - approvals

### 4.2 目录建议

- `apps/agent/project.toml`
- `apps/agent/app.ic`
- `apps/agent/src/main.ic`
- `apps/agent/src/config/`
- `apps/agent/src/session/`
- `apps/agent/src/context/`
- `apps/agent/src/tools/`
- `apps/agent/src/model/`
- `apps/agent/src/runtime/`
- `apps/agent/src/render/`
- `apps/agent/src/testing/`
- `apps/agent/smoke.ic`

### 4.3 import 风格建议

统一使用项目根别名导入，例如：

```icoo
import "@/src/config/defaults.ic" as defaults
import "@/src/session/store.ic" as sessionStore
import "@/src/runtime/loop.ic" as loop
```

不再沿用 `examples/icooa/src/models/*`、`services/*` 的旧拆分。

## 5. 明确放弃的旧路径

以下路径不再作为主线：

- 继续围绕 `examples/icooa` 增量修补
- 为 Agent 单独写 Go 宿主命令编排层
- 用 Go 持有 session / tool / approval 主状态
- 在主线早期就把 Agent 做成语言内置能力

处理原则：

- `examples/icooa` 保留为 legacy
- 新 Agent 不要求兼容旧 session 结构
- 迁移靠文档，不靠兼容层

## 6. 启动与运行方式

### 6.1 主入口

主入口应基于现有 CLI：

```bash
icoo run apps/agent
icoo run apps/agent -- --workspace . --task "summarize repo"
icoo run apps/agent -- --resume demo
```

### 6.2 模式

- 单轮批处理模式
- 持久化多轮模式
- 交互 REPL 风格模式
- smoke / test 模式

### 6.3 后续可评估的 CLI 增强

如果纯 Icoo Agent 在现有 `icoo run` 入口上存在明显摩擦，可以增强通用 CLI，而不是写 Agent 专用 Go 编排层。例如：

- 更好的 script args 透传
- 更稳定的交互 stdin 支持
- 更清晰的项目入口约定
- 更好的中断和退出码语义

这些增强应服务所有 Icoo 应用，而不是只服务 Agent。

## 7. 配置系统

建议统一使用 `ICOO_AGENT_*` 命名空间。

至少包括：

- `ICOO_AGENT_WORKSPACE`
- `ICOO_AGENT_MODEL`
- `ICOO_AGENT_BASE_URL`
- `ICOO_AGENT_API_KEY`
- `ICOO_AGENT_SESSION_DIR`
- `ICOO_AGENT_APPROVAL`
- `ICOO_AGENT_MAX_TURNS`
- `ICOO_AGENT_MAX_FILES`
- `ICOO_AGENT_MAX_TOTAL_BYTES`
- `ICOO_AGENT_LOG_PATH`
- `ICOO_AGENT_MODE`

配置优先级：

1. script args
2. env
3. defaults

如果后续确实需要配置文件，也应先在 Icoo 应用层实现，不急着扩 Go CLI。

## 8. Session / Turn / Event 模型

旧 `icooa` 的问题是消息数组驱动一切。新主线必须改成显式状态模型。

### 8.1 Session

```json
{
  "sessionId": "s_20260507_xxx",
  "workspace": "E:/code/issueye/icoo_lang",
  "mode": "interactive",
  "status": "idle",
  "createdAt": "...",
  "updatedAt": "...",
  "config": {},
  "turns": [],
  "events": [],
  "artifacts": [],
  "budgets": {},
  "approvals": []
}
```

### 8.2 Turn

```json
{
  "id": "turn_001",
  "user": {},
  "analysis": {},
  "plan": {},
  "toolCalls": [],
  "assistant": {},
  "status": "completed"
}
```

### 8.3 Event

```json
{
  "ts": "...",
  "type": "tool_call_started",
  "turnId": "turn_001",
  "payload": {}
}
```

## 9. 工具系统

工具系统必须完全在 Icoo 层实现。

### 9.1 首批工具

- `readFile`
- `listDir`
- `searchText`
- `runCommand`
- `writePatch`
- `applyPatch`

### 9.2 设计要求

- 每个工具有明确 schema
- 每个工具有风险级别
- 每个工具的结果格式统一
- 工具调用要可落盘、可重放、可审计

### 9.3 若标准库不够怎么办

如果 Icoo 层实现这些工具时暴露出缺口，优先补：

- `std.fs`
- `std.exec`
- `std.os`
- `std.io`
- `std.net.http.client`
- `std.core.object` / `std.core.string` / `std.data.json`

而不是把缺口直接转移到 Go 宿主实现。

## 10. 模型交互

### 10.1 基本原则

- 模型调用逻辑写在 Icoo 应用层
- 不引入 `std.ai.*`
- 不引入语言内置 Agent 原语

### 10.2 第一阶段协议

- 先支持 OpenAI-compatible `/v1/chat/completions`

### 10.3 输出结构

建议统一解析为：

```json
{
  "analysis": "...",
  "plan": ["..."],
  "tool_calls": [],
  "final": "..."
}
```

## 11. 上下文系统

新的上下文系统必须是预算驱动的，而不是目录遍历文本拼接。

### 11.1 必须支持

- 忽略规则
- 文件类型识别
- 二进制跳过
- 大文件截断
- 入口文件优先
- README / config 优先
- 与任务关键词相关的文件优先

### 11.2 若实现受阻

若 Icoo 写这部分出现性能或表达力问题，应优先定位真实瓶颈：

- 是 `std.fs.readDir` / `stat` 不够顺手
- 是字符串处理不够方便
- 是 JSON / object 操作成本太高
- 是脚本参数和项目上下文注入不够好

结论应反馈到语言和标准库，而不是直接改成 Go 实现。

## 12. 审批与 patch

审批与 patch 也应在 Icoo 层完成主逻辑。

### 12.1 审批模式

- `never`
- `on-request`
- `on-failure`
- `always`

### 12.2 默认策略

- 默认禁止写操作
- 默认禁止危险命令
- 默认只自动执行只读工具

### 12.3 若交互能力不足

若纯 Icoo 在交互确认上存在明显问题，应增强通用 CLI / runtime 对 stdin/stdout 的支持，不应引入 Agent 专用 Go 控制器。

## 13. 语言与标准库反哺原则

这是本次主线最重要的约束之一。

当 `apps/agent` 开发受阻时，先问：

1. 这是 Agent 专属问题，还是通用应用开发问题？
2. 如果这是通用问题，应该补在语言、runtime 还是标准库？
3. 这个增强是否能被其他 Icoo 项目复用？

优先反哺的方向：

- CLI 参数和项目运行体验
- 更稳定的 stdin/stdout 支持
- 文件系统与进程调用能力
- JSON / object / string 处理能力
- 日志、事件、时间能力

避免的错误方向：

- 因为 Agent 有需求，就写一个 Agent 专属 Go 层
- 因为当前不好写，就把主状态迁回 Go
- 因为想省事，就把 AI 直接做成内置语法或内置库

## 14. 路线图

### Phase 0: 旧路径退出主线

交付：

- `examples/icooa` 标记为 legacy
- 文档明确 `apps/agent` 是主线

### Phase 1: 纯 Icoo Agent 骨架

交付：

- 建立 `apps/agent`
- `icoo run apps/agent` 可启动
- 能输出最小欢迎信息

### Phase 2: Session / Context / Model 基础层

交付：

- session store
- turn/event schema
- context builder
- model client

### Phase 3: Tool / Loop / Multi-turn

交付：

- tool registry
- tool executor
- agent loop
- resume / continue

### Phase 4: Approval / Patch / Stability

交付：

- approval policy
- patch artifact
- dry-run / apply
- smoke / recovery

### Phase 5: 反哺语言与标准库

交付：

- 把开发中遇到的真实阻塞整理为通用改进项
- 逐项补到 `icoo_lang`

## 15. 详细任务计划

### Task 1: 将 `examples/icooa` 降级为 legacy example

**Files:**
- Modify: `examples/icooa/README.md`
- Create: `docs/agent/migration-from-icooa.md`

**Step 1: 更新 README**

- 明确：
  - 这是 legacy example
  - 不再作为 Agent 主线
  - 新主线迁移到 `apps/agent`

**Step 2: 写迁移说明**

- 说明旧入口与新入口
- 说明不兼容项

**Step 3: Commit**

```bash
git add examples/icooa/README.md docs/agent/migration-from-icooa.md
git commit -m "docs: mark icooa as legacy example"
```

### Task 2: 创建纯 Icoo Agent 工程骨架

**Files:**
- Create: `apps/agent/project.toml`
- Create: `apps/agent/app.ic`
- Create: `apps/agent/src/main.ic`
- Create: `apps/agent/src/config/defaults.ic`
- Create: `apps/agent/src/runtime/runtime.ic`

**Step 1: 建立项目入口**

- 让 `icoo run apps/agent` 可启动

**Step 2: 输出最小欢迎信息**

- workspace
- mode
- sessionDir

**Step 3: 验证**

Run: `go run ./cmd/icoo run apps/agent`
Expected: 输出 agent app 启动信息

**Step 4: Commit**

```bash
git add apps/agent/project.toml apps/agent/app.ic apps/agent/src/main.ic apps/agent/src/config/defaults.ic apps/agent/src/runtime/runtime.ic
git commit -m "feat: scaffold pure icoo agent app"
```

### Task 3: 重建配置系统

**Files:**
- Create: `apps/agent/src/config/env.ic`
- Create: `apps/agent/src/config/args.ic`
- Create: `apps/agent/src/config/merge.ic`
- Modify: `apps/agent/src/main.ic`

**Step 1: 统一 `ICOO_AGENT_*`**

- 仅支持新命名空间

**Step 2: 基于 script args + env 合并**

- 不引入 Go 层专用参数解析

**Step 3: 验证**

Run: `go run ./cmd/icoo run apps/agent -- --workspace . --mode batch`
Expected: 参数覆盖生效

**Step 4: Commit**

```bash
git add apps/agent/src/config/env.ic apps/agent/src/config/args.ic apps/agent/src/config/merge.ic apps/agent/src/main.ic
git commit -m "feat: add pure icoo agent config system"
```

### Task 4: 重建 session / turn / event store

**Files:**
- Create: `apps/agent/src/session/store.ic`
- Create: `apps/agent/src/session/schema.ic`
- Create: `apps/agent/src/session/events.ic`
- Create: `apps/agent/src/session/turns.ic`

**Step 1: 定义新 schema**

- session
- turn
- event
- artifacts

**Step 2: 实现保存与恢复**

- 正常恢复
- 损坏文件报错
- 新旧格式不兼容时明确拒绝

**Step 3: 验证**

Run: `go run ./cmd/icoo run apps/agent`
Expected: 能创建新 session 文件

**Step 4: Commit**

```bash
git add apps/agent/src/session/store.ic apps/agent/src/session/schema.ic apps/agent/src/session/events.ic apps/agent/src/session/turns.ic
git commit -m "feat: add pure icoo agent session model"
```

### Task 5: 重建上下文构建器

**Files:**
- Create: `apps/agent/src/context/builder.ic`
- Create: `apps/agent/src/context/budget.ic`
- Create: `apps/agent/src/context/ignore.ic`
- Create: `apps/agent/src/context/ranker.ic`

**Step 1: 实现预算驱动扫描**

- 文件数
- 总字节
- 单文件字节

**Step 2: 实现优先级**

- README / config / entry / task match

**Step 3: 验证**

- 在真实仓库上输出结构化上下文

**Step 4: Commit**

```bash
git add apps/agent/src/context/builder.ic apps/agent/src/context/budget.ic apps/agent/src/context/ignore.ic apps/agent/src/context/ranker.ic
git commit -m "feat: add pure icoo agent context builder"
```

### Task 6: 重建模型客户端

**Files:**
- Create: `apps/agent/src/model/client.ic`
- Create: `apps/agent/src/model/messages.ic`
- Create: `apps/agent/src/model/parser.ic`

**Step 1: 纯 Icoo 实现模型调用**

- 使用 `std.net.http.client`

**Step 2: 结构化解析**

- `analysis`
- `plan`
- `tool_calls`
- `final`

**Step 3: 验证**

- mock model server 可跑通一轮请求

**Step 4: Commit**

```bash
git add apps/agent/src/model/client.ic apps/agent/src/model/messages.ic apps/agent/src/model/parser.ic
git commit -m "feat: add pure icoo agent model client"
```

### Task 7: 重建工具系统

**Files:**
- Create: `apps/agent/src/tools/registry.ic`
- Create: `apps/agent/src/tools/executor.ic`
- Create: `apps/agent/src/tools/read_file.ic`
- Create: `apps/agent/src/tools/list_dir.ic`
- Create: `apps/agent/src/tools/search_text.ic`
- Create: `apps/agent/src/tools/run_command.ic`

**Step 1: 定义工具 schema**

- name
- risk
- approval
- input schema

**Step 2: 优先完成只读工具**

- read / list / search

**Step 3: 再实现命令工具**

- 受审批策略控制

**Step 4: 验证**

- 能完成“搜索 -> 读文件 -> 输出”

**Step 5: Commit**

```bash
git add apps/agent/src/tools/registry.ic apps/agent/src/tools/executor.ic apps/agent/src/tools/read_file.ic apps/agent/src/tools/list_dir.ic apps/agent/src/tools/search_text.ic apps/agent/src/tools/run_command.ic
git commit -m "feat: add pure icoo agent tools"
```

### Task 8: 建立真正的 agent loop

**Files:**
- Create: `apps/agent/src/runtime/loop.ic`
- Create: `apps/agent/src/runtime/planner.ic`
- Create: `apps/agent/src/runtime/turn_runner.ic`
- Modify: `apps/agent/src/runtime/runtime.ic`

**Step 1: 定义每轮流程**

- user input
- context build
- model call
- tool execution
- final answer

**Step 2: 区分 plan 与 execute**

- 简单问题跳过工具
- 复杂问题进入工具回路

**Step 3: 验证**

- 能完成至少一次带工具调用的多步回合

**Step 4: Commit**

```bash
git add apps/agent/src/runtime/loop.ic apps/agent/src/runtime/planner.ic apps/agent/src/runtime/turn_runner.ic apps/agent/src/runtime/runtime.ic
git commit -m "feat: add pure icoo agent loop"
```

### Task 9: 建立审批与 patch 机制

**Files:**
- Create: `apps/agent/src/runtime/approval.ic`
- Create: `apps/agent/src/tools/write_patch.ic`
- Create: `apps/agent/src/tools/apply_patch.ic`
- Create: `apps/agent/src/session/artifacts.ic`

**Step 1: 审批规则**

- 只读自动
- 写操作确认
- 危险命令拒绝

**Step 2: patch 流程**

- 先生成 artifact
- 再预览
- 再确认应用

**Step 3: 验证**

- dry-run 成功
- 拒绝路径正确

**Step 4: Commit**

```bash
git add apps/agent/src/runtime/approval.ic apps/agent/src/tools/write_patch.ic apps/agent/src/tools/apply_patch.ic apps/agent/src/session/artifacts.ic
git commit -m "feat: add pure icoo agent approval and patch flow"
```

### Task 10: 把真实阻塞反哺到 `icoo_lang`

**Files:**
- Modify: `docs/mvp-status.md`
- Create: `docs/agent/language-gaps.md`
- Modify: relevant runtime/stdlib/docs files as needed

**Step 1: 记录阻塞**

- 区分：
  - 表达力不足
  - 标准库不足
  - CLI 运行体验不足
  - 性能问题

**Step 2: 只补通用能力**

- 不补 Agent 专用 Go 编排

**Step 3: 验证**

- `apps/agent` 因这些增强而变得更可实现

**Step 4: Commit**

```bash
git add docs/mvp-status.md docs/agent/language-gaps.md
git commit -m "docs: capture pure icoo agent pressure-test gaps"
```

### Task 11: 完成 smoke 与文档

**Files:**
- Create: `apps/agent/smoke.ic`
- Create: `docs/agent/overview.md`
- Create: `docs/agent/usage.md`
- Create: `docs/agent/session-format.md`
- Create: `docs/agent/pure-icoo-rationale.md`

**Step 1: 集成 smoke**

- mock model server
- temp workspace
- multi-turn
- approval path

**Step 2: 文档**

- 使用
- 架构
- 纯 Icoo 理由
- 不兼容说明

**Step 3: 验证**

Run: `go run ./cmd/icoo run apps/agent/smoke.ic`
Expected: 输出 smoke success

**Step 4: Commit**

```bash
git add apps/agent/smoke.ic docs/agent/overview.md docs/agent/usage.md docs/agent/session-format.md docs/agent/pure-icoo-rationale.md
git commit -m "test: add pure icoo agent smoke and docs"
```

## 16. 风险

- 纯 Icoo 开发会更早暴露语言与标准库短板，短期推进速度可能慢于 Go 混合方案
- 如果团队在遇到阻塞时频繁回退到 Go 实现，主线目标会失真
- 如果不控制边界，容易把“为了支持 Agent”演变成“为了一个项目污染语言设计”
- 因此每次增强都必须证明其通用价值

## 17. 验收标准

达到当前主线目标时，至少满足：

- `apps/agent` 可由 `icoo run apps/agent` 启动
- Agent 的 session/context/model/tool/runtime 主逻辑全部由 Icoo 实现
- 遇到缺口时，有明确反哺记录和对应语言/标准库增强
- `examples/icooa` 已明确退出主线
- 没有新增 Agent 专属 Go 编排层

## 18. 推荐执行顺序

必须按下面顺序推进：

1. 先把 `icooa` 降级为 legacy
2. 再建 `apps/agent` 纯 Icoo 骨架
3. 再建 config/session/context/model
4. 再建 tools/loop
5. 再建 approval/patch
6. 最后整理语言缺口与文档

如果中途发现必须补 Go 逻辑，先暂停并判断它是否属于通用 `icoo_lang` 能力；不是通用能力，就不应进入主线。
