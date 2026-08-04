#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# TFT Copilot 一键打包发布脚本（在开发机上执行）
#
# 流程：make release（打包）→ scp 上传 → 远端解压 → sudo ./install.sh
#
# 前置条件：
#   1. ssh 免密登录服务器（ssh-copy-id）
#   2. 远端用户可 sudo（建议直接用 root：make publish SERVER=root@1.2.3.4）
#   3. 首次部署需先在服务器上配置 /opt/tft-copilot/.env（LLM API Key），
#      之后每次 publish 全自动（.env 会被保留）
#
# 用法：
#   ./deploy/publish.sh root@1.2.3.4
#   make publish SERVER=root@1.2.3.4
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SERVER="${1:-}"
[ -n "$SERVER" ] || { echo "❌ 用法: $0 user@host（例: root@1.2.3.4）"; exit 1; }

# 先验证连通性，避免编译一分钟后才发现连不上
echo "🔌 检查 ssh 连通性 ${SERVER} ..."
ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER" true \
    || { echo "❌ 无法 ssh 到 ${SERVER}（先配置免密: ssh-copy-id ${SERVER}）"; exit 1; }

cd "$(dirname "$0")/.."

# ── 1. 打包 ───────────────────────────────────────────────────────────────────
./deploy/build-release.sh
TARBALL=$(ls -t dist/tft-copilot-*.tar.gz | head -1)
PKG=$(basename "$TARBALL")
echo ""
echo "🚀 发布 ${PKG} → ${SERVER}"

# ── 2. 上传 ───────────────────────────────────────────────────────────────────
scp "$TARBALL" "${SERVER}:~/"

# ── 3. 远端解压 + 一键安装 ────────────────────────────────────────────────────
# shellcheck disable=SC2087
ssh "$SERVER" bash -s <<REMOTE
set -euo pipefail
cd ~
rm -rf tft-copilot-release
tar -xzf "${PKG}"
cd tft-copilot-release
sudo ./install.sh
REMOTE

echo ""
echo "✅ 发布完成！"
echo "   验证: ssh ${SERVER} 'curl -s localhost:8080/v1/tft/health'"
echo "   页面: http://<服务器IP>:8080/（首次部署记得先配 .env，见 DEPLOY.md）"
