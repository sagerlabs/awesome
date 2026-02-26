#!/bin/bash
# 创建 PR 脚本

BRANCH_NAME=${1:-"feature/$(date +%Y%m%d-%H%M%S)"}
BASE_BRANCH=${2:-"main"}
PR_TITLE=${3:-"Feature Update"}
PR_BODY=${4:-"Automated PR creation"}

# 创建分支
git checkout -b "$BRANCH_NAME"

# 提示用户修改
echo "请修改文件后继续..."
read -p "修改完成后按 Enter 继续..."

# 提交
git add .
git commit -m "$PR_TITLE"
git push -u origin "$BRANCH_NAME"

# 使用 gh CLI 创建 PR
if command -v gh &> /dev/null; then
  gh pr create \
    --base "$BASE_BRANCH" \
    --head "$BRANCH_NAME" \
    --title "$PR_TITLE" \
    --body "$PR_BODY"
  echo "PR 创建成功！"
else
  echo "gh CLI 未安装，请手动创建 PR"
  echo "分支: $BRANCH_NAME"
fi
