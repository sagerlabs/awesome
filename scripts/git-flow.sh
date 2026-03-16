#!/bin/bash
# GitFlow 命令包装脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 主分支
MAIN_BRANCH="main"
DEV_BRANCH="dev"

# 帮助信息
show_help() {
    echo -e "${GREEN}GitFlow 助手${NC}"
    echo ""
    echo "用法: ./scripts/git-flow.sh <command> [args]"
    echo ""
    echo "命令列表:"
    echo "  feature start <name>     - 创建功能分支"
    echo "  feature push-pr <name>   - 推送分支并创建 PR"
    echo "  feature finish <name>    - 完成功能分支（本地合并）"
    echo "  release start <version>  - 创建发布分支"
    echo "  release finish <version> - 完成发布分支"
    echo "  hotfix start <version>   - 创建热修复分支"
    echo "  hotfix finish <version>  - 完成热修复分支"
    echo "  help                     - 显示帮助信息"
    echo ""
    echo "示例:"
    echo "  ./scripts/git-flow.sh feature start user-auth"
    echo "  ./scripts/git-flow.sh feature push-pr user-auth"
    echo "  ./scripts/git-flow.sh release start 1.0.0"
    echo "  ./scripts/git-flow.sh hotfix start 1.0.1"
}

# 检查当前分支是否干净
check_clean() {
    if [ -n "$(git status --porcelain)" ]; then
        echo -e "${RED}错误: 工作区有未提交的更改，请先提交或 stash${NC}"
        exit 1
    fi
}

# 功能分支相关
feature_start() {
    local name="$1"
    if [ -z "$name" ]; then
        echo -e "${RED}错误: 请指定功能分支名称${NC}"
        exit 1
    fi
    
    check_clean
    
    echo -e "${GREEN}正在创建功能分支: feature/${name}${NC}"
    
    git checkout "$DEV_BRANCH"
    git pull origin "$DEV_BRANCH"
    git checkout -b "feature/${name}"
    
    echo -e "${GREEN}功能分支 feature/${name} 创建成功！${NC}"
    echo -e "${YELLOW}现在可以开始开发了${NC}"
}

feature_push_pr() {
    local name="$1"
    if [ -z "$name" ]; then
        echo -e "${RED}错误: 请指定功能分支名称${NC}"
        exit 1
    fi
    
    local branch="feature/${name}"
    
    # 检查当前分支是否是目标分支
    CURRENT_BRANCH=$(git branch --show-current)
    if [ "$CURRENT_BRANCH" != "$branch" ]; then
        echo -e "${YELLOW}警告: 当前分支不是 $branch，切换到 $branch${NC}"
        git checkout "$branch"
    fi
    
    # 检查是否有未提交的更改
    if [ -n "$(git status --porcelain)" ]; then
        echo -e "${YELLOW}工作区有未提交的更改，是否提交? (y/n)${NC}"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            echo -e "${YELLOW}请输入提交信息:${NC}"
            read -r commit_msg
            git add .
            git commit -m "$commit_msg"
        else
            echo -e "${RED}错误: 请先提交或 stash 更改${NC}"
            exit 1
        fi
    fi
    
    # 获取最后一个提交信息作为 PR 标题
    PR_TITLE=$(git log -1 --pretty=%s)
    PR_BODY="自动化创建 PR 来自分支: $branch"
    
    echo -e "${GREEN}推送分支到远程...${NC}"
    git push -u origin "$branch"
    
    echo -e "${GREEN}创建 PR 到 $DEV_BRANCH...${NC}"
    if command -v gh &> /dev/null; then
        # 优先使用 GITHUB_TOKEN 环境变量
        if [ -n "$GITHUB_TOKEN" ]; then
            GITHUB_TOKEN="$GITHUB_TOKEN" gh pr create \
                --base "$DEV_BRANCH" \
                --head "$branch" \
                --title "$PR_TITLE" \
                --body "$PR_BODY"
        else
            gh pr create \
                --base "$DEV_BRANCH" \
                --head "$branch" \
                --title "$PR_TITLE" \
                --body "$PR_BODY"
        fi
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}PR 创建成功！${NC}"
        else
            echo -e "${RED}PR 创建失败，但分支已推送${NC}"
            echo -e "${YELLOW}请手动创建 PR: ${branch} -> ${DEV_BRANCH}${NC}"
        fi
    else
        echo -e "${YELLOW}gh CLI 未安装，请手动创建 PR${NC}"
        echo -e "${YELLOW}分支: ${branch}${NC}"
        echo -e "${YELLOW}Base: ${DEV_BRANCH}${NC}"
    fi
    
    echo -e "${GREEN}完成！${NC}"
}

feature_finish() {
    local name="$1"
    if [ -z "$name" ]; then
        echo -e "${RED}错误: 请指定功能分支名称${NC}"
        exit 1
    fi
    
    local branch="feature/${name}"
    
    check_clean
    
    echo -e "${GREEN}正在完成功能分支: ${branch}${NC}"
    echo -e "${YELLOW}注意: 这会直接合并到本地 ${DEV_BRANCH}${NC}"
    echo -e "${YELLOW}如果要创建 PR，请使用: feature push-pr ${name}${NC}"
    echo -e "${YELLOW}继续? (y/n)${NC}"
    read -r answer
    if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
        echo -e "${YELLOW}已取消${NC}"
        exit 0
    fi
    
    git checkout "$DEV_BRANCH"
    git pull origin "$DEV_BRANCH"
    git merge --no-ff "$branch"
    
    echo -e "${GREEN}功能分支 ${branch} 已合并到 ${DEV_BRANCH}${NC}"
    echo -e "${YELLOW}是否删除分支 ${branch}? (y/n)${NC}"
    read -r answer
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        git branch -d "$branch"
        echo -e "${GREEN}分支 ${branch} 已删除${NC}"
    fi
    
    echo -e "${GREEN}功能分支完成！${NC}"
    echo -e "${YELLOW}是否推送到远程 ${DEV_BRANCH}? (y/n)${NC}"
    read -r answer
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        git push origin "$DEV_BRANCH"
        echo -e "${GREEN}已推送到远程 ${DEV_BRANCH}${NC}"
    else
        echo -e "${YELLOW}请记得运行 'git push origin $DEV_BRANCH' 推送到远程${NC}"
    fi
}

# 发布分支相关
release_start() {
    local version="$1"
    if [ -z "$version" ]; then
        echo -e "${RED}错误: 请指定版本号${NC}"
        exit 1
    fi
    
    check_clean
    
    echo -e "${GREEN}正在创建发布分支: release/${version}${NC}"
    
    git checkout "$DEV_BRANCH"
    git pull origin "$DEV_BRANCH"
    git checkout -b "release/${version}"
    
    echo -e "${GREEN}发布分支 release/${version} 创建成功！${NC}"
    echo -e "${YELLOW}请更新版本号、CHANGELOG 等，然后运行 'git flow release finish ${version}'${NC}"
}

release_finish() {
    local version="$1"
    if [ -z "$version" ]; then
        echo -e "${RED}错误: 请指定版本号${NC}"
        exit 1
    fi
    
    local branch="release/${version}"
    
    check_clean
    
    echo -e "${GREEN}正在完成发布分支: ${branch}${NC}"
    
    # 合并到 main
    git checkout "$MAIN_BRANCH"
    git pull origin "$MAIN_BRANCH"
    git merge --no-ff "$branch"
    git tag -a "v${version}" -m "Release version ${version}"
    
    # 合并到 dev
    git checkout "$DEV_BRANCH"
    git merge --no-ff "$branch"
    
    echo -e "${GREEN}发布分支 ${branch} 已合并到 ${MAIN_BRANCH} 和 ${DEV_BRANCH}${NC}"
    echo -e "${GREEN}已创建标签: v${version}${NC}"
    
    echo -e "${YELLOW}是否删除分支 ${branch}? (y/n)${NC}"
    read -r answer
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        git branch -d "$branch"
        echo -e "${GREEN}分支 ${branch} 已删除${NC}"
    fi
    
    echo -e "${GREEN}发布完成！${NC}"
    echo -e "${YELLOW}请运行以下命令推送到远程:${NC}"
    echo -e "  git push origin $MAIN_BRANCH"
    echo -e "  git push origin $DEV_BRANCH"
    echo -e "  git push origin v${version}"
}

# 热修复分支相关
hotfix_start() {
    local version="$1"
    if [ -z "$version" ]; then
        echo -e "${RED}错误: 请指定版本号${NC}"
        exit 1
    fi
    
    check_clean
    
    echo -e "${GREEN}正在创建热修复分支: hotfix/${version}${NC}"
    
    git checkout "$MAIN_BRANCH"
    git pull origin "$MAIN_BRANCH"
    git checkout -b "hotfix/${version}"
    
    echo -e "${GREEN}热修复分支 hotfix/${version} 创建成功！${NC}"
    echo -e "${YELLOW}请修复问题，然后运行 'git flow hotfix finish ${version}'${NC}"
}

hotfix_finish() {
    local version="$1"
    if [ -z "$version" ]; then
        echo -e "${RED}错误: 请指定版本号${NC}"
        exit 1
    fi
    
    local branch="hotfix/${version}"
    
    check_clean
    
    echo -e "${GREEN}正在完成热修复分支: ${branch}${NC}"
    
    # 合并到 main
    git checkout "$MAIN_BRANCH"
    git pull origin "$MAIN_BRANCH"
    git merge --no-ff "$branch"
    git tag -a "v${version}" -m "Hotfix version ${version}"
    
    # 合并到 dev
    git checkout "$DEV_BRANCH"
    git merge --no-ff "$branch"
    
    echo -e "${GREEN}热修复分支 ${branch} 已合并到 ${MAIN_BRANCH} 和 ${DEV_BRANCH}${NC}"
    echo -e "${GREEN}已创建标签: v${version}${NC}"
    
    echo -e "${YELLOW}是否删除分支 ${branch}? (y/n)${NC}"
    read -r answer
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        git branch -d "$branch"
        echo -e "${GREEN}分支 ${branch} 已删除${NC}"
    fi
    
    echo -e "${GREEN}热修复完成！${NC}"
    echo -e "${YELLOW}请运行以下命令推送到远程:${NC}"
    echo -e "  git push origin $MAIN_BRANCH"
    echo -e "  git push origin $DEV_BRANCH"
    echo -e "  git push origin v${version}"
}

# 主逻辑
case "${1:-}" in
    feature)
        case "${2:-}" in
            start)
                feature_start "$3"
                ;;
            push-pr)
                feature_push_pr "$3"
                ;;
            finish)
                feature_finish "$3"
                ;;
            *)
                echo -e "${RED}错误: 无效的 feature 子命令${NC}"
                show_help
                exit 1
                ;;
        esac
        ;;
    release)
        case "${2:-}" in
            start)
                release_start "$3"
                ;;
            finish)
                release_finish "$3"
                ;;
            *)
                echo -e "${RED}错误: 无效的 release 子命令${NC}"
                show_help
                exit 1
                ;;
        esac
        ;;
    hotfix)
        case "${2:-}" in
            start)
                hotfix_start "$3"
                ;;
            finish)
                hotfix_finish "$3"
                ;;
            *)
                echo -e "${RED}错误: 无效的 hotfix 子命令${NC}"
                show_help
                exit 1
                ;;
        esac
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}错误: 无效的命令${NC}"
        show_help
        exit 1
        ;;
esac
