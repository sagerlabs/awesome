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
输出要求：
- 使用 Markdown 格式
- 语气简洁直接，像教练喊话
- 不要废话，直接给结论和操作建议`

// BuildNluFormatPrompt 把 NluEnrichedContext 打包成排版 Prompt
func BuildNluFormatPrompt(input *NluEnrichedContext) (string, error) {
	if input == nil {
		return "", fmt.Errorf("input is nil")
	}

	var sb strings.Builder

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
			sb.WriteString(fmt.Sprintf("\n**%s** 适合以下阵容：\n", item.ItemName))
			for i, comp := range item.CompInfos {
				if i >= 3 {
					break // 最多展示3个
				}
				sb.WriteString(fmt.Sprintf(
					"- %s（%s Tier，平均排名%.2f）→ 给 **%s**，优先级%d/100\n",
					comp.CompName,
					comp.CompTier,
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
			sb.WriteString(fmt.Sprintf("\n### %d. %s（%s Tier）\n", i+1, comp.Name, comp.Tier))
			sb.WriteString(fmt.Sprintf("- 平均排名：%.2f｜前四率：%.0f%%｜吃鸡率：%.0f%%\n",
				comp.AvgPlacement,
				comp.Top4Rate*100,
				comp.WinRate*100,
			))
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
		sb.WriteString("当前条件未匹配到具体阵容数据，请根据玩家描述给出通用建议。\n")
	}

	// ── 5. 输出指令 ──────────────────────────────────────
	sb.WriteString("\n## 你的任务\n")
	sb.WriteString(buildInstruction(ctx.Intent))

	return sb.String(), nil
}

// buildInstruction 根据意图定制输出指令
func buildInstruction(intent string) string {
	switch intent {
	case "item_query":
		return "根据以上装备适配数据，告诉玩家这个装备优先给哪个英雄、对应什么阵容，给出明确结论。"
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
