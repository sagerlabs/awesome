# Awesome Project

## PR 工作流

本项目实现了完整的 PR 自动化流程，借鉴了 Kubernetes、Moby、Prometheus 等知名 Go 项目的最佳实践：

1. **创建 PR** - 使用 `gh pr create` 或脚本
2. **自动检查** - PR 创建后自动运行测试和 lint
3. **自动 Review** - AI 辅助的自动 review 评论
4. **人工合并** - 需要人工审核通过后才能合并

## 快速开始

### 安装依赖

```bash
# 安装 GitHub CLI
sudo apt install gh

# 认证
gh auth login
```

### 开发流程

```bash
# 1. 创建功能分支
git checkout -b feature/my-feature

# 2. 开发代码
# ... 修改文件 ...

# 3. 运行本地检查
make check

# 4. 提交
git add .
git commit -m "feat: add my feature"
git push

# 5. 创建 PR
gh pr create --base main --title "feat: add my feature" --body "Description"
```

## GitHub Actions 工作流

### pr-checks.yml - 自动检查
- **Go 检查**: `gofmt`, `govet`, `staticcheck`, `errcheck`
- **Go 测试**: 单元测试 + 覆盖率检查
- **Python 检查**: `pytest`, `flake8`, `black`
- **文档检查**: 检测是否需要更新 README 和 CHANGELOG

### pr-review.yml - 自动 Review
- **PR 大小检测**: 自动标记 size/XS ~ size/XL
- **敏感信息检测**: 检查硬编码密码、token 等
- **敏感文件检测**: 检查 .env、secret 文件
- **智能建议**: 文档更新建议、PR 拆分建议
- **友好互动**: 欢迎贡献者，感谢贡献
- **自动请求修改**: 发现严重问题时自动 REQUEST_CHANGES

## 分支保护

在仓库设置中启用：
- Require a pull request before merging
- Require approvals: 1
- Require status checks to pass: `go-checks`, `python-checks`, `docs-check`

## PR 标签说明

- `size/XS`: < 10 行变更
- `size/S`: 10-50 行变更
- `size/M`: 50-200 行变更
- `size/L`: 200-1000 行变更
- `size/XL`: > 1000 行变更（建议拆分）

## Makefile 命令

```bash
make help    # 显示帮助
make test    # 运行测试
make lint    # 运行 lint
make format  # 格式化代码
make check   # 运行所有检查
make pr      # 创建 PR
```
