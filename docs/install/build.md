# 源码编译 AtomCode2API

> 适合开发者，从源码编译完整的二进制文件。

## 前置条件

- Go 1.25+
- Node.js 18+ + npm
- GCC（Linux）或 mingw-w64（Windows）
- AtomCode daemon 已在运行

## 一键编译脚本

```bash
# 下载脚本并执行
curl -sL https://raw.githubusercontent.com/vibe-coding-labs/AtomCode2Api/main/scripts/build.sh | bash
```

编译完成后，二进制文件在当前目录下的 `atomcode-2api`。

## 手动编译步骤

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

# 4. 验证
./atomcode-2api --version

# 5. 启动
./atomcode-2api serve -v

# 6. 验证
curl http://localhost:45678/api/health
```

## 验证清单

- [ ] `./atomcode-2api --version` 显示版本号
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 管理面板可访问
