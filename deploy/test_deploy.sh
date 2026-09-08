#!/usr/bin/env bash
# test_deploy.sh：deploy.sh 在沙箱中的桩测试（无真实 ssh/scp/curl）。
#
# 用法：bash deploy/test_deploy.sh
#
# 不连接生产服务器，PATH 前置桩目录覆盖 scp/ssh/curl/wget/ss/systemctl/sudo，
# 每个 case 验证一个失败/成功路径的退出码与文案。
set -uo pipefail

# ── 测试环境 ──────────────────────────────────────────────────────────────────
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TESTDIR="$(mktemp -d -t tft-deploy-test.XXXXXX)"
trap 'rm -rf "$TESTDIR" "$STUBDIR"' EXIT
STUBDIR="$TESTDIR/stubs"
mkdir -p "$STUBDIR"

# 准备一个真实存在的假 tarball（deploy.sh 只看 -f 存在性）
FAKE_TARBALL="$TESTDIR/tft-copilot-test.tar.gz"
echo "fake" > "$FAKE_TARBALL"

# ── 桩生成器 ──────────────────────────────────────────────────────────────────
# 用文件记录被调用次数与参数，case 结尾断言
make_stub() {
    local name="$1"
    local log="$TESTDIR/${name}.log"
    local mode="${2:-success}"   # success | fail | sleep

    cat > "$STUBDIR/$name" <<EOF
#!/usr/bin/env bash
echo "\$@" >> "$log"
case "$mode" in
    success) exit 0 ;;
    fail)    exit 1 ;;
    sleep)   sleep 999 ;;
esac
EOF
    chmod +x "$STUBDIR/$name"
    : > "$log"
}

# 让 tar 走 GNU 行为（在 macOS 上可能叫 gtar）
if command -v gtar >/dev/null 2>&1; then
    ln -sf "$(command -v gtar)" "$STUBDIR/tar"
else
    ln -sf "$(command -v tar)" "$STUBDIR/tar"
fi

# ── 运行 deploy.sh 的辅助函数 ──────────────────────────────────────────────────
# 用 env 清空可能干扰的代理变量，把桩目录放到 PATH 最前
run_deploy() {
    local tarball="$1"
    local server="$2"
    local real_path="/usr/bin:/bin:/usr/sbin:/sbin"
    if command -v gtar >/dev/null 2>&1; then
        real_path="$(dirname "$(command -v gtar)"):$real_path"
    fi
    env -i \
        PATH="$STUBDIR:$real_path" \
        HOME="$TESTDIR" \
        TERM="dumb" \
        bash "$ROOT/deploy/deploy.sh" "$server" "$tarball" \
        2>&1
    return $?
}

# Case 4 专用：完全屏蔽 /usr/bin 等，仅留桩与必需运行时
# 必需：BASH 内置的 dirname/sed/grep 等已内置；tar 由 stub 提供
run_deploy_no_real_tools() {
    local tarball="$1"
    local server="$2"
    local bash_abs
    bash_abs="$(command -v bash)"
    env -i \
        PATH="$STUBDIR" \
        HOME="$TESTDIR" \
        TERM="dumb" \
        "$bash_abs" "$ROOT/deploy/deploy.sh" "$server" "$tarball" \
        2>&1
    return $?
}

# ── 用例定义 ──────────────────────────────────────────────────────────────────
PASS=0
FAIL=0
assert_exit() {
    local expected="$1"; local actual="$2"; local name="$3"; local output="$4"
    if [ "$expected" = "$actual" ]; then
        printf "  \033[32m✓\033[0m %s (exit=%d)\n" "$name" "$actual"
        PASS=$((PASS+1))
    else
        printf "  \033[31m✗\033[0m %s (expected %d, got %d)\n" "$name" "$expected" "$actual"
        echo "    output: $output" | head -10
        FAIL=$((FAIL+1))
    fi
}

assert_output_contains() {
    local needle="$1"; local output="$2"; local name="$3"
    if echo "$output" | grep -qF "$needle"; then
        PASS=$((PASS+1))
    else
        printf "  \033[31m✗\033[0m %s (expected output to contain: %s)\n" "$name" "$needle"
        echo "    output: $output" | head -10
        FAIL=$((FAIL+1))
    fi
}

# 每个 case 前重置桩
reset_stubs() {
    rm -rf "$STUBDIR"
    mkdir -p "$STUBDIR"
    if command -v gtar >/dev/null 2>&1; then ln -sf "$(command -v gtar)" "$STUBDIR/tar"; else ln -sf "$(command -v tar)" "$STUBDIR/tar"; fi
}

# ── Case 1: 缺 SERVER 参数 → exit 1 ──────────────────────────────────────────
echo "Case 1: 缺 SERVER 参数"
reset_stubs
make_stub scp success
make_stub ssh success
make_stub curl success
out=$(run_deploy "$FAKE_TARBALL" "" 2>&1) && actual=0 || actual=$?
assert_exit 1 "$actual" "缺 SERVER 退出 1" "$out"
assert_output_contains "缺少 SERVER" "$out" "缺 SERVER 错误文案"

# ── Case 2: 缺 TARBALL 参数 → exit 1 ─────────────────────────────────────────
echo ""
echo "Case 2: 缺 TARBALL 参数"
reset_stubs
make_stub scp success
make_stub ssh success
make_stub curl success
out=$(run_deploy "" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 1 "$actual" "缺 TARBALL 退出 1" "$out"
assert_output_contains "缺少 TARBALL" "$out" "缺 TARBALL 错误文案"

# ── Case 3: tarball 不存在 → exit 2 ──────────────────────────────────────────
echo ""
echo "Case 3: tarball 不存在"
reset_stubs
make_stub scp success
make_stub ssh success
make_stub curl success
out=$(run_deploy "/nonexistent/foo.tar.gz" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 2 "$actual" "tarball 不存在退出 2" "$out"
assert_output_contains "找不到 tarball" "$out" "tarball 不存在错误文案"

# ── Case 4: 缺 scp 工具 → exit 3 ─────────────────────────────────────────────
echo ""
echo "Case 4: 缺 scp 工具"
reset_stubs
make_stub ssh success
make_stub curl success
make_stub wget success
# scp 不创建；用 no_real_tools 模式确保 deploy.sh 看到 PATH 中无 scp
out=$(run_deploy_no_real_tools "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 3 "$actual" "缺 scp 退出 3" "$out"
assert_output_contains "缺少工具" "$out" "缺工具错误文案"

# ── Case 5: scp 失败 → exit 4（不打印"部署完成"） ──────────────────────────
echo ""
echo "Case 5: scp 失败"
reset_stubs
make_stub scp fail
make_stub ssh success
make_stub curl success
out=$(run_deploy "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 4 "$actual" "scp 失败退出 4" "$out"
if echo "$out" | grep -q "部署成功"; then
    echo "  ✗ 失败路径不应打印'部署成功'"
    FAIL=$((FAIL+1))
else
    echo "  ✓ 失败路径未打印'部署成功'"
    PASS=$((PASS+1))
fi
if echo "$out" | grep -q "scp 失败"; then
    echo "  ✓ 失败路径打印了 scp 错误原因"
    PASS=$((PASS+1))
else
    echo "  ✗ 失败路径应打印 scp 错误原因"
    FAIL=$((FAIL+1))
fi

# ── Case 6: 远端 ssh tar 失败 → exit 5 ──────────────────────────────────────
echo ""
echo "Case 6: 远端解压失败"
reset_stubs
make_stub scp success
# 让 ssh 第一次（health-check 段不需要 ssh）走 fail；之前的上传 step 后 install 段会调 ssh
# 但 deploy.sh 中 ssh 在 scp 之后第一次出现。直接让它 fail，scp 也会
# 受影响：把 scp 设为 success，ssh 在 tar 步骤 fail：
# 方案：替换 ssh，让它在包含 "tar -xzf" 的命令上返回非零
cat > "$STUBDIR/ssh" <<'EOF'
#!/usr/bin/env bash
echo "$@" >> "$STUBDIR_SSH_LOG"
case "$*" in
    *tar*-xzf*|*tar*test*) exit 1 ;;
    *) exit 0 ;;
esac
EOF
chmod +x "$STUBDIR/ssh"
STUBDIR_SSH_LOG="$TESTDIR/ssh.log"
: > "$STUBDIR_SSH_LOG"
make_stub curl success
out=$(run_deploy "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 5 "$actual" "远端解压失败退出 5" "$out"
if echo "$out" | grep -q "远端解压失败"; then
    PASS=$((PASS+1))
else
    echo "  ✗ 应打印'远端解压失败'"
    FAIL=$((FAIL+1))
fi

# ── Case 7: 远端 install 失败 → exit 6 ─────────────────────────────────────
echo ""
echo "Case 7: 远端 install 失败"
reset_stubs
make_stub scp success
# ssh：仅在 install.sh 步骤失败；step 2 (tar -xzf) 仍要成功
# step 2 末尾是 "test -f ~/tft-copilot-release/install.sh"，不含 sudo
# step 3 是 "PORT='..' sudo -E ./install.sh"，含 sudo -E ./install.sh
cat > "$STUBDIR/ssh" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "$STUBDIR_SSH_LOG"
case "$*" in
    *sudo*install.sh*) exit 7 ;;  # 仅 install 步骤失败
    *) exit 0 ;;
esac
EOF
chmod +x "$STUBDIR/ssh"
STUBDIR_SSH_LOG="$TESTDIR/ssh.log"
: > "$STUBDIR_SSH_LOG"
make_stub curl success
out=$(run_deploy "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 6 "$actual" "远端 install 失败退出 6" "$out"
if echo "$out" | grep -q "install.sh 失败"; then
    PASS=$((PASS+1))
else
    echo "  ✗ 应打印'install.sh 失败'"
    FAIL=$((FAIL+1))
fi

# ── Case 8: 健康检查失败 → exit 7 ───────────────────────────────────────────
echo ""
echo "Case 8: 健康检查失败"
reset_stubs
make_stub scp success
cat > "$STUBDIR/ssh" <<'EOF'
#!/usr/bin/env bash
echo "$@" >> "$STUBDIR_SSH_LOG"
exit 0
EOF
chmod +x "$STUBDIR/ssh"
STUBDIR_SSH_LOG="$TESTDIR/ssh.log"
: > "$STUBDIR_SSH_LOG"
make_stub curl fail
out=$(run_deploy "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 7 "$actual" "健康检查失败退出 7" "$out"
if echo "$out" | grep -q "健康检查未通过"; then
    PASS=$((PASS+1))
else
    echo "  ✗ 应打印'健康检查未通过'"
    FAIL=$((FAIL+1))
fi

# ── Case 9: 成功路径 → exit 0 ──────────────────────────────────────────────
echo ""
echo "Case 9: 成功路径"
reset_stubs
make_stub scp success
cat > "$STUBDIR/ssh" <<'EOF'
#!/usr/bin/env bash
echo "$@" >> "$STUBDIR_SSH_LOG"
exit 0
EOF
chmod +x "$STUBDIR/ssh"
STUBDIR_SSH_LOG="$TESTDIR/ssh.log"
: > "$STUBDIR_SSH_LOG"
# 健康检查用 curl 桩：返回包含 comp_count 的 JSON
cat > "$STUBDIR/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"status":"ok","comp_count":26,"version":"v0.0.0-dev"}'
exit 0
EOF
chmod +x "$STUBDIR/curl"
out=$(run_deploy "$FAKE_TARBALL" "root@1.2.3.4" 2>&1) && actual=0 || actual=$?
assert_exit 0 "$actual" "成功路径退出 0" "$out"
if echo "$out" | grep -q "部署成功"; then
    PASS=$((PASS+1))
else
    echo "  ✗ 成功路径应打印'部署成功'"
    FAIL=$((FAIL+1))
fi
if echo "$out" | grep -q "comp_count=26"; then
    PASS=$((PASS+1))
else
    echo "  ✗ 应展示 comp_count=26"
    FAIL=$((FAIL+1))
fi

# ── Case 10: 个人 Python 路径已被移除（无绝对路径） ─────────────────────────
echo ""
echo "Case 10: 无个人 Python 绝对路径"
if grep -qE "/Users/[a-z]+/\.workbuddy" "$ROOT/deploy/deploy.sh"; then
    echo "  ✗ deploy.sh 仍含个人绝对路径"
    FAIL=$((FAIL+1))
else
    echo "  ✓ deploy.sh 不含个人绝对路径"
    PASS=$((PASS+1))
fi
if grep -qE "/Users/[a-z]+/\.workbuddy" "$ROOT/deploy/publish.sh"; then
    echo "  ✗ publish.sh 仍含个人绝对路径"
    FAIL=$((FAIL+1))
else
    echo "  ✓ publish.sh 不含个人绝对路径"
    PASS=$((PASS+1))
fi

# ── Case 11: 不再硬编码旧包名 ───────────────────────────────────────────────
echo ""
echo "Case 11: 不硬编码旧包名"
if grep -qE "v0\.0\.0-dev-20260901" "$ROOT/deploy/deploy.sh"; then
    echo "  ✗ deploy.sh 仍硬编码 20260901 包名"
    FAIL=$((FAIL+1))
else
    echo "  ✓ deploy.sh 无硬编码包名"
    PASS=$((PASS+1))
fi

# ── Case 12: publish.sh 不再用 ls newest 误选旧包 ────────────────────────────
echo ""
echo "Case 12: publish.sh 不依赖 ls newest"
if grep -qE "ls -t dist/" "$ROOT/deploy/publish.sh"; then
    echo "  ✗ publish.sh 仍用 'ls -t' 选最新包"
    FAIL=$((FAIL+1))
else
    echo "  ✓ publish.sh 从 build-release.sh 输出读取 TARBALL"
    PASS=$((PASS+1))
fi

# ── Case 13: install.sh 接受并同步 PORT ──────────────────────────────────────
echo ""
echo "Case 13: install.sh 接受 PORT env var"
# 验证 install.sh 既读取 PORT，又把它写入 .env
if grep -qE 'PORT=.*\$\{PORT:-' "$ROOT/deploy/install.sh" \
   && grep -qE 'sed.*PORT=' "$ROOT/deploy/install.sh" \
   && grep -qE 'grep.*PORT=' "$ROOT/deploy/install.sh"; then
    echo "  ✓ install.sh 同步 PORT 到 .env"
    PASS=$((PASS+1))
else
    echo "  ✗ install.sh 未同步 PORT 到 .env"
    FAIL=$((FAIL+1))
fi

# ── Case 14: 验证 sed 替换 PORT 行的实际行为（用临时文件模拟 .env） ──────────
echo ""
echo "Case 14: PORT 同步 sed 替换实际生效"
TMPENV="$(mktemp -t tft-env.XXXXXX)"
cat > "$TMPENV" <<'ENVEOF'
# 已有配置
PORT=8080
LLM_PROVIDER=deepseek
OPENAI_API_KEY=sk-actual-real-key-not-placeholder
ENVEOF
# 跑 install.sh 里那一段 sed 命令
PORT=9000
sed -i.bak -E "s|^[[:space:]]*PORT=.*|PORT=${PORT}|" "$TMPENV" && rm -f "$TMPENV.bak"
new_port=$(grep -E '^[[:space:]]*PORT=' "$TMPENV" | head -1 | sed -E 's/.*PORT=//')
if [ "$new_port" = "9000" ]; then
    echo "  ✓ sed 替换后 PORT=9000"
    PASS=$((PASS+1))
else
    echo "  ✗ sed 替换后 PORT=$new_port（期望 9000）"
    FAIL=$((FAIL+1))
fi
# 验证其他字段没被破坏
if grep -q "LLM_PROVIDER=deepseek" "$TMPENV" && grep -q "OPENAI_API_KEY=sk-actual-real-key" "$TMPENV"; then
    echo "  ✓ 其他字段未被破坏"
    PASS=$((PASS+1))
else
    echo "  ✗ sed 误伤了其他字段"
    FAIL=$((FAIL+1))
fi
rm -f "$TMPENV"

# ── 汇总 ──────────────────────────────────────────────────────────────────────
echo ""
echo "============================================================"
echo " Test summary: PASS=$PASS  FAIL=$FAIL"
echo "============================================================"
[ "$FAIL" = 0 ]
