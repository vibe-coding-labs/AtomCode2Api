# AtomCodeProxy

**A reverse proxy that translates AtomCode Daemon's private API into standard OpenAI/Anthropic APIs. Turn your free CodingPlan quota into a universal AI endpoint.**

[English](#english) | [简体中文](#简体中文)

---

## English

> AtomCodeProxy is an engineering implementation based on reverse-engineered analysis of the [AtomCode](https://atomgit.com/atomgit_atomcode/atomcode) daemon protocol. It acts as a protocol translation layer between your favorite AI tools (Claude Code, Cursor, Cline) and the AtomCode local daemon, exposing its CodingPlan free-tier AI quota through standard OpenAI and Anthropic-compatible APIs.

### Problem

AtomCode (a Chinese AI coding assistant) provides a **CodingPlan Lite** free tier with generous monthly quotas:
- **deepseek-v4-flash** model with **1M context** tokens
- **8,000 requests/month** (≈267/day), **500 requests per 5-hour window**

However, AtomCode uses its own private daemon protocol (`localhost:13456`). Mainstream AI coding tools like **Claude Code**, **Cursor**, and **Cline** only support standard OpenAI Chat Completions or Anthropic Messages APIs — they cannot directly connect to AtomCode.

**AtomCodeProxy solves this** by sitting between your tools and the daemon, translating requests and responses in real-time.

### Architecture

```
Your AI Tool (Claude Code / Cursor / Cline)
  │ POST /v1/chat/completions (OpenAI) or POST /v1/messages (Anthropic)
  ▼
AtomCodeProxy (:13457) ─── protocol translation layer
  │ POST /chat (SSE stream)
  ▼
AtomCode Daemon (:13456) ─── private daemon protocol
  │ → llm-api.atomgit.com (upstream)
  ▼
AI Models (deepseek-v4-flash, etc.)
```

### Features

| Feature | Details |
|---------|---------|
| **Dual Protocol** | OpenAI Chat Completions (`/v1/chat/completions`) + Anthropic Messages (`/v1/messages`) |
| **Streaming** | SSE streaming for both protocols, real-time token-by-token output |
| **Tool Calling** | Full tool_use/tool_call translation — file operations, commands, etc. |
| **Reasoning** | Maps daemon `reasoning` events to OpenAI `reasoning_content` / Anthropic `thinking` blocks |
| **Session Persistence** | Multi-turn conversation context tracked via daemon `session_id` (30-minute TTL) |
| **Daemon Mode** | Background supervisor with automatic crash restart (exponential backoff: 1s→2s→4s→...→30s) |
| **Health Monitoring** | Auto-detects daemon status, logged-in state, and model availability |
| **Request Logging** | SQLite-backed request history with model, latency, token usage |
| **TLS Support** | Auto-generated self-signed certificate for HTTPS |
| **Log Rotation** | Automatic log rotation with configurable size limits and retention |
| **Docker** | Multi-stage Dockerfile for containerized deployment |

### Quick Start

```bash
# 1. Make sure AtomCode daemon is running
atomcode daemon

# 2. First time: login & claim CodingPlan
atomcode-proxy setup

# 3. Start the proxy
atomcode-proxy serve

# 4. Configure Claude Code
export ANTHROPIC_BASE_URL=http://localhost:13457
export ANTHROPIC_API_KEY=atomcode
claude
```

### Daemon Mode (Background Service)

```bash
# Start as background daemon (auto-restart on crash)
atomcode-proxy daemon start

# Check status
atomcode-proxy daemon status

# View logs (last N lines)
atomcode-proxy daemon logs

# Stop
atomcode-proxy daemon stop
```

### API Endpoints

| Path | Method | Protocol | Description |
|------|--------|----------|-------------|
| `POST /v1/chat/completions` | OpenAI | Chat Completions API | For Cursor, Cline, etc. |
| `POST /v1/messages` | Anthropic | Messages API | For Claude Code |
| `GET /v1/models` | - | Model list | Available models from daemon |
| `GET /health` | - | Health check | Daemon + auth status |

### CLI Commands

```
atomcode-proxy serve         Start proxy server
atomcode-proxy setup         Login + CodingPlan claim
atomcode-proxy login         OAuth login only
atomcode-proxy daemon        Background service management
atomcode-proxy models        List available models
atomcode-proxy status        Show daemon/proxy status
atomcode-proxy whoami        Show current logged-in user
atomcode-proxy check         Full health check
atomcode-proxy config show   Show configuration
```

### Build

```bash
go build -o atomcode-proxy ./cmd/atomcode-proxy/
```

### Project Structure

```
cmd/atomcode-proxy/    CLI entry (cobra commands)
pkg/atmc/              AtomCode daemon HTTP client + SSE translator
pkg/openai/            OpenAI protocol handler
pkg/anthropic/         Anthropic protocol handler
pkg/store/             SQLite storage (accounts, settings, logs)
pkg/auth/              JWT auth + encryption
pkg/logrot/            Log rotation
pkg/keepalive/         Credential keepalive
pkg/dashboard/         Web dashboard API (planned)
pkg/proxy/             Session tracking
```

### License

Apache 2.0

---

## 简体中文

> AtomCodeProxy 是基于 [AtomCode](https://atomgit.com/atomgit_atomcode/atomcode) Daemon 协议逆向分析的工程化实现。它在你的 AI 工具和 AtomCode 本地 daemon 之间充当协议翻译层，将 AtomCode 的 CodingPlan 免费 AI 额度通过标准的 OpenAI 和 Anthropic 兼容 API 暴露出来。

### 问题

AtomCode（一个国产 AI 编程助手）提供了 **CodingPlan Lite** 免费套餐，包含不错的额度：
- **deepseek-v4-flash** 模型，**1M 上下文** tokens
- 每月 **8,000 次请求**（日均 ~267 次），每 5 小时 **500 次** 短期窗口

然而 AtomCode 使用的是自己的私有 daemon 协议（`localhost:13456`）。主流的 AI 编程工具如 **Claude Code**、**Cursor**、**Cline** 只支持标准的 OpenAI Chat Completions 或 Anthropic Messages API — 它们无法直连 AtomCode。

**AtomCodeProxy 解决了这个问题**：它充当工具和 daemon 之间的翻译层，实时转换请求和响应。

### 架构

```
AI 工具 (Claude Code / Cursor / Cline)
  │ POST /v1/chat/completions (OpenAI) 或 POST /v1/messages (Anthropic)
  ▼
AtomCodeProxy (:13457) ─── 协议翻译层
  │ POST /chat (SSE 流)
  ▼
AtomCode Daemon (:13456) ─── 私有 daemon 协议
  │ → llm-api.atomgit.com (上游 API)
  ▼
AI 模型 (deepseek-v4-flash 等)
```

### 功能特性

| 特性 | 说明 |
|------|------|
| **双协议** | OpenAI Chat Completions + Anthropic Messages，一套引擎双出口 |
| **流式输出** | SSE 实时流式返回，打字机效果 |
| **工具调用** | 完整的 tool_use/tool_call 翻译，文件操作、命令执行等 |
| **推理过程** | daemon reasoning → OpenAI reasoning_content / Anthropic thinking |
| **多轮对话** | 基于 daemon session_id 的上下文保持（30 分钟过期） |
| **守护进程** | 后台运行 + 崩溃自动重启（指数退避：1s→2s→4s→...→30s） |
| **健康监控** | 自动检测 daemon 状态、登录状态、模型可用性 |
| **请求日志** | SQLite 记录请求历史（模型、延迟、Token 用量）|
| **TLS 支持** | 自动生成自签名证书，支持 HTTPS |
| **日志轮转** | 自动日志轮转，可配置大小和保留数量 |
| **Docker 部署** | 多阶段构建 Dockerfile |

### 快速开始

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

### 配置 Cursor（OpenAI 协议）

在 Cursor Settings → Models 中：
- **API Base URL**: `http://localhost:13457/v1`
- **API Key**: 任意值（可以为空）
- **Model**: `deepseek-v4-flash`

### CLI 命令

```
atomcode-proxy serve         启动代理服务器
atomcode-proxy setup         登录 + 领取 CodingPlan
atomcode-proxy daemon        守护进程管理
atomcode-proxy models        列出可用模型
atomcode-proxy status        查看状态
atomcode-proxy check         全面健康检查
atomcode-proxy whoami        当前用户信息
atomcode-proxy config show   查看配置
```

### 项目结构

```
cmd/atomcode-proxy/    CLI 入口（cobra 命令）
pkg/atmc/              AtomCode daemon HTTP 客户端 + SSE 翻译器
pkg/openai/            OpenAI 协议层
pkg/anthropic/         Anthropic 协议层
pkg/store/             SQLite 存储（账号、设置、日志）
pkg/auth/              JWT 认证 + 加密
pkg/logrot/            日志轮转
pkg/keepalive/         凭据保活
pkg/dashboard/         Web 管理面板 API（规划中）
pkg/proxy/             会话追踪
```

### 许可证

Apache 2.0
