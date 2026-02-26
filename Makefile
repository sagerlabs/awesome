.PHONY: test lint format check pr help

help:  ## 显示帮助
	@awk 'BEGIN {FS = ":.*##"; printf "\n用法:\n  make \033[36m<目标>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

test:  ## 运行测试
	@if [ -d tests ]; then pytest tests/ -v; else echo "未找到 tests 目录"; fi

lint:  ## 运行 lint
	@if command -v flake8 &> /dev/null; then flake8 . 2>/dev/null || true; else echo "flake8 未安装"; fi

format:  ## 格式化代码
	@if command -v black &> /dev/null; then black .; else echo "black 未安装"; fi

check:  ## 运行所有检查
	./scripts/run-checks.sh

pr:  ## 创建 PR
	./scripts/create-pr.sh
