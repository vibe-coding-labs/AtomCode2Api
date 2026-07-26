#!/bin/bash
# AtomCode2API 一键编译脚本
# 用法: curl -sL https://raw.githubusercontent.com/vibe-coding-labs/AtomCode2Api/main/scripts/build.sh | bash
# 或: wget -qO- https://raw.githubusercontent.com/vibe-coding-labs/AtomCode2Api/main/scripts/build.sh | bash

set -e

echo "================================================"
echo "  AtomCode2API 一键编译"
echo "================================================"
echo ""

# 检查依赖
echo "🔍 检查依赖..."

# Go
if ! command -v go &> /dev/null; then
    echo "❌ 未找到 Go，请先安装 Go 1.25+"
    echo "   安装指南: https://go.dev/doc/install"
    exit 1
fi
echo "  ✅ Go $(go version | grep -oP 'go\S+' || go version)"

# Node.js
if ! command -v node &> /dev/null; then
    echo "❌ 未找到 Node.js，请先安装 Node.js 18+"
    echo "   安装指南: https://nodejs.org/"
    exit 1
fi
echo "  ✅ Node.js $(node -v)"

# npm
if ! command -v npm &> /dev/null; then
    echo "❌ 未找到 npm"
    exit 1
fi
echo "  ✅ npm $(npm -v)"

echo ""

# 克隆或进入目录
if [ ! -d "AtomCode2Api" ]; then
    echo "📦 克隆仓库..."
    git clone https://github.com/vibe-coding-labs/AtomCode2Api.git
    cd AtomCode2Api
else
    echo "📂 使用已有目录 AtomCode2Api"
    cd AtomCode2Api
    git pull
fi

echo ""

# 构建前端
echo "🔨 构建前端..."
cd web
npm install --silent
npm run build
cd ..
echo "  ✅ 前端构建完成"

# 构建后端
echo "🔨 构建后端..."
go build -o atomcode-2api ./cmd/atomcode-2api/
echo "  ✅ 后端构建完成"

echo ""
echo "================================================"
echo "  ✅ 编译完成！"
echo "================================================"
echo ""
echo "  二进制文件: $(pwd)/atomcode-2api"
echo "  版本: $($(pwd)/atomcode-2api --version 2>&1 || echo 'unknown')"
echo ""
echo "  启动方式:"
echo "    ./atomcode-2api serve -v"
echo ""
echo "  管理面板:"
echo "    http://localhost:45678/"
echo ""
echo "  验证:"
echo "    curl http://localhost:45678/api/health"
echo ""
