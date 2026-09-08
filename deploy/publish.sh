#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# TFT Copilot 一键打包+发布（在开发机上执行）
#
# 流程：build-release.sh（打包）→ scp 上传 → 远端 install → 远端健康检查
#
# 退出码：
#   0   发布成功
#   1   用法错误
#   2   远端 ssh 不通
#   3   本机构建失败
#   4   构建产物缺失
#   5   上传失败
#   6   远端 install 失败
#   7   健康检查失败
#
# 用法：
#   ./deploy/publish.sh user@host
#   make publish SERVER=root@1.2.3.4
#   PORT=9000 ./deploy/publish.sh user@host
#
# 前置：
#   1. ssh 免密（ssh-copy-id）
#   2. 远端用户可 sudo（建议直接用 root）
#   3. 首次部署需先在服务器上配置 /opt/tft-copilot/.env（LLM API Key），
#      之后每次 publish 全自动（.env 会被保留）
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SERVER="${1:-${SERVER:-}}"
PORT="${PORT:-8080}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-15}"

if [ -z "$SERVER" ]; then
    echo "❌ 用法: $0 user@host（例: $0 root@1.2.3.4）"
    exit 1
fi

# ── 前置工具 ──────────────────────────────────────────────────────────────────
for tool in ssh scp tar; do
    command -v "$tool" >/dev/null 2>&1 || {
        echo "❌ 缺少工具: $tool"
        exit 1
    }
done
if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    echo "❌ 缺少 curl 或 wget（健康检查需要）"
    exit 1
fi

# ── 远端连通性 ────────────────────────────────────────────────────────────────
echo "🔌 检查 ssh 连通性 ${SERVER} ..."
if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER" true; then
    echo "❌ 无法 ssh 到 ${SERVER}（先配置免密: ssh-copy-id ${SERVER}）"
    exit 2
fi

cd "$(dirname "$0")/.."
ROOT=$(pwd)

# ── 1. 打包（让 build-release.sh 自己决定 TARBALL 路径并打印） ───────────────
echo ""
echo "📦 1/4 打包 ..."
TARBALL="$(bash deploy/build-release.sh | tee /dev/stderr | sed -n 's/^✅ 打包完成: \(dist\/tft-copilot-.*\.tar\.gz\) .*/\1/p' | head -1)"
if [ -z "$TARBALL" ] || [ ! -f "$TARBALL" ]; then
    echo "❌ 构建失败或未生成 dist/ 下的 tarball"
    exit 3
fi
PKG=$(basename "$TARBALL")
echo "🚀 发布 ${PKG} → ${SERVER}"

# ── 2. 上传 ──────────────────────────────────────────────────────────────────
echo ""
echo "📤 2/4 上传 ..."
if ! scp "$TARBALL" "${SERVER}:~/"; then
    echo "❌ scp 失败"
    exit 5
fi

# ── 3. 远端 install ─────────────────────────────────────────────────────────
echo ""
echo "⚙️  3/4 远端 install ..."
# shellcheck disable=SC2087
if ! ssh "$SERVER" bash -s <<REMOTE
set -euo pipefail
cd ~
rm -rf tft-copilot-release
tar -xzf "${PKG}"
cd tft-copilot-release
PORT='${PORT}' sudo -E ./install.sh
REMOTE
then
    echo "❌ 远端 install.sh 失败"
    exit 6
fi

# ── 4. 健康检查（远端 curl，避免本机到远端端口不通误判） ────────────────────
echo ""
echo "🔍 4/4 健康检查 http://127.0.0.1:${PORT}/v1/tft/health (timeout ${HEALTH_TIMEOUT}s) ..."

http_get() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsS --max-time 5 "$1"
    else
        wget -qO- --timeout=5 "$1"
    fi
}

ok=""
deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    if http_get "http://127.0.0.1:${PORT}/v1/tft/health" 2>/dev/null | grep -q "comp_count"; then
        ok=1
        break
    fi
    sleep 1
done

if [ -z "$ok" ]; then
    echo "❌ 健康检查未通过（${HEALTH_TIMEOUT}s 内未拿到有效响应）"
    echo "   排查: ssh ${SERVER} 'systemctl status tft-copilot --no-pager -l | tail -20'"
    exit 7
fi

echo ""
echo "✅ 发布成功！"
echo "   页面: http://<服务器IP>:${PORT}/"
echo "   验证: ssh ${SERVER} 'curl -s localhost:${PORT}/v1/tft/health'"
