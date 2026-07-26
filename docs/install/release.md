# 从 GitHub Release 下载 AtomCode2API

> 适合不想安装编译环境的用户，直接下载编译好的二进制文件。

## 前置条件

- AtomCode daemon 已在运行（`atomcode daemon --port 13456 --idle-timeout 0`）

## 选择对应平台的二进制

### 确定你的系统架构

```bash
uname -sm
```

输出示例与对应下载文件：

| 输出 | 架构 | 下载文件 |
|------|------|----------|
| `Linux x86_64` | Linux amd64 | `AtomCode2Api_Linux_x86_64.tar.gz` |
| `Linux aarch64` | Linux arm64 | `AtomCode2Api_Linux_arm64.tar.gz` |
| `Linux armv7l` | Linux armhf | `AtomCode2Api_Linux_armhf.tar.gz` |
| `Linux i686` | Linux 386 | `AtomCode2Api_Linux_i386.tar.gz` |
| `Linux riscv64` | Linux riscv64 | `AtomCode2Api_Linux_riscv64.tar.gz` |
| `Darwin x86_64` | macOS Intel | `AtomCode2Api_Darwin_x86_64.tar.gz` |
| `Darwin arm64` | macOS Apple Silicon | `AtomCode2Api_Darwin_arm64.tar.gz` |
| `Windows` (PowerShell) | Windows x86_64 | `AtomCode2Api_Windows_x86_64.zip` |

### 完整平台列表

所有 15 个可用平台见 [GitHub Releases 页面](https://github.com/vibe-coding-labs/AtomCode2Api/releases)。

## 操作步骤

### Linux / macOS

```bash
# 1. 下载最新版本（替换 v0.1.0 为最新版本号）
# 查看最新版本：https://github.com/vibe-coding-labs/AtomCode2Api/releases/latest
VERSION=v0.1.0
ARCH=x86_64  # 按上面表格替换为你对应的架构

# 下载
wget "https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/${VERSION}/AtomCode2Api_Linux_${ARCH}.tar.gz"

# 或使用 curl
curl -L -O "https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/${VERSION}/AtomCode2Api_Linux_${ARCH}.tar.gz"

# 2. 解压
tar xzf "AtomCode2Api_Linux_${ARCH}.tar.gz"

# 3. 移到 PATH
sudo mv AtomCode2Api /usr/local/bin/atomcode-2api

# 4. 启动
atomcode-2api serve -v

# 5. 验证
curl http://localhost:45678/api/health
```

### Windows（PowerShell）

```powershell
# 1. 下载
$version = "v0.1.0"
Invoke-WebRequest -Uri "https://github.com/vibe-coding-labs/AtomCode2Api/releases/download/$version/AtomCode2Api_Windows_x86_64.zip" -OutFile "atomcode-2api.zip"

# 2. 解压
Expand-Archive -Path "atomcode-2api.zip" -DestinationPath "atomcode-2api"

# 3. 运行
cd atomcode-2api
.\AtomCode2Api.exe serve -v
```

## 验证清单

- [ ] 下载的二进制能运行（`./atomcode-2api --version`）
- [ ] `curl http://localhost:45678/api/health` 返回 `{"status":"ok"}`
- [ ] 管理面板可访问
