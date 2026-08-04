#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# TFT Copilot 一键打包脚本（在开发机上执行）
#
# 产出：dist/tft-copilot-<版本>-<日期>.tar.gz
#   - Linux amd64 + arm64 双架构静态二进制（纯 Go，CGO_ENABLED=0，无依赖）
#   - 运行时数据（metadata + knowledge，与镜像内布局一致）
#   - install.sh（服务器一键安装）/ tft-copilot.service / .env.example / DEPLOY.md
#
# 用法：
#   ./deploy/build-release.sh            # 自动从 git 读取版本
#   VERSION=v1.0.1 ./deploy/build-release.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)

# ── 版本信息 ──────────────────────────────────────────────────────────────────
VERSION="${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0-dev")}"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
DATE_TAG=$(date '+%Y%m%d')

VERSION_PKG="github.com/sagerlabs/awesome/tft"
LDFLAGS="-s -w -X '${VERSION_PKG}.Version=${VERSION}' -X '${VERSION_PKG}.GitCommit=${GIT_COMMIT}' -X '${VERSION_PKG}.BuildTime=${BUILD_TIME}'"

echo "📦 打包 TFT Copilot ${VERSION} (${GIT_COMMIT})"

# ── 数据完整性检查 ────────────────────────────────────────────────────────────
for f in comps_for_agent.json items_priority.json localization.json; do
    [ -f "metadata/tft-meta/data/$f" ] || { echo "❌ 缺少 metadata/tft-meta/data/$f，请先 make data"; exit 1; }
done
[ -d "tft/knowledge/data" ] || { echo "❌ 缺少 tft/knowledge/data，请先 make data"; exit 1; }

# ── 交叉编译双架构 ────────────────────────────────────────────────────────────
STAGE="dist/tft-copilot-release"
rm -rf "$STAGE"
mkdir -p "$STAGE/bin"

for target in "amd64" "arm64"; do
    echo "🔨 编译 linux/${target} ..."
    CGO_ENABLED=0 GOOS=linux GOARCH=${target} \
        go build -ldflags "${LDFLAGS}" -o "${STAGE}/bin/tft-copilot-linux-${target}" ./main.go
done

# ── 运行时数据（布局与 Docker 镜像一致，binary 以 /opt/tft-copilot 为工作目录） ──
mkdir -p "$STAGE/metadata/tft-meta"
cp -R metadata/tft-meta/data "$STAGE/metadata/tft-meta/data"
mkdir -p "$STAGE/tft/knowledge"
cp -R tft/knowledge/data "$STAGE/tft/knowledge/data"

# ── 部署资产 ──────────────────────────────────────────────────────────────────
cp deploy/install.sh "$STAGE/install.sh"
cp deploy/tft-copilot.service "$STAGE/tft-copilot.service"
cp deploy/.env.example "$STAGE/.env.example"
cp deploy/DEPLOY.md "$STAGE/DEPLOY.md"
chmod +x "$STAGE/install.sh"

# ── 打 tarball ────────────────────────────────────────────────────────────────
TARBALL="dist/tft-copilot-${VERSION}-${DATE_TAG}.tar.gz"
mkdir -p dist
# 兼容性：gnutar 格式（Linux GNU tar 原生读取，无 PAX 扩展头警告）；
# 属主归零（服务器 sudo 解压即 root:root）；COPYFILE_DISABLE 防 macOS 混入 AppleDouble
COPYFILE_DISABLE=1 tar --format=gnutar --uid 0 --gid 0 --numeric-owner \
    -czf "$TARBALL" -C dist tft-copilot-release

SIZE=$(du -h "$TARBALL" | cut -f1)
echo ""
echo "✅ 打包完成: ${TARBALL} (${SIZE})"
echo ""
echo "云服务器一键部署："
echo "  1. 上传:   scp ${TARBALL} user@<服务器>:~/"
echo "  2. 解压:   tar -xzf tft-copilot-${VERSION}-${DATE_TAG}.tar.gz && cd tft-copilot-release"
echo "  3. 配置:   cp .env.example .env && vi .env   # 填入 LLM API Key"
echo "  4. 安装:   sudo ./install.sh"
