# AtomCode2API — 安装指南

> 本文档是为 AI 编程助手（Claude Code、Codex、Cursor 等）准备的安装引导。
> 目标是让 AI agent 根据用户环境自动选择最佳的安装方式，完成部署。

---

## 快速判断：选哪种方式？

```
用户有 Docker 吗？ ──是──→ 方式一：Docker 部署
        │
        否
        │
用户有 Go 1.25+ 吗？ ──是──→ 方式二：源码编译
        │
        否
        │
        方式三：下载 Release 二进制
```

---

## 方式一：Docker 部署（推荐，最省心）

**详细文档：** https://github.com/vibe-coding-labs/AtomCode2Api/blob/main/docs/install/docker.md

### 快速操作

```bash
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api
docker build -t atomcode-2api .
docker run -d --name atomcode-2api --add-host host.docker.internal:host-gateway -p 45678:45678 atomcode-2api
curl http://localhost:45678/api/health
```

### 验证清单
- [ ] `docker ps` 显示容器在运行
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 浏览器打开 `http://localhost:45678/` 能看到管理面板

---

## 方式二：源码编译

**详细文档：** https://github.com/vibe-coding-labs/AtomCode2Api/blob/main/docs/install/build.md

**一键编译脚本：** https://github.com/vibe-coding-labs/AtomCode2Api/blob/main/scripts/build.sh

### 快速操作

```bash
curl -sL https://raw.githubusercontent.com/vibe-coding-labs/AtomCode2Api/main/scripts/build.sh | bash
```

或手动：

```bash
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api
cd web && npm install && npm run build && cd ..
go build -o atomcode-2api ./cmd/atomcode-2api/
./atomcode-2api serve -v
```

### 前置条件
- Go 1.25+
- Node.js 18+ + npm
- GCC（Linux）或 mingw-w64（Windows）

### 验证清单
- [ ] `./atomcode-2api --version` 显示版本号
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`

---

## 方式三：下载 Release 二进制

**详细文档：** https://github.com/vibe-coding-labs/AtomCode2Api/blob/main/docs/install/release.md

### 快速操作

```bash
# 确定系统架构
uname -sm

# 下载最新版
VERSION=v0.1.0  # 查看最新: https://github.com/vibe-coding-labs/AtomCode2Api/releases/latest
ARCH=x86_64     # 替换为你的架构
wget "https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/${VERSION}/AtomCode2Api_Linux_${ARCH}.tar.gz"
tar xzf "AtomCode2Api_Linux_${ARCH}.tar.gz"
sudo mv AtomCode2Api /usr/local/bin/atomcode-2api
atomcode-2api serve -v
```

### 支持平台

Linux (amd64/arm64/armhf/386/riscv64/ppc64le/s390x), macOS (Intel/Apple Silicon), Windows (amd64/386/arm64), FreeBSD (amd64), OpenBSD (amd64)

### 验证清单
- [ ] `./atomcode-2api --version` 显示版本号
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`

---

## 安装后配置客户端

### Claude Code

```bash
export ANTHROPIC_BASE_URL=http://localhost:45678
export ANTHROPIC_API_KEY=sk-atmc-xxxxx   # 从管理面板获取
claude
```

### Codex / OpenAI SDK

```bash
export OPENAI_BASE_URL=http://localhost:45678/v1
export OPENAI_API_KEY=sk-atmc-xxxxx
export OPENAI_MODEL=deepseek-v4-flash
codex exec "你的问题"
```

### 获取 API Token

打开管理面板 `http://localhost:45678/` → 账号管理 → 点击账号 → 复制 API Token。

---

## 常见问题

### Q: 连接失败？
A: 确保 daemon 在运行：`atomcode daemon --port 13456 --idle-timeout 0`

### Q: 提示 `user has no codingplan`？
A: 运行 `atomcode-2api setup` 登录并领取免费额度。

### Q: Docker 连不上 daemon？
A: 启动容器时加 `--add-host host.docker.internal:host-gateway`

---

> 如果安装过程中遇到问题，请在 GitHub 提 [issue](https://github.com/vibe-coding-labs/AtomCode2Api/issues)。