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
- 只能使用用户消息中“装备适配数据”“垂直英雄数据”“羁绊数据”“官方版本环境”“推荐阵容数据”出现的阵容、英雄、装备、强度、平均排名、前四率、吃鸡率和版本原因。
- 不要使用模型记忆、旧版本攻略、外部榜单或用户自己声称的版本信息来补充事实。
- 不要编造 T0/T0.5、版本号、赛区、高场次、平均排名、前四率、吃鸡率、运营节奏。
- 如果数据里没有某个阵容或数值，必须明确说“当前知识库没有命中”，不要硬给具体结论。
- 如果出现“已识别黑话”，内部查询以标准名为准；回答时可以说“我按A=B理解”，但不要把标准名强行改回黑话。
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
	if len(input.NormalizedTerms) > 0 {
		terms := make([]string, 0, len(input.NormalizedTerms))
		for _, term := range input.NormalizedTerms {
			terms = append(terms, fmt.Sprintf("%s => %s", term.Raw, term.Normalized))
		}
		sb.WriteString(fmt.Sprintf("- 已识别黑话：%s\n", strings.Join(terms, "、")))
	}
	if input.Feedback != nil {
		sb.WriteString(fmt.Sprintf("- 上轮反馈：%s\n", formatAdviceFeedback(input.Feedback.Type)))
		if input.Feedback.Reason != "" {
			sb.WriteString(fmt.Sprintf("- 反馈原因：%s\n", input.Feedback.Reason))
		}
		if input.Feedback.PreviousUserInput != "" {
			sb.WriteString(fmt.Sprintf("- 上轮问题：%s\n", input.Feedback.PreviousUserInput))
		}
		if input.Feedback.LastAdviceSummary != "" {
			sb.WriteString(fmt.Sprintf("- 上轮回答摘要：%s\n", input.Feedback.LastAdviceSummary))
		}
	}
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
	if ctx.UnitCost != nil {
		sb.WriteString(fmt.Sprintf("- 查询费用：%d费卡\n", *ctx.UnitCost))
	}
	if ctx.RoleQuery != "" {
		sb.WriteString(fmt.Sprintf("- 查询定位：%s\n", formatRoleQuery(ctx.RoleQuery)))
	}
	if hints := buildDecisionPolicyHints(ctx, input); len(hints) > 0 {
		sb.WriteString("\n## 局内决策提示\n")
		for _, hint := range hints {
			sb.WriteString(fmt.Sprintf("- %s\n", hint))
		}
	}

	// ── 2. 知识库元信息 ──────────────────────────────────
	if input.Metadata != nil {
		sb.WriteString("\n## 知识库元信息\n")
		if input.Metadata.Source != "" {
			sb.WriteString(fmt.Sprintf("- 来源：%s\n", input.Metadata.Source))
		}
		if input.Metadata.Version != "" {
			sb.WriteString(fmt.Sprintf("- 版本：%s\n", input.Metadata.Version))
		}
		if input.Metadata.UpdatedAt != "" {
			sb.WriteString(fmt.Sprintf("- 更新时间：%s\n", input.Metadata.UpdatedAt))
		}
		if input.Metadata.SampleCount > 0 {
			sb.WriteString(fmt.Sprintf("- 样本量：%d\n", input.Metadata.SampleCount))
		}
	}

	// ── 3. 装备匹配结果 ──────────────────────────────────
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

	// ── 4. 垂直英雄数据 ──────────────────────────────────
	if len(input.MatchedChampions) > 0 {
		sb.WriteString("\n## 垂直英雄数据\n")
		if isWorkQuery(ctx.RoleQuery) {
			sb.WriteString("打工问题说明：优先判断该英雄是否适合前中期临时过渡/凑羁绊；不要把装备携带数据直接说成必须主C或必须追3。\n")
		}
		for i, champion := range input.MatchedChampions {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("\n### %d. %s", i+1, champion.Name))
			if champion.Cost > 0 {
				sb.WriteString(fmt.Sprintf("（%d费", champion.Cost))
				if champion.Role != "" && !isWorkQuery(ctx.RoleQuery) {
					sb.WriteString("，" + champion.Role)
				}
				sb.WriteString("）\n")
			} else if champion.Role != "" && !isWorkQuery(ctx.RoleQuery) {
				sb.WriteString(fmt.Sprintf("（%s）\n", champion.Role))
			} else {
				sb.WriteString("\n")
			}
			if champion.BestAvgPlacement > 0 {
				sb.WriteString(fmt.Sprintf("- 最佳平均排名：%.2f\n", champion.BestAvgPlacement))
			}
			if isWorkQuery(ctx.RoleQuery) && champion.WorkScore > 0 {
				sb.WriteString(fmt.Sprintf("- 打工评分：%.0f/100\n", champion.WorkScore))
			}
			if isWorkQuery(ctx.RoleQuery) && champion.WorkReason != "" {
				sb.WriteString(fmt.Sprintf("- 打工判断：%s\n", champion.WorkReason))
			}
			if len(champion.Tags) > 0 && !isWorkQuery(ctx.RoleQuery) {
				sb.WriteString(fmt.Sprintf("- 定位判断：%s\n", strings.Join(champion.Tags, "、")))
			}
			if len(champion.BestComps) > 0 {
				best := champion.BestComps[0]
				sb.WriteString(fmt.Sprintf("- 最适合阵容：%s（%s，平均排名%.2f，前四率%.0f%%，吃鸡率%.0f%%）\n",
					best.Name,
					formatTier(best.Tier),
					best.AvgPlacement,
					best.Top4Rate*100,
					best.WinRate*100,
				))
			}
			if len(champion.BestBuilds) > 0 && len(champion.BestBuilds[0].Items) > 0 {
				if isWorkQuery(ctx.RoleQuery) {
					sb.WriteString(fmt.Sprintf("- 可携带装备数据：%s（仅表示知识库里出现过的携带方案，不等于打工必做装备）\n", strings.Join(champion.BestBuilds[0].Items, " + ")))
				} else {
					sb.WriteString(fmt.Sprintf("- 推荐装备：%s\n", strings.Join(champion.BestBuilds[0].Items, " + ")))
				}
			}
		}
	}

	// ── 5. 羁绊数据 ──────────────────────────────────────
	if len(input.MatchedTraits) > 0 {
		sb.WriteString("\n## 羁绊数据\n")
		for i, trait := range input.MatchedTraits {
			if i >= 3 {
				break
			}
			sb.WriteString(fmt.Sprintf("\n### %d. %s\n", i+1, trait.Name))
			if len(trait.Activations) > 0 {
				sb.WriteString(fmt.Sprintf("- 命中档位：%s\n", strings.Join(trait.Activations, "、")))
			}
			if len(trait.Units) > 0 {
				sb.WriteString(fmt.Sprintf("- 常见单位：%s\n", strings.Join(trait.Units, "、")))
			}
			for j, comp := range trait.BestComps {
				if j >= 3 {
					break
				}
				sb.WriteString(fmt.Sprintf("- 代表阵容：%s（%s，平均排名%.2f，前四率%.0f%%，吃鸡率%.0f%%）\n",
					comp.Name,
					formatTier(comp.Tier),
					comp.AvgPlacement,
					comp.Top4Rate*100,
					comp.WinRate*100,
				))
			}
		}
	}

	// ── 6. 官方版本环境 ──────────────────────────────────
	if len(input.PatchNotes) > 0 {
		sb.WriteString("\n## 官方版本环境\n")
		for i, note := range input.PatchNotes {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("\n### %d. %s - %s\n", i+1, note.Patch, note.SectionTitle))
			if note.Summary != "" {
				sb.WriteString(fmt.Sprintf("- 摘要：%s\n", note.Summary))
			}
			if len(note.ImpactTags) > 0 {
				sb.WriteString(fmt.Sprintf("- 影响标签：%s\n", strings.Join(note.ImpactTags, "、")))
			}
			if note.Source != "" || note.PublishedAt != "" {
				sb.WriteString(fmt.Sprintf("- 来源：%s %s\n", note.Source, note.PublishedAt))
			}
			if len(note.Details) > 0 {
				sb.WriteString(fmt.Sprintf("- 关键细节：%s\n", strings.Join(note.Details, "；")))
			}
		}
	}

	// ── 7. 推荐阵容数据 ──────────────────────────────────
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
			if comp.Metadata != nil {
				if comp.Metadata.UpdatedAt != "" {
					sb.WriteString(fmt.Sprintf("- 数据更新时间：%s\n", comp.Metadata.UpdatedAt))
				}
				if comp.Metadata.SampleCount > 0 && comp.Metadata.SampleCount != comp.Count {
					sb.WriteString(fmt.Sprintf("- 阵容样本：%d\n", comp.Metadata.SampleCount))
				}
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
			if comp.Plan != nil {
				writeCompPlan(&sb, comp.Plan)
			}
		}
	}

	// ── 8. 没有匹配到任何数据 ────────────────────────────
	if len(input.MatchedComps) == 0 && len(input.MatchedItems) == 0 && len(input.MatchedChampions) == 0 && len(input.MatchedTraits) == 0 {
		sb.WriteString("\n## 数据说明\n")
		sb.WriteString("当前条件未匹配到具体阵容数据。不要编造具体阵容、版本、数值或运营节奏，只能说明当前知识库没有命中，并提示玩家补充英雄、装备、羁绊或重新更新知识库。\n")
	}

	// ── 9. 输出指令 ──────────────────────────────────────
	sb.WriteString("\n## 你的任务\n")
	sb.WriteString("先给玩家明确结论，再给理由和操作；阵容强度结论必须对应上方统计数据，版本原因可以引用官方版本环境。")
	sb.WriteString(buildFeedbackInstruction(input.Feedback))
	sb.WriteString(buildInstruction(ctx))

	return sb.String(), nil
}

func buildFeedbackInstruction(feedback *AdviceFeedback) string {
	if feedback == nil {
		return ""
	}
	switch feedback.Type {
	case FeedbackRejected:
		return "用户已表示上一轮建议没命中。开头用一句话承认并重新判断，不要重复上一轮原话；如果知识库证据不足，要直接说明不足。"
	case FeedbackNeedsContext:
		return "用户补充了新局面。先说明这个上下文会改变判断，并基于新阶段/血量/装备重新给操作。"
	case FeedbackAcceptedOrContinued:
		return "用户沿着上一轮继续追问。不要重新科普整套阵容，直接推进下一步操作。"
	default:
		return ""
	}
}

func formatAdviceFeedback(feedbackType string) string {
	switch feedbackType {
	case FeedbackRejected:
		return "上一轮建议可能没命中"
	case FeedbackNeedsContext:
		return "用户补充了新局面"
	case FeedbackAcceptedOrContinued:
		return "用户沿着上一轮继续追问"
	default:
		return feedbackType
	}
}

func buildDecisionPolicyHints(ctx Context, input *NluEnrichedContext) []string {
	stageMajor, stageRound := parseStageNumber(ctx.GameStage)
	hasGameState := ctx.GameStage != nil || ctx.Level != nil || ctx.HP != nil || ctx.Gold != nil
	if !hasGameState {
		return nil
	}

	hints := make([]string, 0, 6)
	switch {
	case stageMajor > 0 && stageMajor <= 2:
		hints = append(hints, "当前偏前期，回答重点放在稳血过渡、装备先合和不要过早锁死最终阵容。")
	case stageMajor == 3:
		hints = append(hints, "当前进入三阶段，回答需要明确是否补质量：血量低优先稳血，经济好才考虑贪人口。")
	case stageMajor >= 4:
		hints = append(hints, "当前已到四阶段以后，回答要给出明确成型路线：该D就D，该上人口就上人口，不要只说阵容强度。")
	}

	if stageMajor == 3 && stageRound == 2 {
		hints = append(hints, "3-2 是常见启动点，如果质量弱或血量低，要优先说明是否拉6/小D稳血。")
	}
	if stageMajor == 4 && stageRound == 1 {
		hints = append(hints, "4-1 是常见大节奏点，如果目标阵容依赖4费/高费，应说明拉7/拉8和搜牌取舍。")
	}

	if ctx.HP != nil {
		switch {
		case *ctx.HP <= 35:
			hints = append(hints, "血量很低，策略偏保命：优先即时战力和止血，不建议空等完美成型。")
		case *ctx.HP <= 55:
			hints = append(hints, "血量中低，策略偏稳血：可以牺牲一点经济换质量。")
		case *ctx.HP >= 75:
			hints = append(hints, "血量健康，可以更贪经济或人口，但仍要结合装备和来牌判断。")
		}
	}

	if ctx.Gold != nil {
		switch {
		case *ctx.Gold < 20:
			hints = append(hints, "经济偏低，除非血量危险，否则不要建议大搜。")
		case *ctx.Gold >= 50:
			hints = append(hints, "经济健康，可以讨论卡利息拉人口或在关键等级搜牌。")
		}
	}

	if ctx.Level != nil {
		if stageMajor >= 4 && *ctx.Level <= 6 {
			hints = append(hints, "当前阶段等级偏低，回答要提醒节奏落后，优先补人口或补质量。")
		}
		if stageMajor <= 3 && *ctx.Level >= 7 {
			hints = append(hints, "当前等级偏领先，回答可以考虑连胜压制或提前抢体系牌。")
		}
	}

	if input != nil && len(input.MatchedComps) > 0 {
		best := input.MatchedComps[0]
		if best.AvgPlacement > 0 {
			hints = append(hints, fmt.Sprintf("当前推荐第一套是 %s（平均排名%.2f），回答要把它和备选阵容按可执行性排序。", best.Name, best.AvgPlacement))
		}
		if best.Plan != nil {
			if best.Plan.Early != nil || best.Plan.Middle != nil {
				hints = append(hints, "知识库有前中期棋盘，回答可以引用 early/middle/final（前期/中期/成型）路径。")
			} else if len(best.Plan.Final.Units) > 0 {
				hints = append(hints, "知识库只有成型棋盘，回答不能编造前中期过渡。")
			}
		}
	}

	return hints
}

func parseStageNumber(stage *string) (int, int) {
	if stage == nil {
		return 0, 0
	}
	trimmed := strings.TrimSpace(*stage)
	if trimmed == "" {
		return 0, 0
	}
	var major, round int
	if _, err := fmt.Sscanf(trimmed, "%d-%d", &major, &round); err == nil {
		return major, round
	}
	if _, err := fmt.Sscanf(trimmed, "%d阶段", &major); err == nil {
		return major, 0
	}
	return 0, 0
}

func writeCompPlan(sb *strings.Builder, plan *CompPlan) {
	if plan == nil {
		return
	}
	if plan.Early != nil {
		sb.WriteString(fmt.Sprintf("- 前期棋盘：%s\n", formatBoardSnapshot(*plan.Early)))
	}
	if plan.Middle != nil {
		sb.WriteString(fmt.Sprintf("- 中期棋盘：%s\n", formatBoardSnapshot(*plan.Middle)))
	}
	if len(plan.Final.Units) > 0 || len(plan.Final.Traits) > 0 {
		sb.WriteString(fmt.Sprintf("- 成型棋盘：%s\n", formatBoardSnapshot(plan.Final)))
	}
}

func formatBoardSnapshot(snapshot BoardSnapshot) string {
	parts := make([]string, 0, 3)
	if snapshot.Level != "" {
		parts = append(parts, snapshot.Level+"级")
	}
	if len(snapshot.Units) > 0 {
		units := make([]string, 0, len(snapshot.Units))
		for _, unit := range snapshot.Units {
			name := strings.TrimSpace(unit.Name)
			if name == "" {
				continue
			}
			if unit.IsCore && len(unit.Items) > 0 {
				name += "（" + strings.Join(unit.Items, "+") + "）"
			}
			units = append(units, name)
		}
		if len(units) > 0 {
			parts = append(parts, strings.Join(units, "、"))
		}
	}
	if len(snapshot.Traits) > 0 {
		traits := make([]string, 0, len(snapshot.Traits))
		for _, trait := range snapshot.Traits {
			if trait.Name == "" {
				continue
			}
			if trait.Count > 0 {
				traits = append(traits, fmt.Sprintf("%d%s", trait.Count, trait.Name))
			} else {
				traits = append(traits, trait.Name)
			}
		}
		if len(traits) > 0 {
			parts = append(parts, "羁绊："+strings.Join(traits, "、"))
		}
	}
	if len(parts) == 0 {
		return "当前知识库无棋盘细节"
	}
	return strings.Join(parts, "；")
}

// buildInstruction 根据意图定制输出指令
func buildInstruction(ctx Context) string {
	switch ctx.Intent {
	case "item_query":
		return "根据以上装备适配数据推荐阵容；如果装备携带者和阵容名主核心不同，只能说“这件装备可给某英雄/副C/功能位”，不要说成“核心英雄”。"
	case "lineup_recommend":
		return "根据以上阵容数据，推荐1~3个最适合当前局面的阵容，说明推荐理由、运营节奏和装备方向。"
	case "trait_query":
		return "根据以上羁绊数据，先回答这个羁绊能不能玩，再解释强阵容、常见单位、激活档位和装备方向；如果没有解锁机制数据，不要编造解锁规则。"
	case "vertical_query":
		return "根据以上垂直英雄数据，按玩家问法分清谁能C、谁能抗、谁综合最强；优先给前3个选择，不能把未命中的英雄塞进答案。"
	case "champion_query":
		if isWorkQuery(ctx.RoleQuery) {
			return "玩家问的是打工/过渡强度，不是后期主C。先回答“能不能拿来打工”，再说明适合凑什么羁绊/过渡到什么强阵容；不要默认说能当主C，不要强推追3；装备只按“可携带”表达。"
		}
		return "根据以上数据，说明该英雄在哪个阵容中最强、推荐出装、是否值得追3星。"
	case "playstyle_query":
		return "根据玩家的玩法偏好，从以上阵容中推荐最适合的方向，说明操作要点。"
	case "augment_query":
		return "根据以上数据，说明该海克斯适合哪些阵容，优先级如何，给出选择建议。"
	default:
		return "根据以上所有数据，给出最适合当前局面的建议。"
	}
}

func formatRoleQuery(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "carry":
		return "主C/输出"
	case "tank":
		return "前排/能抗"
	case "work":
		return "打工/过渡"
	case "all":
		return "综合比较"
	default:
		return role
	}
}

func isWorkQuery(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "work", "worker", "打工", "过渡", "前期", "二阶段":
		return true
	default:
		return false
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
