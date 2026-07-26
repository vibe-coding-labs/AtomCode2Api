# Docker 部署 AtomCode2API

> 适合有 Docker 环境的用户，不需要安装 Go 或 Node.js。

## 前置条件

- Docker 已安装
- AtomCode daemon 已在宿主机运行（`atomcode daemon --port 13456 --idle-timeout 0`）

## 操作步骤

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

## 验证清单

- [ ] `docker ps` 显示 `atomcode-2api` 容器在运行
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 浏览器能打开管理面板

## 常见问题

### 容器无法连接 daemon
启动容器时加 `--add-host host.docker.internal:host-gateway`，daemon 地址设为 `http://host.docker.internal:13456`。

### 数据持久化
`-v atomcode-data:/data` 会把 SQLite 数据库持久化到 Docker volume 中，容器重启数据不丢失。
