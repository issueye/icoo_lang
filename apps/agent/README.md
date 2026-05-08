# icoo-agent

`apps/agent` 支持两种运行模式：

- `interactive`：命令行输出结果
- `server`：启动 HTTP 服务

当前仅从 `config.toml` 读取配置，不再读取命令行参数、环境变量中的业务配置。

## Server 模式

示例 `config.toml`：

```toml
workspace = "./runtime"
mode = "server"
model = "gpt-4.1-mini"
base_url = "https://api.openai.com/v1"
api_key = "YOUR_API_KEY"
approval = "on-request"
max_turns = 12
max_files = 24
max_total_bytes = 32768
stream_final_answer = true
agent_name = "icoo_agents"

server_host = "127.0.0.1"
server_port = 8080
server_read_timeout_ms = 0
server_read_header_timeout_ms = 0
server_write_timeout_ms = 0
server_idle_timeout_ms = 0

session_dir = "./runtime/.agents/sessions"
log_path = "./runtime/.agents/agent.log"
```

超时字段会透传给 `std.web.express.listen(...)`：

- `server_read_timeout_ms`
- `server_read_header_timeout_ms`
- `server_write_timeout_ms`
- `server_idle_timeout_ms`

启动：

```powershell
cd apps/agent
.\build.ps1 -RepoRoot E:\code\issueye\icoo_lang -SkipVerify
cd .\dist\icoo-agent
.\icoo-agent.exe
```

## API

### `GET /healthz`

返回服务健康状态。

示例响应：

```json
{
  "ok": true,
  "agent": "icoo_agents",
  "mode": "server",
  "model": "gpt-4.1-mini"
}
```

### `POST /chat`

同步返回完整结果。

示例请求：

```json
{
  "task": "今天成都的天气",
  "model": "gpt-4.1-mini"
}
```

示例响应：

```json
{
  "ok": true,
  "sessionId": "s_20260508-000000",
  "sessionPath": "E:/path/to/session.json",
  "resumed": false,
  "mode": "remote",
  "text": "......",
  "reasoningContent": null,
  "toolRuns": [],
  "turnsUsed": 1,
  "stopReason": "assistant response completed",
  "contextSummary": {
    "fileCount": 3
  }
}
```

### `POST /stream_chat`

使用 `text/event-stream` 返回 SSE 事件。

事件顺序：

- `started`
- `delta`
- `completed`
- `done`

如果执行失败，会返回：

- `error`

`delta` 事件示例：

```text
event: delta
data: {"text":"hello ","fullText":"hello ","reasoningContent":"","finishReason":null}
```

完成事件示例：

```text
event: completed
data: {"ok":true,"sessionId":"s_20260508-000000","text":"hello world"}
```
