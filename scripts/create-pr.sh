#!/bin/bash
# 创建 PR 脚本

# 使用方式: ./scripts/create-pr.sh [branch-name] [base-branch] [pr-title] [pr-body]

BRANCH_NAME=${1:-"feature/$(date +%Y%m%d-%H%M%S)"}
BASE_BRANCH=${2:-"dev"}
PR_TITLE=${3:-"Feature Update"}
PR_BODY=${4:-"Automated PR creation"}

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}开始创建 PR...${NC}"

# 检查当前分支是否干净
if [ -n "$(git status --porcelain)" ]; then
  echo -e "${RED}错误: 工作区有未提交的更改，请先提交或 stash${NC}"
  exit 1
fi

# 检查是否已经在目标分支上
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BASE_BRANCH" ]; then
  echo -e "${YELLOW}警告: 当前分支不是 $BASE_BRANCH，切换到 $BASE_BRANCH${NC}"
  git checkout "$BASE_BRANCH"
  git pull origin "$BASE_BRANCH"
fi

# 创建分支
echo -e "${GREEN}创建分支: $BRANCH_NAME${NC}"
git checkout -b "$BRANCH_NAME"

# 提示用户修改
echo ""
echo -e "${YELLOW}请修改文件后继续...${NC}"
read -p "修改完成后按 Enter 继续..."

# 检查是否有修改
if [ -z "$(git status --porcelain)" ]; then
  echo -e "${RED}错误: 没有文件被修改${NC}"
  exit 1
fi

# 提交
echo -e "${GREEN}提交更改...${NC}"
git add .
git commit -m "$PR_TITLE"

# 推送到远程
echo -e "${GREEN}推送到远程分支...${NC}"
git push -u origin "$BRANCH_NAME"

# 使用 gh CLI 创建 PR
echo -e "${GREEN}创建 PR...${NC}"
if command -v gh &> /dev/null; then
  # 优先使用 GITHUB_TOKEN 环境变量
  if [ -n "$GITHUB_TOKEN" ]; then
    echo "使用 GITHUB_TOKEN 环境变量"
    GITHUB_TOKEN="$GITHUB_TOKEN" gh pr create \
      --base "$BASE_BRANCH" \
      --head "$BRANCH_NAME" \
      --title "$PR_TITLE" \
      --body "$PR_BODY"
  else
    gh pr create \
      --base "$BASE_BRANCH" \
      --head "$BRANCH_NAME" \
      --title "$PR_TITLE" \
      --body "$PR_BODY"
  fi
  
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}PR 创建成功！${NC}"
  else
    echo -e "${RED}PR 创建失败，请手动创建${NC}"
    echo "分支: $BRANCH_NAME"
    echo "Base: $BASE_BRANCH"
  fi
else
  echo -e "${YELLOW}gh CLI 未安装，请手动创建 PR${NC}"
  echo "分支: $BRANCH_NAME"
  echo "Base: $BASE_BRANCH"
  echo "PR 标题: $PR_TITLE"
fi

echo ""
echo -e "${GREEN}完成！${NC}"

