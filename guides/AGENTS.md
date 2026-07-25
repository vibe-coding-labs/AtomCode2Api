# AtomCode2API — AI Agent 安装指南

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

### 前置条件
- Docker 已安装
- AtomCode daemon 已在宿主机运行（`atomcode daemon --port 13456 --idle-timeout 0`）

### 操作步骤

```bash
# 1. 克隆仓库
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api

# 2. 构建镜像
docker build -t atomcode-2api .

# 3. 启动容器
docker run -d \
  --name atomcode-2api \
  --add-host host.docker.internal:host-gateway \
  -p 45678:45678 \
  -v atomcode-data:/data \
  atomcode-2api

# 4. 验证
curl http://localhost:45678/api/health

# 5. 打开管理面板
# 浏览器访问 http://localhost:45678/
```

### 验证清单
- [ ] `docker ps` 显示 `atomcode-2api` 容器在运行
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 浏览器能打开管理面板

---

## 方式二：源码编译（适合开发者）

### 前置条件
- Go 1.25+
- Node.js 18+ + npm
- GCC（Linux）或 mingw-w64（Windows）
- AtomCode daemon 已在运行

### 操作步骤

```bash
# 1. 克隆仓库
git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
cd AtomCode2Api

# 2. 构建前端（编译后嵌入 Go 二进制）
cd web
npm install
npm run build
cd ..

# 3. 构建后端
go build -o atomcode-2api ./cmd/atomcode-2api/

# 4. 确保 daemon 在运行
atomcode daemon --port 13456 --idle-timeout 0 &

# 5. 启动代理
./atomcode-2api serve -v

# 6. 验证
curl http://localhost:45678/api/health
```

### 快速启动脚本

```bash
# 添加到 ~/.zshrc 或 ~/.bashrc
alias atomcode-serve='cd ~/AtomCode2Api && go build -o atomcode-2api ./cmd/atomcode-2api/ && ./atomcode-2api serve -v'
alias atomcode-cc='ANTHROPIC_BASE_URL=http://localhost:45678 \
ANTHROPIC_API_KEY=$(curl -s http://localhost:45678/api/accounts | python3 -c "import sys,json;print(json.load(sys.stdin)[\"accounts\"][0][\"api_token\"])") \
ANTHROPIC_MODEL=deepseek-v4-flash \
claude --dangerously-skip-permissions'
```

### 验证清单
- [ ] `./atomcode-2api --version` 显示版本号
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 管理面板可访问

---

## 方式三：下载 Release 二进制（无需编译环境）

### 前置条件
- 从 [GitHub Releases](https://github.com/vibe-coding-labs/AtomCode2Api/releases) 下载对应平台的压缩包
- AtomCode daemon 已在运行

### 选择对应平台的二进制

| 操作系统 | 架构 | 下载文件 |
|----------|------|----------|
| Linux | x86_64 | `AtomCode2Api_Linux_x86_64.tar.gz` |
| Linux | arm64 | `AtomCode2Api_Linux_arm64.tar.gz` |
| Linux | armhf | `AtomCode2Api_Linux_armhf.tar.gz` |
| Linux | i386 | `AtomCode2Api_Linux_i386.tar.gz` |
| Linux | riscv64 | `AtomCode2Api_Linux_riscv64.tar.gz` |
| Linux | ppc64le | `AtomCode2Api_Linux_ppc64le.tar.gz` |
| Linux | s390x | `AtomCode2Api_Linux_s390x.tar.gz` |
| macOS | Intel | `AtomCode2Api_Darwin_x86_64.tar.gz` |
| macOS | Apple Silicon | `AtomCode2Api_Darwin_arm64.tar.gz` |
| Windows | x86_64 | `AtomCode2Api_Windows_x86_64.zip` |
| Windows | i386 | `AtomCode2Api_Windows_i386.zip` |
| Windows | arm64 | `AtomCode2Api_Windows_arm64.zip` |
| FreeBSD | x86_64 | `AtomCode2Api_Freebsd_x86_64.tar.gz` |
| OpenBSD | x86_64 | `AtomCode2Api_Openbsd_x86_64.tar.gz` |

### 操作步骤

```bash
# 1. 确定你的系统架构
uname -sm
# 输出示例: "Linux x86_64" → 下载 Linux_x86_64

# 2. 下载对应版本的二进制（示例：Linux x86_64）
wget https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/v0.1.0/AtomCode2Api_Linux_x86_64.tar.gz

# 或使用 curl
curl -L -O https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/v0.1.0/AtomCode2Api_Linux_x86_64.tar.gz

# 3. 解压
tar xzf AtomCode2Api_Linux_x86_64.tar.gz

# 4. 移到 PATH
sudo mv AtomCode2Api /usr/local/bin/atomcode-2api

# 5. 启动
atomcode-2api serve -v

# 6. 验证
curl http://localhost:45678/api/health
```

### Windows 用户

```powershell
# 1. 下载 .zip 文件
Invoke-WebRequest -Uri "https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/v0.1.0/AtomCode2Api_Windows_x86_64.zip" -OutFile "atomcode-2api.zip"

# 2. 解压
Expand-Archive -Path "atomcode-2api.zip" -DestinationPath "atomcode-2api"

# 3. 运行
cd atomcode-2api
.\AtomCode2Api.exe serve -v
```

### 验证清单
- [ ] 下载的二进制能运行（`./atomcode-2api --version`）
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 管理面板可访问

---

## 安装后：配置客户端

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

### Q: `curl http://localhost:45678/api/health` 连接失败
A: 代理服务没启动。先运行 `atomcode-2api serve -v` 查看日志。

### Q: 提示 `daemon unreachable`
A: AtomCode daemon 没在运行。确保 `atomcode daemon --port 13456 --idle-timeout 0` 在后台运行。

### Q: 提示 `user has no codingplan`
A: 运行 `atomcode-2api setup` 登录并领取 CodingPlan 免费额度。

### Q: Docker 容器无法连接 daemon
A: 启动容器时加 `--add-host host.docker.internal:host-gateway`，daemon 地址设为 `http://host.docker.internal:13456`。

---

> 如果安装过程中遇到问题，请在 GitHub 提 [issue](https://github.com/vibe-coding-labs/AtomCode2Api/issues)。
