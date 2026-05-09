# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TFT Copilot - Makefile
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# ── 变量 ──────────────────────────────────────────────────────────────────────

BINARY     := tft-copilot
BUILD_DIR  := ./bin
MAIN       := ./main.go

# 数据目录
METADATA_DIR  := ./metadata/tft-meta/data
KNOWLEDGE_DIR := ./tft/knowledge/data
UPDATE_SCRIPT := ./scripts/update_cn_knowledge.py
OPGG_MCP_SCRIPT := ./scripts/update_opgg_mcp_mvp.py
PYTHON        := python3

# Go 工具
GOFMT      := gofmt
GOVET      := go vet
GOLINT     := golangci-lint

# 版本信息（从 git 读取，CI 环境没有 git 时降级为 unknown）
GIT_TAG    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "unknown")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date '+%Y-%m-%d %H:%M:%S')

# 编译时注入版本信息
LDFLAGS := -X 'main.Version=$(GIT_TAG)' \
           -X 'main.GitCommit=$(GIT_COMMIT)' \
           -X 'main.BuildTime=$(BUILD_TIME)'

# ── 默认目标 ──────────────────────────────────────────────────────────────────

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示帮助信息
	@echo ""
	@echo "TFT Copilot - 可用命令："
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

# ── 开发 ──────────────────────────────────────────────────────────────────────

.PHONY: run
run: ## 本地启动服务（需要先配置环境变量）
	go run $(MAIN)

.PHONY: run-deepseek
run-deepseek: ## 使用 DeepSeek 启动（需要设置 OPENAI_API_KEY）
	LLM_PROVIDER=deepseek \
	OPENAI_BASE_URL=https://api.deepseek.com \
	OPENAI_MODEL=deepseek-chat \
	go run $(MAIN)

.PHONY: run-openai
run-openai: ## 使用 OpenAI 启动（需要设置 OPENAI_API_KEY）
	LLM_PROVIDER=openai \
	OPENAI_MODEL=gpt-4o-mini \
	go run $(MAIN)

.PHONY: run-ark
run-ark: ## 使用火山引擎豆包启动（需要设置 ARK_API_KEY 和 ARK_MODEL_ID）
	LLM_PROVIDER=ark \
	go run $(MAIN)

# ── 构建 ──────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## 编译二进制（输出到 ./bin/tft-copilot）
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(MAIN)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(BINARY)"

.PHONY: build-linux
build-linux: ## 交叉编译 Linux amd64（用于部署服务器）
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 \
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(MAIN)
	@echo "✅ Linux 构建完成: $(BUILD_DIR)/$(BINARY)-linux-amd64"

.PHONY: build-mac
build-mac: ## 交叉编译 macOS arm64（Apple Silicon）
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 \
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(MAIN)
	@echo "✅ macOS 构建完成: $(BUILD_DIR)/$(BINARY)-darwin-arm64"

# ── 数据 ──────────────────────────────────────────────────────────────────────

.PHONY: data
data: ## 拉取最新中文数据并拆分到 knowledge（可传 PATCH_NOTE_URL=官方公告链接）
	@echo "📦 开始更新 TFT knowledge 数据..."
	$(PYTHON) $(UPDATE_SCRIPT) $(if $(PATCH_NOTE_URL),--patch-note-url "$(PATCH_NOTE_URL)")
	@echo "✅ knowledge 数据更新完成"

.PHONY: data-local
data-local: ## 使用本地 metadata JSON 重新生成 knowledge（不访问网络）
	@echo "📦 使用本地 metadata 重新生成 knowledge..."
	$(PYTHON) $(UPDATE_SCRIPT) --skip-fetch
	@echo "✅ knowledge 数据生成完成"

.PHONY: data-opgg-mvp
data-opgg-mvp: ## 从 OP.GG MCP 拉取最小 knowledge（默认最多 20 套，可传 LIMIT=20）
	@echo "📦 从 OP.GG MCP 更新最小 TFT knowledge..."
	$(PYTHON) $(OPGG_MCP_SCRIPT) --limit $(if $(LIMIT),$(LIMIT),20)
	@echo "✅ OP.GG MCP MVP knowledge 数据更新完成"

.PHONY: data-opgg-mvp-check
data-opgg-mvp-check: ## 使用 OP.GG MCP 响应样例做 dry-run 校验（可传 INPUT_RESPONSE=/path/response.json）
	@echo "🔍 校验 OP.GG MCP MVP 数据管线..."
	$(PYTHON) $(OPGG_MCP_SCRIPT) --dry-run $(if $(INPUT_RESPONSE),--input-response "$(INPUT_RESPONSE)")
	@echo "✅ OP.GG MCP MVP 数据管线校验完成"

.PHONY: data-check
data-check: ## 检查数据文件是否存在且非空
	@echo "🔍 检查数据文件..."
	@for f in comps_for_agent.json items_priority.json localization.json; do \
		path="$(METADATA_DIR)/$$f"; \
		if [ ! -f "$$path" ]; then \
			echo "❌ 缺少文件: $$path，请先运行 make data"; \
			exit 1; \
		fi; \
		echo "  ✓ $$f"; \
	done
	@for d in champions items team_comps; do \
		path="$(KNOWLEDGE_DIR)/$$d"; \
		if [ ! -d "$$path" ]; then \
			echo "❌ 缺少目录: $$path，请先运行 make data"; \
			exit 1; \
		fi; \
		echo "  ✓ $$d/"; \
	done
	@if [ ! -f "$(KNOWLEDGE_DIR)/aliases.json" ]; then \
		echo "❌ 缺少文件: $(KNOWLEDGE_DIR)/aliases.json"; \
		exit 1; \
	fi
	@echo "  ✓ aliases.json"
	@echo "✅ 数据文件完整"

# ── 测试 ──────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## 运行所有单元测试
	go test ./... -v -count=1

.PHONY: test-tft
test-tft: ## 只运行 tft 包的测试
	go test ./tft/... -v -count=1

.PHONY: test-cover
test-cover: ## 运行测试并生成覆盖率报告
	@mkdir -p $(BUILD_DIR)
	go test ./... -coverprofile=$(BUILD_DIR)/coverage.out
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "✅ 覆盖率报告: $(BUILD_DIR)/coverage.html"

.PHONY: test-api
test-api: ## 用 curl 测试本地接口（需要服务已启动）
	@echo "--- 健康检查 ---"
	curl -s http://localhost:8080/v1/tft/health | jq .
	@echo ""
	@echo "--- 主接口：NLU JSON ---"
	curl -s -X POST http://localhost:8080/v1/tft/nlu \
		-H "Content-Type: application/json" \
		-d '{"input":"当前版本最强的三套阵容是什么？"}' | jq .
	@echo ""
	@echo "--- 主接口：NLU SSE ---"
	curl -X POST http://localhost:8080/v1/tft/nlu/stream \
		-H "Content-Type: application/json" \
		-d '{"input":"剑魔打工强吗"}' \
		--no-buffer

# ── 代码质量 ──────────────────────────────────────────────────────────────────

.PHONY: fmt
fmt: ## 格式化所有 Go 代码
	$(GOFMT) -w .
	@echo "✅ 格式化完成"

.PHONY: vet
vet: ## 运行 go vet 静态检查
	$(GOVET) ./...
	@echo "✅ vet 检查通过"

.PHONY: lint
lint: ## 运行 golangci-lint（需要先安装）
	$(GOLINT) run ./...

.PHONY: tidy
tidy: ## 整理 go.mod 和 go.sum
	go mod tidy
	@echo "✅ go mod tidy 完成"

.PHONY: check
check: fmt vet tidy ## 提交前检查（fmt + vet + tidy）
	@echo "✅ 所有检查通过"

# ── 依赖 ──────────────────────────────────────────────────────────────────────

.PHONY: deps
deps: ## 安装 Go 依赖
	go mod download
	@echo "✅ Go 依赖安装完成"

.PHONY: deps-py
deps-py: ## 安装 Python 爬虫依赖
	$(PYTHON) -m pip install requests
	@echo "✅ Python 依赖安装完成"

.PHONY: deps-all
deps-all: deps deps-py ## 安装所有依赖（Go + Python）

.PHONY: install-lint
install-lint: ## 安装 golangci-lint
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ golangci-lint 安装完成"

# ── 清理 ──────────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## 清理编译产物
	@rm -rf $(BUILD_DIR)
	@echo "✅ 清理完成"

.PHONY: clean-data
clean-data: ## 清理爬取的数据文件（谨慎使用）
	@rm -f $(METADATA_DIR)/comps_for_agent.json \
	        $(METADATA_DIR)/items_priority.json \
	        $(METADATA_DIR)/localization.json \
	        $(METADATA_DIR)/comps_full.json \
	        $(METADATA_DIR)/comps_full_cn.json \
	        $(METADATA_DIR)/items_priority_cn.json
	@echo "✅ 数据文件已清理"

.PHONY: clean-all
clean-all: clean clean-data ## 清理所有产物（含数据文件）

# ── 完整工作流 ─────────────────────────────────────────────────────────────────

.PHONY: setup
setup: deps-all data ## 首次初始化（安装依赖 + 拉取数据）
	@echo ""
	@echo "🎉 初始化完成！接下来："
	@echo "   1. 配置环境变量（LLM_PROVIDER / OPENAI_API_KEY 等）"
	@echo "   2. 运行 make run 启动服务"
	@echo "   3. 运行 make test-api 验证接口"

.PHONY: ci
ci: deps check test ## CI 流水线（依赖 + 检查 + 测试）
