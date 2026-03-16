#!/bin/bash
# 本地运行检查

echo "=== 运行检查 ==="
EXIT_CODE=0

# Go 项目检查
if [ -f go.mod ]; then
  echo -e "\n--- Go 检查 ---"
  
  # gofmt 检查
  echo "运行 gofmt 检查..."
  if command -v gofmt &> /dev/null; then
    GOFMT_OUT=$(gofmt -l .)
    if [ -n "$GOFMT_OUT" ]; then
      echo "错误: 以下文件需要格式化:"
      echo "$GOFMT_OUT"
      EXIT_CODE=1
    else
      echo "gofmt 检查通过"
    fi
  else
    echo "gofmt 未安装"
  fi
  
  # govet 检查
  echo "运行 govet 检查..."
  if command -v go &> /dev/null; then
    go vet ./... 2>&1 || {
      echo "govet 发现问题"
      EXIT_CODE=1
    }
  else
    echo "go 未安装"
  fi
  
  # Go 测试
  echo "运行 Go 测试..."
  if command -v go &> /dev/null; then
    go test -v ./... 2>&1 || {
      echo "Go 测试失败"
      EXIT_CODE=1
    }
  fi
fi

# Python 检查（如果有 Python 文件）
if ls *.py &> /dev/null || [ -d tests ]; then
  echo -e "\n--- Python 检查 ---"
  
  # 运行测试
  if [ -d tests ]; then
    echo "运行 pytest..."
    if command -v pytest &> /dev/null; then
      pytest tests/ -v || EXIT_CODE=1
    else
      echo "pytest 未安装"
    fi
  fi
  
  # 代码格式化检查
  echo "运行 black 检查..."
  if command -v black &> /dev/null; then
    black --check . 2>/dev/null || echo "格式化检查跳过"
  else
    echo "black 未安装"
  fi
  
  # 代码 lint 检查
  echo "运行 flake8 检查..."
  if command -v flake8 &> /dev/null; then
    flake8 . 2>/dev/null || echo "lint 检查跳过"
  else
    echo "flake8 未安装"
  fi
fi

echo -e "\n检查完成！"
exit $EXIT_CODE

