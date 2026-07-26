#!/bin/bash
# AtomCode2API 发布脚本
# 用法: ./scripts/release.sh <version>

set -e

if [ -z "$1" ]; then
    echo "用法: ./scripts/release.sh <version>"
    echo "示例: ./scripts/release.sh v0.2.0"
    exit 1
fi

VERSION="$1"

echo "================================================"
echo "  AtomCode2API 发布 v${VERSION#v}"
echo "================================================"
echo ""

# 检查是否有未提交的变更
if [ -n "$(git status --porcelain)" ]; then
    echo "❌ 有未提交的变更，请先提交"
    git status --short
    exit 1
fi

# 更新版本
echo "📝 更新版本号..."
echo "package main

// Version is set at build time.
const Version = \"${VERSION#v}\"" > cmd/atomcode-2api/version.go
git add cmd/atomcode-2api/version.go
git commit -m "chore: bump version to ${VERSION#v}"

# 打标签
echo "🏷️  打标签 $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

# 推送
echo "📤 推送标签..."
git push origin "$VERSION"

# 让 goreleaser 发布
echo "🚀 发布到 GitHub Release..."
GITHUB_TOKEN=$(gh auth token) goreleaser release --clean

echo ""
echo "✅ 发布完成: https://github.com/vibe-coding-labs/AtomCode2Api/releases/tag/$VERSION"