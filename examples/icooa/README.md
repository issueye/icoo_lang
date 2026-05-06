# Icooa 示例项目

`examples/icooa` 是一个面向代码仓库的一次性代理示例项目，参考了 Claude Code 的核心工作流，但刻意收敛到当前 `icoo_lang` 已具备的能力边界。

它不是完整的交互式终端代理，也不是多轮会话编排器；当前版本重点展示下面四件事：

1. 读取工作区并生成仓库快照
2. 在宿主机上执行一个可选命令
3. 调用 OpenAI 兼容 `/v1/chat/completions` 接口生成代码建议
4. 把请求上下文、命令结果和模型输出保存成会话 JSON

现在还额外支持“持久化多轮”模式：

5. 使用固定 `sessionId` 连续运行时，把上一轮 user/assistant 消息继续带入下一轮请求

## 目录

- `app.ic`
  - 项目入口
- `src/main.ic`
  - 对外暴露 `run` / `main`
- `src/models/config_model.ic`
  - 默认配置、环境变量覆盖
- `src/models/workspace_model.ic`
  - 工作区扫描与上下文压缩
- `src/models/session_model.ic`
  - 会话落盘
- `src/services/agent_service.ic`
  - 命令执行、模型调用、日志和输出编排
- `smoke.ic`
  - 本地 mock server 冒烟脚本

## 环境变量

- `ICOOA_WORKSPACE`
  - 要扫描的工作目录，默认当前目录
- `ICOOA_TASK`
  - 本次任务描述
- `ICOOA_COMMAND`
  - 任务前要执行的宿主命令，例如 `git status`
- `ICOOA_MODEL`
  - 模型名，默认 `gpt-4.1-mini`
- `ICOOA_BASE_URL`
  - OpenAI 兼容服务地址，默认 `https://api.openai.com/v1`
- `OPENAI_API_KEY` / `ICOOA_API_KEY`
  - API Key
- `ICOOA_SESSION_DIR`
  - 会话保存目录
- `ICOOA_SESSION_ID`
  - 指定会话 ID；相同 ID 会写入同一个会话文件
- `ICOOA_CONTINUE`
  - 为 `true/1` 时，继续读取并复用已有会话历史
- `ICOOA_LOG_PATH`
  - 日志路径
- `ICOOA_MAX_ENTRIES`
  - 最多采样多少个文件
- `ICOOA_MAX_DEPTH`
  - 最深扫描目录层级
- `ICOOA_MAX_FILE_BYTES`
  - 单文件最大采样字节数
- `ICOOA_INCLUDE_HIDDEN`
  - 是否包含点文件

## 运行

也可以直接传脚本参数，不再只能依赖环境变量：

```powershell
go run ./cmd/icoo run examples/icooa -- --workspace E:/codes/icoo_lang --task "Review the runtime and propose the next step." --command "git status --short"
```

不带远端模型时，它会退化成“本地上下文采样 + fallback 提示”，仍然可以跑通：

```powershell
$env:ICOOA_TASK = "Summarize this repo and propose the next implementation step."
go run ./cmd/icoo run examples/icooa
```

带远端模型时：

```powershell
$env:OPENAI_API_KEY = "sk-..."
$env:ICOOA_WORKSPACE = "E:/codes/icoo_lang"
$env:ICOOA_TASK = "Review the runtime and propose a safe refactor plan."
$env:ICOOA_COMMAND = "git status --short"
go run ./cmd/icoo run examples/icooa
```

执行后会输出：

- 工作目录
- 当前任务
- 会话 JSON 保存路径
- 命令执行结果摘要
- 模型或 fallback 的代码建议

默认还会把结构化日志写到 `.icooa/icooa.log`，并做基础日志切割。

## 多轮继续

第一轮：

```powershell
go run ./cmd/icoo run examples/icooa -- --session demo --workspace E:/codes/icoo_lang --task "Summarize the runtime."
```

继续同一个会话：

```powershell
go run ./cmd/icoo run examples/icooa -- --session demo --continue --workspace E:/codes/icoo_lang --task "Now focus on the most important files and concrete next edits."
```

继续模式下，`icooa` 会：

- 读取 `sessions/demo.json`
- 取回之前保存的 user / assistant 消息
- 把这些历史消息拼进下一次 `/v1/chat/completions` 请求
- 用同一个会话文件覆盖保存最新轮次

## 冒烟

```powershell
go run ./cmd/icoo run examples/icooa/smoke.ic
```

这个脚本会：

- 创建临时工作区
- 启动一个本地 mock `/v1/chat/completions` 服务
- 调用 `icooa.run(...)`
- 校验远端模式输出和会话落盘

## 当前边界

这个示例当前明确没有覆盖：

- 交互式 stdin 多轮对话
- 流式 token 输出
- 文件修改 / patch 自动应用
- 真正的工具调用协议
- 宿主命令的细粒度沙箱

因此它更准确的定位是：

- `icoo_lang` 的“代码代理雏形”示例
- 标准库能力组合示例
- 未来代理式 CLI / IDE 集成的设计探针
