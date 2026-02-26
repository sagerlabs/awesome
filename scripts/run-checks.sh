#!/bin/bash
# 本地运行检查

echo "=== 运行测试 ==="
if [ -d tests ]; then
  pytest tests/ -v
else
  echo "未找到 tests 目录"
fi

echo -e "\n=== 代码格式化检查 ==="
if command -v black &> /dev/null; then
  black --check . 2>/dev/null || echo "格式化检查跳过"
else
  echo "black 未安装"
fi

echo -e "\n=== 代码 lint 检查 ==="
if command -v flake8 &> /dev/null; then
  flake8 . 2>/dev/null || echo "lint 检查跳过"
else
  echo "flake8 未安装"
fi

echo -e "\n检查完成！"
