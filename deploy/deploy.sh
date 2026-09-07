#!/usr/bin/env bash
# TFT Copilot 一键上传部署脚本（TFTSet18 数据包 + 18.1 公告，2026-09-01）
# 用法:
#   ./deploy.sh root@1.2.3.4              # 默认 PORT=8077
#   PORT=8080 ./deploy.sh root@1.2.3.4    # 自定义端口
# 前置: 本机已 ssh 免密到服务器；远端有 sudo 权限。
set -euo pipefail

SERVER="${1:-${SERVER:-}}"
PORT="${PORT:-8077}"
TARBALL="dist/tft-copilot-v0.0.0-dev-20260901.tar.gz"
BASENAME="$(basename "$TARBALL")"

# 提取 host（去掉 user@ 前缀），用于健康检查
HOST="${SERVER#*@}"

if [ -z "$SERVER" ]; then
  echo "用法: $0 user@IP   例: $0 root@1.2.3.4"
  echo "      PORT=9000 $0 root@1.2.3.4   # 自定义端口"
  exit 1
fi
if [ ! -f "$TARBALL" ]; then
  echo "❌ 找不到 $TARBALL，请先在项目根执行 make release"
  exit 1
fi

echo "============================================================"
echo " TFT Copilot 部署 → $SERVER (健康检查端口 $PORT)"
echo " 包: $TARBALL ($(du -h "$TARBALL" | cut -f1))"
echo "============================================================"

echo ""
echo "[1/4] 上传 tarball → $SERVER:~/"
scp "$TARBALL" "$SERVER:~/"
echo "    ✅ 上传完成"

echo ""
echo "[2/4] 远程解压"
ssh "$SERVER" "tar -xzf ~/$BASENAME -C ~ && test -f ~/tft-copilot-release/install.sh"
echo "    ✅ 解压完成: ~/tft-copilot-release/"

echo ""
echo "[3/4] 远程安装（install.sh 行为：保留 /opt 现有 .env；原子替换二进制+数据；自动 restart）"
echo "      ⚠️  需要 sudo，若提示密码请输入。.env 优先级: /opt/tft-copilot/.env > 包内 > 模板"
ssh -t "$SERVER" 'cd ~/tft-copilot-release && sudo ./install.sh'

echo ""
echo "[4/4] 健康检查 http://$HOST:$PORT/v1/tft/health"
echo "      （服务刚 restart，给 3 秒缓冲）"
sleep 3
if curl -fsS --max-time 8 "http://$HOST:$PORT/v1/tft/health" | tee /tmp/tft_health.json 2>/dev/null; then
  echo ""
  echo "    ✅ 服务正常，关键字段:"
  /Users/cokecola/.workbuddy/binaries/python/versions/3.13.12/bin/python3 -c "import json,sys;d=json.load(open('/tmp/tft_health.json'));print(f\"      status={d.get('status')} comp_count={d.get('comp_count')} version={d.get('version')}\")" 2>/dev/null || cat /tmp/tft_health.json
  rm -f /tmp/tft_health.json
else
  echo "    ⚠️  健康检查未通（可能端口/防火墙/服务还在启动），手动排查:"
  echo "      ssh $SERVER 'systemctl status tft-copilot --no-pager -l | tail -20'"
  echo "      ssh $SERVER 'cat /opt/tft-copilot/.env | grep -E \"^(PORT|LLM_PROVIDER)\"'"
  echo "      curl http://$HOST:$PORT/v1/tft/health"
fi

echo ""
echo "============================================================"
echo " 部署完成。"
echo " 若要验证数据是否为 TFTSet18:"
echo "   ssh $SERVER 'grep tft_set /opt/tft-copilot/metadata/tft-meta/data/comps_full.json | head -1'"
echo " 若 .env 的 LLM 配置不对（404/401），编辑 /opt/tft-copilot/.env 后:"
echo "   ssh $SERVER 'sudo systemctl restart tft-copilot'"
echo "============================================================"
