# AtomCode2API

**AtomCode 有个免费版 CodingPlan，每个月能免费调用几千次 AI。但问题是 Claude Code、Cursor 这些工具连不上它，因为 AtomCode 用的是自己的私有协议。**

AtomCode2API 就是干这个活的——在中间做个翻译，把你的 AI 工具和 AtomCode 连接起来，不用再另外花钱买 API Key。

---

## 长话短说

```bash
# 你已经有 AtomCode 了，装好这个代理，然后一行命令就能用 Claude Code 了：
export ANTHROPIC_BASE_URL=http://localhost:13457
export ANTHROPIC_API_KEY=sk-atmc-xxxxx
claude
```

## 怎么工作的

```
你的工具（Claude Code / Codex / Cursor）
  │ 发标准请求（OpenAI 格式或 Anthropic 格式）
  ▼
AtomCode2API（端口 13457）——翻译层
  │ 翻译成 AtomCode 能懂的协议
  ▼
AtomCode Daemon（端口 13456）
  │ 调上游模型
  ▼
deepseek-v4-flash 等模型
```

## 能干嘛

- **不用买 API Key**，用 AtomCode 的免费额度就行
- Claude Code、Codex、Cursor、Cline 都能连
- 有 Web 管理面板，能看到请求记录、Token 用量、模型列表
- 支持多账号，可以管理多个 AtomCode 账号
- 支持流式输出、工具调用、多轮对话

## 快速开始

### 前提

- 装好了 [AtomCode](https://atomcode.atomgit.com/) 并且登录了
- 跑着 `atomcode daemon --port 13456 --idle-timeout 0`

### 本地运行

```bash
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api
go build -o atomcode-2api ./cmd/atomcode-2api/
./atomcode-2api serve -v

# 打开 http://localhost:13457/ 就能看到管理面板
```

### Docker 运行

```bash
docker build -t atomcode-2api .
docker run -d --name atomcode-2api \
  --add-host host.docker.internal:host-gateway \
  -p 13457:13457 \
  atomcode-2api
```

### 配置 Claude Code

```bash
export ANTHROPIC_BASE_URL=http://localhost:13457
export ANTHROPIC_API_KEY=sk-atmc-xxxxx   # 从管理面板获取
export ANTHROPIC_MODEL=deepseek-v4-flash
claude
```

### 配置 Codex

```bash
export OPENAI_BASE_URL=http://localhost:13457/v1
export OPENAI_API_KEY=sk-atmc-xxxxx
export OPENAI_MODEL=deepseek-v4-flash
codex exec "你的问题"
```

## 管理面板

打开 `http://localhost:13457/`：

| 页面 | 说干嘛的 |
|------|----------|
| 数据概览 | 请求数、Token 用量、图表 |
| 账号管理 | 导入/添加/删除账号 |
| 账号详情 | 复制一键启动命令、看请求日志、选模型 |
| 系统设置 | 超时时间、日志开关、改密码 |

## 可用模型

| 模型 | 上下文 | 免费? |
|------|--------|:-----:|
| deepseek-v4-flash | 1,000,000 | ✅ 免费 |
| Qwen/Qwen3-VL-8B-Instruct | 64,000 | ✅ 免费 |
| deepseek-chat | 128,000 | ❌ 需 Pro |
| deepseek-reasoner | 128,000 | ❌ 需 Pro |
| glm-5.2 | 128,000 | ❌ 需 Pro |

## 项目结构

```
cmd/atomcode-2api/    命令行入口
pkg/atmc/             和 AtomCode daemon 通信的客户端
pkg/openai/           OpenAI 协议处理
pkg/anthropic/        Anthropic 协议处理
pkg/store/            数据库（账号、设置、日志）
pkg/auth/             认证
pkg/dashboard/        Web 管理面板
web/                  前端页面（React + Ant Design）
```

## 许可证

Apache 2.0