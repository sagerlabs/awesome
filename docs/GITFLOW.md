# GitFlow 工作流指南

本项目使用 GitFlow 作为分支管理工作流。本文档详细介绍如何使用。

## 分支结构

```
main (生产环境)
  ↑
  └── dev (开发环境)
       ↑
       ├── feature/* (功能分支)
       ├── release/* (发布分支)
       └── hotfix/* (热修复分支，从 main 创建)
```

### 主要分支

- **main** - 生产环境分支，只有稳定的发布版本
- **dev** - 开发分支，所有功能合并到这里

### 辅助分支

- **feature/*** - 功能分支，用于开发新功能
- **release/*** - 发布分支，用于准备发布
- **hotfix/*** - 热修复分支，用于紧急修复生产问题

## 快速开始

### 使用 GitFlow 脚本（推荐）

项目提供了便捷的 GitFlow 脚本：

```bash
# 显示帮助
./scripts/git-flow.sh help

# 功能分支
./scripts/git-flow.sh feature start my-feature
./scripts/git-flow.sh feature finish my-feature

# 发布分支
./scripts/git-flow.sh release start 1.0.0
./scripts/git-flow.sh release finish 1.0.0

# 热修复分支
./scripts/git-flow.sh hotfix start 1.0.1
./scripts/git-flow.sh hotfix finish 1.0.1
```

### 使用原生 Git 命令

如果你喜欢使用原生 Git 命令，请看下面的详细说明。

---

## 详细流程

### 1. 功能开发 (Feature)

用于开发新功能。

#### 开始功能开发

```bash
# 从 dev 分支创建功能分支
git checkout dev
git pull origin dev
git checkout -b feature/my-awesome-feature
```

#### 开发功能

```bash
# 编写代码...
# 提交更改
git add .
git commit -m "feat: add awesome feature"

# 推送到远程（可选）
git push -u origin feature/my-awesome-feature
```

#### 完成功能开发

```bash
# 切换到 dev 分支
git checkout dev
git pull origin dev

# 合并功能分支（使用 --no-ff 保留分支历史）
git merge --no-ff feature/my-awesome-feature

# 删除功能分支（可选）
git branch -d feature/my-awesome-feature

# 推送到远程
git push origin dev
```

**或者使用 GitHub PR：**
1. 将 feature 分支推送到远程
2. 创建 PR 到 dev 分支
3. 代码审查通过后合并

---

### 2. 发布 (Release)

用于准备发布新版本。

#### 开始发布

```bash
# 从 dev 分支创建发布分支
git checkout dev
git pull origin dev
git checkout -b release/1.0.0
```

#### 准备发布

在发布分支上进行以下操作：

```bash
# 1. 更新版本号
# 编辑相关文件，更新版本号

# 2. 更新 CHANGELOG.md
# 添加本次发布的变更内容

# 3. 运行测试
make check

# 4. 提交更改
git add .
git commit -m "chore: prepare release 1.0.0"

# 推送到远程
git push -u origin release/1.0.0
```

#### 完成发布

```bash
# 1. 合并到 main
git checkout main
git pull origin main
git merge --no-ff release/1.0.0

# 2. 创建标签
git tag -a v1.0.0 -m "Release version 1.0.0"

# 3. 合并到 dev
git checkout dev
git merge --no-ff release/1.0.0

# 4. 删除发布分支
git branch -d release/1.0.0

# 5. 推送到远程
git push origin main
git push origin dev
git push origin v1.0.0
```

**或者使用 GitHub PR：**
1. 将 release 分支推送到远程
2. 创建 PR 到 main 分支
3. 合并后，手动合并到 dev 并创建标签

---

### 3. 热修复 (Hotfix)

用于紧急修复生产环境的问题。

#### 开始热修复

```bash
# 从 main 分支创建热修复分支
git checkout main
git pull origin main
git checkout -b hotfix/1.0.1
```

#### 修复问题

```bash
# 1. 修复问题
# 编写修复代码...

# 2. 运行测试
make check

# 3. 提交更改
git add .
git commit -m "fix: critical bug fix"

# 4. 推送到远程
git push -u origin hotfix/1.0.1
```

#### 完成热修复

```bash
# 1. 合并到 main
git checkout main
git pull origin main
git merge --no-ff hotfix/1.0.1

# 2. 创建标签
git tag -a v1.0.1 -m "Hotfix version 1.0.1"

# 3. 合并到 dev
git checkout dev
git merge --no-ff hotfix/1.0.1

# 4. 删除热修复分支
git branch -d hotfix/1.0.1

# 5. 推送到远程
git push origin main
git push origin dev
git push origin v1.0.1
```

---

## GitHub Actions 自动化

项目配置了以下自动化工作流：

### PR 检查 (pr-checks.yml)
- 触发条件：PR 到 main、dev 分支
- 功能：自动运行测试、lint 检查

### GitFlow 发布 (gitflow-release.yml)
- 触发条件：release/* 分支推送、PR 合并到 main
- 功能：
  - 发布分支检查
  - 自动创建标签和 GitHub Release

---

## Makefile 命令

```bash
make help    # 显示帮助
make test    # 运行测试
make lint    # 运行 lint
make format  # 格式化代码
make check   # 运行所有检查
```

---

## 最佳实践

1. **提交信息规范**
   - `feat:` - 新功能
   - `fix:` - 修复
   - `docs:` - 文档
   - `style:` - 格式
   - `refactor:` - 重构
   - `test:` - 测试
   - `chore:` - 构建/工具

2. **分支命名**
   - 功能：`feature/描述性名称`
   - 发布：`release/版本号`
   - 热修复：`hotfix/版本号`

3. **使用 --no-ff**
   - 合并时使用 `--no-ff` 保留完整的分支历史

4. **频繁提交**
   - 小步快跑，频繁提交，便于回滚和审查

---

## 常见问题

### Q: 功能分支开发到一半，dev 分支更新了怎么办？
A: 在功能分支上 rebase dev：
```bash
git checkout feature/my-feature
git rebase dev
```

### Q: 发布分支上发现 bug 怎么办？
A: 直接在发布分支上修复，然后一起发布。

### Q: 多个功能可以同时开发吗？
A: 可以！每个功能一个独立的 feature 分支。

### Q: 热修复需要合并到 dev 吗？
A: 是的！热修复也要合并到 dev，确保下一个版本包含修复。

---

## 参考资料

- [GitFlow 原始论文](https://nvie.com/posts/a-successful-git-branching-model/)
- [GitFlow Cheat Sheet](https://danielkummer.github.io/git-flow-cheatsheet/)
