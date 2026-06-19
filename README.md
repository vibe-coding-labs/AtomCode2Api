# AtomCodeProxy

将 AtomCode Daemon 的 CodingPlan 免费额度以 OpenAI/Anthropic 兼容 API 形式暴露。

## 架构

```
外部工具 (Claude Code / Cursor / Cline)
  │ POST /v1/chat/completions (OpenAI) 或 POST /v1/messages (Anthropic)
  ▼
AtomCodeProxy (:13457)
  │ POST /chat (SSE stream)
  ▼
AtomCode Daemon (:13456)
  │ → llm-api.atomgit.com (OpenAI-compatible)
  ▼
AI Models (deepseek-v4-flash, etc.)
```

## 快速开始

```bash
# 1. 确保 AtomCode daemon 在运行
atomcode daemon

# 2. 首次使用：登录 + 领取 CodingPlan
atomcode-proxy setup

# 3. 启动代理
atomcode-proxy serve

# 4. 配置 Claude Code
export ANTHROPIC_BASE_URL=http://localhost:13457
export ANTHROPIC_API_KEY=atomcode
claude
```

## 守护进程模式

```bash
# 后台运行（崩溃自动重启）
atomcode-proxy daemon start

# 查看状态
atomcode-proxy daemon status

# 停止
atomcode-proxy daemon stop
```

## API 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `POST /v1/chat/completions` | OpenAI | Chat Completions API |
| `POST /v1/messages` | Anthropic | Messages API (Claude Code) |
| `GET /v1/models` | - | 模型列表 |
| `GET /health` | - | 健康检查 |

## 构建

```bash
go build -o atomcode-proxy ./cmd/atomcode-proxy/
```