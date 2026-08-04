#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# TFT Copilot 服务器一键安装脚本（在云服务器上执行，release tarball 解压后）
#
# 行为：
#   1. 识别 CPU 架构（x86_64 → amd64，aarch64 → arm64）
#   2. 安装二进制 + 运行时数据到 /opt/tft-copilot
#   3. 环境变量：/opt/tft-copilot/.env 不存在时从 .env.example 复制
#   4. 有 systemd → 注册并启动服务；无 systemd → nohup 后台启动
#
# 用法：sudo ./install.sh        # 已配置好 .env 后直接启动
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

INSTALL_DIR=/opt/tft-copilot
SERVICE_NAME=tft-copilot
SRC_DIR=$(cd "$(dirname "$0")" && pwd)

[ "$(id -u)" = "0" ] || { echo "❌ 请用 root 或 sudo 执行: sudo ./install.sh"; exit 1; }

# ── 架构识别 ──────────────────────────────────────────────────────────────────
case "$(uname -m)" in
    x86_64)          ARCH=amd64 ;;
    aarch64|arm64)   ARCH=arm64 ;;
    *) echo "❌ 不支持的架构: $(uname -m)（release 仅含 amd64/arm64）"; exit 1 ;;
esac
BINARY="${SRC_DIR}/bin/tft-copilot-linux-${ARCH}"
[ -f "$BINARY" ] || { echo "❌ 缺少二进制: $BINARY"; exit 1; }
echo "🔎 服务器架构: linux/${ARCH}"

# ── 安装文件 ──────────────────────────────────────────────────────────────────
echo "📂 安装到 ${INSTALL_DIR} ..."
mkdir -p "$INSTALL_DIR"
# 原子替换二进制：先拷到临时文件再 mv，避免服务运行时覆盖报 "Text file busy"
cp "$BINARY" "$INSTALL_DIR/tft-copilot.new"
chmod +x "$INSTALL_DIR/tft-copilot.new"
mv -f "$INSTALL_DIR/tft-copilot.new" "$INSTALL_DIR/tft-copilot"

mkdir -p "$INSTALL_DIR/metadata/tft-meta"
rm -rf "$INSTALL_DIR/metadata/tft-meta/data"
cp -R "$SRC_DIR/metadata/tft-meta/data" "$INSTALL_DIR/metadata/tft-meta/data"

mkdir -p "$INSTALL_DIR/tft/knowledge"
rm -rf "$INSTALL_DIR/tft/knowledge/data"
cp -R "$SRC_DIR/tft/knowledge/data" "$INSTALL_DIR/tft/knowledge/data"

# ── 环境变量 ──────────────────────────────────────────────────────────────────
# 优先级：已安装的 /opt .env（升级时保留） > 发布包内用户已配置的 .env > 模板
if [ ! -f "$INSTALL_DIR/.env" ]; then
    if [ -f "$SRC_DIR/.env" ]; then
        cp "$SRC_DIR/.env" "$INSTALL_DIR/.env"
        echo "🔑 使用发布包内已配置的 .env"
    else
        cp "$SRC_DIR/.env.example" "$INSTALL_DIR/.env"
        echo "⚠️  已生成 ${INSTALL_DIR}/.env（模板），请编辑填入 LLM API Key 后重新执行本脚本"
        echo "    vi ${INSTALL_DIR}/.env && sudo ./install.sh"
        exit 0
    fi
else
    echo "🔑 使用已存在的 ${INSTALL_DIR}/.env（发布包内的 .env 不覆盖；改配置请直接编辑它）"
fi
chmod 600 "$INSTALL_DIR/.env"   # 含 API Key，仅属主可读写
# 校验 key 已配置（拒绝占位符）
set -a; . "$INSTALL_DIR/.env"; set +a
case "${LLM_PROVIDER:-openai}" in
    openai|deepseek)
        [ -n "${OPENAI_API_KEY:-}" ] && [[ "${OPENAI_API_KEY}" != *"在这里"* ]] \
            || { echo "❌ .env 中 OPENAI_API_KEY 未配置（仍为占位符）"; exit 1; } ;;
    ark)
        [ -n "${ARK_API_KEY:-}" ] && [ -n "${ARK_MODEL_ID:-}" ] \
            || { echo "❌ .env 中 ARK_API_KEY / ARK_MODEL_ID 为空"; exit 1; } ;;
esac

# ── 端口预检：目标端口被非本服务进程占用时提前预警 ────────────────────────────
PORT="${PORT:-8080}"
if command -v ss > /dev/null 2>&1; then
    occupier=$(ss -tlnp 2>/dev/null | grep ":${PORT} " | grep -v "tft-copilot" | head -1 || true)
    if [ -n "$occupier" ]; then
        echo "⚠️  端口 ${PORT} 已被其他进程占用："
        echo "    $occupier"
        echo "    服务可能启动失败（bind 冲突）。如需换端口：vi ${INSTALL_DIR}/.env 修改 PORT"
    fi
fi

# ── 启动 ──────────────────────────────────────────────────────────────────────
if command -v systemctl > /dev/null 2>&1 && [ -d /run/systemd/system ]; then
    echo "⚙️  注册 systemd 服务 ..."
    cp "$SRC_DIR/tft-copilot.service" "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" > /dev/null 2>&1 || true
    systemctl restart "$SERVICE_NAME"
    sleep 2
    systemctl --no-pager --full status "$SERVICE_NAME" | head -8 || true
else
    echo "⚙️  未检测到 systemd，使用 nohup 后台启动 ..."
    pkill -f "$INSTALL_DIR/tft-copilot" 2>/dev/null || true
    cd "$INSTALL_DIR"
    set -a; . ./.env; set +a
    nohup ./tft-copilot > "$INSTALL_DIR/tft-copilot.log" 2>&1 &
    echo "   PID: $!  日志: $INSTALL_DIR/tft-copilot.log"
    sleep 2
fi

# ── 健康检查 ──────────────────────────────────────────────────────────────────
# HTTP 工具探测：精简镜像可能没有 curl，回退 wget；都没有则跳过健康检查（不算失败）
HAS_HTTP=1
if command -v curl > /dev/null 2>&1; then
    http_get() { curl -fsS "$1"; }
elif command -v wget > /dev/null 2>&1; then
    http_get() { wget -qO- "$1"; }
else
    HAS_HTTP=""
    http_get() { return 1; }
fi

PORT="${PORT:-8080}"
ok=""
if [ -n "$HAS_HTTP" ]; then
    echo "🔍 等待健康检查 http://127.0.0.1:${PORT}/v1/tft/health ..."
    for _ in $(seq 1 15); do
        # 严格校验：响应必须含 comp_count，避免被同端口的其他服务假阳性
        if http_get "http://127.0.0.1:${PORT}/v1/tft/health" 2>/dev/null | grep -q "comp_count"; then ok=1; break; fi
        sleep 1
    done
fi

if [ -n "$ok" ]; then
    echo ""
    echo "✅ 部署完成！"
    http_get "http://127.0.0.1:${PORT}/v1/tft/health" || true
    echo ""
    echo "   Web 页面: http://<服务器IP>:${PORT}/"
    echo "   运维命令: systemctl status|restart|stop ${SERVICE_NAME}；日志 journalctl -u ${SERVICE_NAME} -f"
    echo "   ⚠️  请在云控制台安全组放行 ${PORT} 端口（或仅放行 80/443 并用 Nginx 反代，见 DEPLOY.md）"
elif [ -z "$HAS_HTTP" ]; then
    echo ""
    echo "✅ 安装完成（未做健康检查：系统缺少 curl/wget）"
    echo "   服务已在运行，安装 curl 后可验证: curl http://127.0.0.1:${PORT}/v1/tft/health"
    echo "   Web 页面: http://<服务器IP>:${PORT}/"
else
    echo "❌ 健康检查未通过，请查看日志: journalctl -u ${SERVICE_NAME} -n 50 --no-pager（或 $INSTALL_DIR/tft-copilot.log）"
    exit 1
fi
