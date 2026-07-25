# AtomCode2API

> **将 AtomCode Daemon 的私有协议转换为标准的 OpenAI / Anthropic 兼容 API。**
> 把你的 CodingPlan 免费额度变成通用的 AI 接口，让 Claude Code、Codex、Cursor、Cline 等工具都能用上。
>
> _A reverse proxy that translates AtomCode Daemon's private API into standard OpenAI/Anthropic-compatible APIs. Turn your free CodingPlan quota into a universal AI endpoint._

---

## 解决的问题 / Problem

[AtomCode](https://atomcode.atomgit.com/) 是一个国产 AI 编程助手，提供 **CodingPlan Lite** 免费套餐，包含 **deepseek-v4-flash** 模型（1M 上下文窗口）和每月数千次免费请求。

但 AtomCode 使用**私有 daemon 协议**（`localhost:13456`），主流的 AI 工具（Claude Code、Codex、Cursor、Cline）只支持标准 OpenAI/Anthropic API，无法直接连接 AtomCode。

**AtomCode2API 充当翻译层**，让你的工具通过标准协议使用 AtomCode 的免费额度。

## 架构 / Architecture

```
AI 工具 (Claude Code / Codex / Cursor / Cline)
  │ POST /v1/chat/completions (OpenAI)
  │ POST /v1/messages (Anthropic)
  ▼
AtomCode2API (:13457) ─── 协议翻译层
  │ POST /chat (SSE stream)
  ▼
AtomCode Daemon (:13456) ─── 私有 daemon 协议
  │ → llm-api.atomgit.com
  ▼
AI 模型 (deepseek-v4-flash, Qwen3-VL, etc.)
```

## 功能特性 / Features

| 特性 | 说明 |
|------|------|
| **双协议** | OpenAI Chat Completions + Anthropic Messages，一套引擎双出口 |
| **流式输出** | SSE 实时流式返回，打字机效果 |
| **工具调用** | 完整的 tool_use/tool_call 翻译，文件操作、命令执行等 |
| **推理过程** | daemon reasoning → OpenAI reasoning_content / Anthropic thinking |
| **多轮对话** | 基于 daemon session_id 的上下文保持（30 分钟过期） |
| **Web 管理面板** | 内置 Dashboard，管理账号、查看统计、配置设置 |
| **账号管理** | 多账号支持、一键导入、OAuth 授权登录、Token 管理 |
| **请求日志** | SQLite 记录请求历史（模型、延迟、Token 用量）|
| **Docker 部署** | 多阶段构建，一行命令启动 |

## 快速开始 / Quick Start

### 前置条件

- [AtomCode](https://atomcode.atomgit.com/) 已安装并登录
- Go 1.25+（本地构建）或 Docker（容器化部署）

### 方式一：本地运行

```bash
# 1. 确保 AtomCode daemon 在运行
atomcode daemon --port 13456 --idle-timeout 0

# 2. 克隆并构建
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api
go build -o atomcode-2api ./cmd/atomcode-2api/

# 3. 启动代理
./atomcode-2api serve -v

# 访问管理面板
# http://localhost:13457/
```

### 方式二：Docker 运行

```bash
# 构建镜像
docker build -t atomcode-2api .

# 运行（与宿主机 daemon 通信）
docker run -d \
  --name atomcode-2api \
  --add-host host.docker.internal:host-gateway \
  -p 13457:13457 \
  -v atomcode-data:/data \
  atomcode-2api

# 访问管理面板
# http://localhost:13457/
```

### 配置 Claude Code

```bash
export ANTHROPIC_BASE_URL=http://localhost:13457
export ANTHROPIC_API_KEY=sk-atmc-xxxxx
export ANTHROPIC_MODEL=deepseek-v4-flash
claude
```

### 配置 Codex / OpenAI SDK

```bash
export OPENAI_BASE_URL=http://localhost:13457/v1
export OPENAI_API_KEY=sk-atmc-xxxxx
export OPENAI_MODEL=deepseek-v4-flash
codex exec "你的问题"
```

### 快捷启动脚本

在 `~/.zshrc` 中添加：

```bash
# 启动 Claude Code 连接到本代理
alias atomcode-cc='ANTHROPIC_BASE_URL=http://localhost:13457 \
ANTHROPIC_API_KEY=$(curl -s http://localhost:13457/api/accounts | python3 -c "import sys,json; print(json.load(sys.stdin)[\"accounts\"][0][\"api_token\"])") \
ANTHROPIC_MODEL=deepseek-v4-flash \
claude --dangerously-skip-permissions'

# 启动 Codex 连接到本代理
alias atomcode-codex='OPENAI_BASE_URL=http://localhost:13457/v1 \
OPENAI_API_KEY=$(curl -s http://localhost:13457/api/accounts | python3 -c "import sys,json; print(json.load(sys.stdin)[\"accounts\"][0][\"api_token\"])") \
OPENAI_MODEL=deepseek-v4-flash \
codex'
```

## Web 管理面板

访问 `http://localhost:13457/` 进入 Dashboard：

| 页面 | 路径 | 功能 |
|------|------|------|
| 数据概览 | `/dashboard` | 请求统计、图表、模型用量 |
| 账号管理 | `/accounts` | 添加/导入/导出账号，管理 Token |
| 账号详情 | `/accounts/:userId` | 快速接入命令、模型目录、请求日志 |
| 系统设置 | `/settings` | 请求超时、日志开关、修改密码 |

## API 端点

| 路径 | 方法 | 协议 | 说明 |
|------|------|------|------|
| `POST /v1/chat/completions` | OpenAI | Chat Completions | 对话补全（支持流式） |
| `POST /v1/messages` | Anthropic | Messages API | 对话补全（支持流式） |
| `GET /v1/models` | - | Model list | 可用模型列表 |
| `GET /health` | - | Health check | 健康检查 |

## 可用模型 / Available Models

| 模型 | 提供商 | 上下文 | CodingPlan |
|------|--------|--------|:----------:|
| **deepseek-v4-flash** | AtomGit | 1,000,000 | ✅ 免费 |
| **Qwen/Qwen3-VL-8B-Instruct** | AtomGit | 64,000 | ✅ 免费 |
| **deepseek-chat** | DeepSeek | 128,000 | ❌ 需 Pro |
| **deepseek-reasoner** | DeepSeek | 128,000 | ❌ 需 Pro |
| **glm-5.2** | Zhipu AI | 128,000 | ❌ 需 Pro |

## 构建 / Build

```bash
# 构建后端
go build -o atomcode-2api ./cmd/atomcode-2api/

# 构建前端（嵌入二进制）
cd web && npm ci && npm run build

# Docker 构建
docker build -t atomcode-2api .
```

## 项目结构 / Project Structure

```
cmd/atomcode-2api/    CLI 入口（cobra 命令）
pkg/atmc/              AtomCode daemon HTTP 客户端 + SSE 翻译器
pkg/openai/            OpenAI 协议层
pkg/anthropic/         Anthropic 协议层
pkg/store/             SQLite 存储（账号、设置、日志）
pkg/auth/              JWT 认证 + 加密
pkg/dashboard/         Web 管理面板
pkg/keepalive/         凭据保活
pkg/proxy/             会话追踪
web/                   React 前端（TypeScript + Ant Design）
```

## 许可证 / License

Apache 2.0