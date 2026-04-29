package agent

import (
	"fmt"
	"github.com/sagerlabs/awesome/tft/data"
	"strings"
)

// ── Prompt 构建 ───────────────────────────────────────────────────────────────

const systemPrompt = ` 你是一位云顶之弈高段位教练，擅长根据玩家当前棋盘状态给出精准的阵容推荐和运营建议。
请用简洁清晰的中文回答，重点突出：推荐阵容、原因、下一步购买优先级。`

// BuildPrompt 把交集计算结果转成 LLM 的用户 prompt
func BuildPrompt(in *data.IntersectionOutput) string {
	if len(in.Recommendations) == 0 {
		return "当前英雄和装备组合没有找到强力阵容方向，请分析可能的转型方向。"
	}

	top := in.Recommendations[0]
	prompt := fmt.Sprintf(`
当前棋盘分析结果：

【推荐主方向】
阵容：%s
强度：%s Tier（平均名次 %.2f，进前4率 %.0f%%）
命中来源：%v
置信度：%.0f%%

【已有核心英雄】%v
【缺少核心英雄】%v
【命中装备】%v
【建议 Carry】%s
【推荐升级节点】%s

`,
		top.Comp.Name,
		top.Comp.Tier,
		top.Comp.AvgPlacement,
		top.Top4Rate*100,
		top.HitSources,
		top.Confidence*100,
		top.MatchedUnits,
		top.MissingUnits,
		top.MatchedItems,
		top.SuggestedCarry,
		top.Comp.Levelling,
	)

	// 备选方向
	if len(in.Recommendations) > 1 {
		prompt += "【备选方向】\n"
		for _, r := range in.Recommendations[1:3] {
			prompt += fmt.Sprintf("- %s (%s Tier, 置信度 %.0f%%)\n",
				r.Comp.Name, r.Comp.Tier, r.Confidence*100)
		}
	}

	prompt += "\n请根据以上分析，给出详细的运营建议和购买优先级。"
	return prompt
}

const FormatSystemPrompt = `你是云顶之弈顶级教练，根据玩家当前局面数据给出简洁精准的建议。
硬性事实边界：
- 只能使用用户消息中“装备适配数据”和“推荐阵容数据”出现的阵容、英雄、装备、强度、平均排名、前四率、吃鸡率。
- 不要使用模型记忆、旧版本攻略、外部榜单或用户自己声称的版本信息来补充事实。
- 不要编造 T0/T0.5、版本号、赛区、高场次、平均排名、前四率、吃鸡率、运营节奏。
- 如果数据里没有某个阵容或数值，必须明确说“当前知识库没有命中”，不要硬给具体结论。
输出要求：
- 使用 Markdown 格式
- 语气简洁直接，像教练喊话
- 第一句话先回答玩家最关心的结论，例如“能冲”“别硬冲”“优先玩X”
- 输出风格贴近国服云顶老玩家：把 S Tier/A Tier 说成“S级/A级”“版本强势/可玩”，可以使用“保底分”“上限吃鸡”“成型有鸡”“锁血”“稳血”“别硬冲”等自然说法
- 使用老玩家话术时不能夸大数据：只有吃鸡率和强度都明显靠前时才说“上限吃鸡/成型有鸡”，不要把所有阵容都说成“必鸡”
- 不要废话，直接给结论和操作建议
- 数据来源说明放在回答末尾一句轻提示，不要放在开头`

// BuildNluFormatPrompt 把 NluEnrichedContext 打包成排版 Prompt
func BuildNluFormatPrompt(input *NluEnrichedContext) (string, error) {
	if input == nil {
		return "", fmt.Errorf("input is nil")
	}

	var sb strings.Builder
	itemMatchIndex := buildItemMatchIndex(input.MatchedItems)

	// ── 1. 玩家当前局面 ──────────────────────────────────
	sb.WriteString("## 玩家当前局面\n")
	sb.WriteString(fmt.Sprintf("- 原始问题：%s\n", input.UserInput))

	ctx := input.Ctx
	if ctx.Gold != nil {
		sb.WriteString(fmt.Sprintf("- 金币：%d\n", *ctx.Gold))
	}
	if ctx.Level != nil {
		sb.WriteString(fmt.Sprintf("- 等级：%d\n", *ctx.Level))
	}
	if ctx.HP != nil {
		sb.WriteString(fmt.Sprintf("- 血量：%d\n", *ctx.HP))
	}
	if ctx.GameStage != nil {
		sb.WriteString(fmt.Sprintf("- 阶段：%s\n", *ctx.GameStage))
	}
	if len(ctx.Champions) > 0 {
		champs := make([]string, 0, len(ctx.Champions))
		for name, star := range ctx.Champions {
			champs = append(champs, fmt.Sprintf("%s(%d星)", name, star))
		}
		sb.WriteString(fmt.Sprintf("- 英雄：%s\n", strings.Join(champs, "、")))
	}
	if len(ctx.Items) > 0 {
		sb.WriteString(fmt.Sprintf("- 装备：%s\n", strings.Join(ctx.Items, "、")))
	}
	if len(ctx.Augments) > 0 {
		sb.WriteString(fmt.Sprintf("- 海克斯：%s\n", strings.Join(ctx.Augments, "、")))
	}
	if ctx.ExplicitLineup != nil && *ctx.ExplicitLineup != "" {
		sb.WriteString(fmt.Sprintf("- 目标阵容：%s\n", *ctx.ExplicitLineup))
	}
	if ctx.Playstyle != "" {
		sb.WriteString(fmt.Sprintf("- 玩法偏好：%s\n", ctx.Playstyle))
	}

	// ── 2. 装备匹配结果 ──────────────────────────────────
	if len(input.MatchedItems) > 0 {
		sb.WriteString("\n## 装备适配数据\n")
		for _, item := range input.MatchedItems {
			sb.WriteString(fmt.Sprintf("\n**%s** 命中的装备携带点（装备携带者不一定等于阵容名里的主核心）：\n", item.ItemName))
			for i, comp := range item.CompInfos {
				if i >= 3 {
					break // 最多展示3个
				}
				sb.WriteString(fmt.Sprintf(
					"- %s（%s，平均排名%.2f）里可给 **%s**，优先级%d/100\n",
					comp.CompName,
					formatTier(comp.CompTier),
					comp.CompAvg,
					comp.CarryName,
					comp.PriorityScore,
				))
			}
		}
	}

	// ── 3. 推荐阵容数据 ──────────────────────────────────
	if len(input.MatchedComps) > 0 {
		sb.WriteString("\n## 推荐阵容数据\n")
		for i, comp := range input.MatchedComps {
			if i >= 3 {
				break // 最多展示3个
			}
			sb.WriteString(fmt.Sprintf("\n### %d. %s（%s）\n", i+1, comp.Name, formatTier(comp.Tier)))
			sb.WriteString(fmt.Sprintf("- 平均排名：%.2f｜前四率：%.0f%%｜吃鸡率：%.0f%%\n",
				comp.AvgPlacement,
				comp.Top4Rate*100,
				comp.WinRate*100,
			))
			if comp.Count > 0 {
				sb.WriteString(fmt.Sprintf("- 样本场次：%d\n", comp.Count))
			}
			if comp.Levelling != "" {
				sb.WriteString(fmt.Sprintf("- 运营节奏：%s\n", formatLevelling(comp.Levelling)))
			}
			if matches := itemMatchIndex[comp.ClusterID]; len(matches) > 0 {
				sb.WriteString(fmt.Sprintf("- 本次装备适配：%s\n", strings.Join(matches, "；")))
			}
			if len(comp.Stars) > 0 {
				sb.WriteString(fmt.Sprintf("- 追3星：%s\n", strings.Join(comp.Stars, "、")))
			}
			if comp.BestBuild.Carry != "" {
				sb.WriteString(fmt.Sprintf("- 核心装备：%s 带 %s\n",
					comp.BestBuild.Carry,
					strings.Join(comp.BestBuild.Items, " + "),
				))
			}
		}
	}

	// ── 4. 没有匹配到任何数据 ────────────────────────────
	if len(input.MatchedComps) == 0 && len(input.MatchedItems) == 0 {
		sb.WriteString("\n## 数据说明\n")
		sb.WriteString("当前条件未匹配到具体阵容数据。不要编造具体阵容、版本、数值或运营节奏，只能说明当前知识库没有命中，并提示玩家补充英雄、装备、羁绊或重新更新知识库。\n")
	}

	// ── 5. 输出指令 ──────────────────────────────────────
	sb.WriteString("\n## 你的任务\n")
	sb.WriteString("先给玩家明确结论，再给理由和操作；所有阵容、装备、数值必须逐项对应上方数据。")
	sb.WriteString(buildInstruction(ctx.Intent))

	return sb.String(), nil
}

// buildInstruction 根据意图定制输出指令
func buildInstruction(intent string) string {
	switch intent {
	case "item_query":
		return "根据以上装备适配数据推荐阵容；如果装备携带者和阵容名主核心不同，只能说“这件装备可给某英雄/副C/功能位”，不要说成“核心英雄”。"
	case "lineup_recommend":
		return "根据以上阵容数据，推荐1~3个最适合当前局面的阵容，说明推荐理由、运营节奏和装备方向。"
	case "trait_query":
		return "根据以上数据，解释该羁绊的强度和最佳激活方式，推荐相关阵容。"
	case "champion_query":
		return "根据以上数据，说明该英雄在哪个阵容中最强、推荐出装、是否值得追3星。"
	case "playstyle_query":
		return "根据玩家的玩法偏好，从以上阵容中推荐最适合的方向，说明操作要点。"
	case "augment_query":
		return "根据以上数据，说明该海克斯适合哪些阵容，优先级如何，给出选择建议。"
	default:
		return "根据以上所有数据，给出最适合当前局面的建议。"
	}
}

func formatTier(tier string) string {
	trimmed := strings.TrimSpace(tier)
	if trimmed == "" {
		return "未知强度"
	}

	upper := strings.ToUpper(trimmed)
	switch upper {
	case "S":
		return "S级"
	case "A":
		return "A级"
	case "B":
		return "B级"
	case "C":
		return "C级"
	case "D":
		return "D级"
	}

	if strings.HasSuffix(upper, " TIER") {
		return strings.TrimSpace(strings.TrimSuffix(upper, " TIER")) + "级"
	}
	return trimmed
}

func buildItemMatchIndex(items []MatchedItemInfo) map[string][]string {
	index := make(map[string][]string)
	for _, item := range items {
		for _, comp := range item.CompInfos {
			if item.ItemName == "" || comp.CarryName == "" || comp.ClusterID == "" {
				continue
			}
			text := fmt.Sprintf("%s可给%s（优先级%d/100）", item.ItemName, comp.CarryName, comp.PriorityScore)
			index[comp.ClusterID] = append(index[comp.ClusterID], text)
		}
	}
	return index
}

func formatLevelling(levelling string) string {
	trimmed := strings.TrimSpace(levelling)
	switch strings.ToLower(trimmed) {
	case "fast 9":
		return "快速上9"
	case "fast 8":
		return "快速上8"
	case "slow roll":
		return "慢D追三"
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "lvl ") {
		level := strings.TrimSpace(trimmed[4:])
		if level != "" {
			return level + "级节奏"
		}
	}
	return trimmed
}
