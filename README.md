# Awesome Project

## PR 工作流

本项目实现了完整的 PR 自动化流程：

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

- **pr-checks.yml** - 自动运行测试、lint、格式化检查
- **pr-review.yml** - 自动添加 review 评论

## 分支保护

在仓库设置中启用：
- Require a pull request before merging
- Require approvals: 1
- Require status checks to pass
