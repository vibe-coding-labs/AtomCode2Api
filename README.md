# AtomCode2API

嗨，你好。如果你点进来了，大概是因为你也在用 AtomCode 的免费额度，想把它用到别的工具上？那来对地方了。

## 这东西是干嘛的

简单说就是：**AtomCode 有个免费版（CodingPlan Lite），每个月能免费用很多次 AI。但 Claude Code、Cursor、Codex 这些工具连不上它，因为 AtomCode 用的是自己的私有协议，不是标准的 OpenAI 接口。**

AtomCode2API 就是一个"翻译官"——它坐在你的工具和 AtomCode 之间，把工具发的标准请求翻译成 AtomCode 能听懂的话，再把 AtomCode 的回复翻译回来。

所以，你不需要再花钱买 OpenAI 的 API Key，直接用 AtomCode 的免费额度就够了。

## 大概长这样

装好之后，你会有个 Web 管理面板，打开浏览器就能看到：

```
http://localhost:45678/
```

里面有这么几个页面：

- **数据概览** — 今天发了多少请求、用了多少 Token、成功失败了几个，图表都有
- **账号管理** — 你的 AtomCode 账号列表，可以一键导入，也能手动添加
- **账号详情** — 点进某个账号，能看到：
  - 复制即用的 Claude Code / Codex 启动命令（不用自己拼环境变量）
  - 可用模型列表（哪个免费哪个付费，价格多少）
  - 请求日志（什么时候调的哪个模型，延迟多少）
  - 套餐信息（你的 CodingPlan 啥时候到期，还剩多少天）
- **系统设置** — 超时时间、日志开关、改密码

## 解决了什么问题

你有 AtomCode 的免费额度，但你想用 Claude Code 或者 Codex 或者 Cursor 来写代码。这些工具只认 OpenAI 和 Anthropic 的标准接口，不认 AtomCode 的私有协议。

之前你可能得：
1. 再掏钱买 OpenAI 的 API Key
2. 或者在几个工具之间来回切换

现在不用了，装个 AtomCode2API 就行。

## 怎么装

### 前提

你已经装好了 [AtomCode](https://atomcode.atomgit.com/)，登录了，并且在跑着：

```bash
atomcode daemon --port 13456 --idle-timeout 0
```

### 方式一：本地跑

```bash
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api
go build -o atomcode-2api ./cmd/atomcode-2api/
./atomcode-2api serve -v
```

然后打开 http://localhost:45678/ 就能看到管理面板了。

### 方式二：Docker 跑

```bash
docker build -t atomcode-2api .
docker run -d --name atomcode-2api \
  --add-host host.docker.internal:host-gateway \
  -p 45678:45678 \
  atomcode-2api
```

## 怎么用

### 配置 Claude Code

```bash
export ANTHROPIC_BASE_URL=http://localhost:45678
export ANTHROPIC_API_KEY=sk-atmc-xxxxx
export ANTHROPIC_MODEL=deepseek-v4-flash
claude
```

以上三个环境变量在管理面板的账号详情页有一键复制按钮，点一下就能复制完整命令。

### 配置 Codex

```bash
export OPENAI_BASE_URL=http://localhost:45678/v1
export OPENAI_API_KEY=sk-atmc-xxxxx
export OPENAI_MODEL=deepseek-v4-flash
codex exec "你的问题"
```

### 配置 Cursor

在 Cursor Settings → Models 里填：
- API Base URL: `http://localhost:45678/v1`
- API Key: 从管理面板复制
- Model: `deepseek-v4-flash`

## 有哪些模型可以用

| 模型 | 提供商 | 上下文 | 要钱吗 |
|------|--------|--------|:------:|
| deepseek-v4-flash | AtomGit | 1,000,000 | 免费 ✅ |
| Qwen/Qwen3-VL-8B-Instruct | AtomGit | 64,000 | 免费 ✅ |
| deepseek-chat | DeepSeek | 128,000 | 需要 Pro |
| deepseek-reasoner | DeepSeek | 128,000 | 需要 Pro |
| glm-5.2 | Zhipu AI | 128,000 | 需要 Pro |

免费的模型 CodingPlan 直接覆盖，不用额外花钱。付费的模型需要升级 Pro 套餐。

## 代码结构

```
cmd/atomcode-2api/    命令行工具（启动、设置、登录等）
pkg/atmc/             和 AtomCode Daemon 通信的客户端
pkg/openai/           OpenAI 协议翻译
pkg/anthropic/        Anthropic 协议翻译
pkg/store/            SQLite 数据库（存账号、设置、请求日志）
pkg/auth/             认证相关
pkg/dashboard/        Web 管理面板
web/                  前端页面（React + Ant Design）
```

## 提示词：让 AI agent 帮你本地构建

如果你想让你的 AI agent（Claude Code、Codex 等）直接帮你把 AtomCode2API 在本地搭起来，把下面这段提示词复制给它就行：

> 请帮我完成以下操作：
>
> 1. 克隆仓库：`git clone https://github.com/vibe-coding-labs/AtomCode2Api.git`
> 2. 进入目录：`cd AtomCode2Api`
> 3. 构建前端：`cd web && npm install && npm run build && cd ..`
> 4. 构建后端：`go build -o atomcode-2api ./cmd/atomcode-2api/`
> 5. 确保 AtomCode daemon 在运行（`atomcode daemon --port 13456 --idle-timeout 0`）
> 6. 启动代理：`./atomcode-2api serve -v`
> 7. 确认代理正常运行：`curl http://localhost:45678/api/health`
>
> 以上步骤完成后，代理服务会在 `http://localhost:45678/` 启动，管理面板可正常访问。

## 说在最后

这个项目是开源的（Apache 2.0），如果你觉得有用，欢迎 star。如果你遇到问题，提 issue 就行。

能用免费额度解决的问题，何必花钱呢？