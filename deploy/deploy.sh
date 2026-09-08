#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# TFT Copilot 远端一键部署（上传 + 远端 install + 健康检查）
#
# 用法：
#   ./deploy.sh user@host [path/to/tarball.tar.gz]
#   TARBALL=path/to.tar.gz ./deploy.sh user@host
#   PORT=9000 ./deploy.sh user@host path/to/tarball.tar.gz
#
# 退出码：
#   0   部署成功，健康检查通过
#   1   用法错误 / 参数缺失
#   2   本机缺包（找不到 TARBALL）
#   3   本机缺 ssh/scp/curl 等前置工具
#   4   上传失败
#   5   远端解压失败
#   6   远端 install 失败
#   7   健康检查失败（远端 install 已成功，但服务没起来）
#
# 设计原则：
#   - TARBALL 来自参数或环境变量；禁止硬编码到日期
#   - PORT 同时作用于远端服务端口与本机健康检查端口
#   - 任意步骤失败立刻 exit 非零，禁止"部署完成"等成功文案
#   - 临时文件使用 mktemp 任务独立目录
#   - 避免个人 Python 绝对路径
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

readonly SCRIPT_NAME="$(basename "$0")"
readonly USAGE="用法: $SCRIPT_NAME user@host [path/to/tarball.tar.gz]
  示例: $SCRIPT_NAME root@1.2.3.4
        PORT=9000 TARBALL=dist/tft-copilot-v1.0.1-20260908.tar.gz $SCRIPT_NAME root@1.2.3.4"

# ── 参数解析 ──────────────────────────────────────────────────────────────────
SERVER="${1:-${SERVER:-}}"
shift || true
TARBALL="${1:-${TARBALL:-}}"

PORT="${PORT:-8080}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-15}"  # 健康检查总等待秒数
HEALTH_INTERVAL="${HEALTH_INTERVAL:-1}" # 每次重试间隔

if [ -z "$SERVER" ]; then
    echo "❌ 缺少 SERVER"
    echo "$USAGE"
    exit 1
fi

if [ -z "$TARBALL" ]; then
    echo "❌ 缺少 TARBALL（位置参数 2 或环境变量）"
    echo "$USAGE"
    exit 1
fi

# ── 前置工具检查 ──────────────────────────────────────────────────────────────
missing_tools=()
for tool in ssh scp; do
    command -v "$tool" >/dev/null 2>&1 || missing_tools+=("$tool")
done
# 健康检查至少要有一个 HTTP 客户端
if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
    missing_tools+=("curl 或 wget")
fi
if [ ${#missing_tools[@]} -gt 0 ]; then
    echo "❌ 本机缺少工具: ${missing_tools[*]}"
    echo "   解决: brew install ${missing_tools[*]} 或 apt install ${missing_tools[*]}"
    exit 3
fi

# ── TARBALL 校验 ──────────────────────────────────────────────────────────────
if [ ! -f "$TARBALL" ]; then
    echo "❌ 找不到 tarball: $TARBALL"
    echo "   构建: make release（输出在 dist/）或 ./deploy/build-release.sh"
    exit 2
fi
TARBALL="$(cd "$(dirname "$TARBALL")" && pwd)/$(basename "$TARBALL")"   # 绝对路径
BASENAME="$(basename "$TARBALL")"
PKG="${BASENAME%.tar.gz}"

HOST="${SERVER#*@}"

# ── 临时工作目录（任务独立，不污染 /tmp） ───────────────────────────────────
WORK_DIR="$(mktemp -d -t tft-deploy.XXXXXX)"
trap 'rm -rf "$WORK_DIR"' EXIT

http_get() {
    # 优先 curl，回退 wget；超时 5s
    if command -v curl >/dev/null 2>&1; then
        curl -fsS --max-time 5 "$1"
    else
        wget -qO- --timeout=5 "$1"
    fi
}

echo "============================================================"
echo " TFT Copilot 部署"
echo "   目标:   $SERVER"
echo "   服务端口: $PORT"
echo "   包:     $TARBALL ($(du -h "$TARBALL" | cut -f1))"
echo "   健康检查: http://$HOST:$PORT/v1/tft/health (timeout ${HEALTH_TIMEOUT}s)"
echo "============================================================"

# ── 1/4 上传 ─────────────────────────────────────────────────────────────────
echo ""
echo "[1/4] 上传 tarball → $SERVER:~/"
if ! scp "$TARBALL" "$SERVER:~/"; then
    echo "❌ scp 失败，请检查 ssh 免密、磁盘空间、scp/sshd 版本"
    exit 4
fi
echo "    ✅ 上传完成"

# ── 2/4 远端解压 ─────────────────────────────────────────────────────────────
echo ""
echo "[2/4] 远端解压"
if ! ssh "$SERVER" "set -euo pipefail; tar -xzf ~/${BASENAME} -C ~ && test -f ~/tft-copilot-release/install.sh"; then
    echo "❌ 远端解压失败，请检查磁盘、tar 版本、SSH 网络"
    exit 5
fi
echo "    ✅ 解压完成: ~/tft-copilot-release/"

# ── 3/4 远端 install ─────────────────────────────────────────────────────────
echo ""
echo "[3/4] 远端安装（install.sh 行为：保留 /opt 现有 .env；原子替换二进制）"
echo "      ⚠️  需要 sudo；.env 优先级: /opt/tft-copilot/.env > 包内 > 模板"
echo "      服务端口通过 PORT=$PORT 传给 install.sh"
if ! ssh "$SERVER" "cd ~/tft-copilot-release && PORT='$PORT' sudo -E ./install.sh"; then
    echo "❌ 远端 install.sh 失败，请用以下命令排查："
    echo "   ssh $SERVER 'journalctl -u tft-copilot -n 50 --no-pager'"
    echo "   ssh $SERVER 'ls -la /opt/tft-copilot/'"
    exit 6
fi
echo "    ✅ install.sh 完成"

# ── 4/4 健康检查 ─────────────────────────────────────────────────────────────
echo ""
echo "[4/4] 健康检查 http://$HOST:$PORT/v1/tft/health"
HEALTH_RESP="$WORK_DIR/health.json"
ok=""
deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    if http_get "http://$HOST:$PORT/v1/tft/health" > "$HEALTH_RESP" 2>/dev/null; then
        if grep -q "comp_count" "$HEALTH_RESP"; then
            ok=1
            break
        fi
    fi
    sleep "$HEALTH_INTERVAL"
done

if [ -z "$ok" ]; then
    echo "❌ 健康检查未通过（${HEALTH_TIMEOUT}s 内未拿到有效响应）"
    echo "   排查命令:"
    echo "     ssh $SERVER 'systemctl status tft-copilot --no-pager -l | tail -20'"
    echo "     ssh $SERVER 'cat /opt/tft-copilot/.env | grep -E \"^(PORT|LLM_PROVIDER)\"'"
    echo "     curl http://$HOST:$PORT/v1/tft/health"
    exit 7
fi

# ── 解析健康响应，展示关键字段（不依赖外部 Python） ─────────────────────────
echo "    ✅ 服务正常，关键字段:"
status=$(grep -o '"status":"[^"]*"' "$HEALTH_RESP" | head -1 | sed 's/"status":"//;s/"$//')
comp_count=$(grep -o '"comp_count":[0-9]*' "$HEALTH_RESP" | head -1 | sed 's/"comp_count"://')
version=$(grep -o '"version":"[^"]*"' "$HEALTH_RESP" | head -1 | sed 's/"version":"//;s/"$//')
echo "      status=$status  comp_count=${comp_count:-?}  version=${version:-?}"

echo ""
echo "============================================================"
echo " 部署成功。"
echo "============================================================"
