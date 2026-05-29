package agent

import (
	"fmt"
	"strings"
)

// BuildNluFormatPrompt assembles the full structured user prompt from a NluEnrichedContext.
func BuildNluFormatPrompt(input *NluEnrichedContext) (string, error) {
	if input == nil {
		return "", fmt.Errorf("input is nil")
	}

	var sb strings.Builder
	itemMatchIndex := buildItemMatchIndex(input.MatchedItems)
	ctx := input.Ctx

	writeCurrentSituation(&sb, ctx, input)
	writeKnowledgeMetadata(&sb, input)
	writeItemSection(&sb, input.MatchedItems)
	writeChampionSection(&sb, input.MatchedChampions, ctx.RoleQuery)
	writeTraitSection(&sb, input.MatchedTraits)
	writePatchNotesSection(&sb, input.PatchNotes)
	writeCompSection(&sb, input.MatchedComps, itemMatchIndex)
	writeNoDataFallback(&sb, input)
	writeTaskInstruction(&sb, ctx, input)

	return sb.String(), nil
}

func writeCurrentSituation(sb *strings.Builder, ctx Context, input *NluEnrichedContext) {
	sb.WriteString("## 玩家当前局面\n")
	sb.WriteString(fmt.Sprintf("- 原始问题：%s\n", input.UserInput))

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
}

func writeKnowledgeMetadata(sb *strings.Builder, input *NluEnrichedContext) {
	if input.Metadata == nil {
		return
	}
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

func writeItemSection(sb *strings.Builder, items []MatchedItemInfo) {
	if len(items) == 0 {
		return
	}
	sb.WriteString("\n## 装备适配数据\n")
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("\n**%s** 命中的装备携带点（装备携带者不一定等于阵容名里的主核心）：\n", item.ItemName))
		for i, comp := range item.CompInfos {
			if i >= 3 {
				break
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

func writeChampionSection(sb *strings.Builder, champions []ChampionInsight, roleQuery string) {
	if len(champions) == 0 {
		return
	}
	sb.WriteString("\n## 垂直英雄数据\n")
	if isWorkQuery(roleQuery) {
		sb.WriteString("打工问题说明：优先判断该英雄是否适合前中期临时过渡/凑羁绊；不要把装备携带数据直接说成必须主C或必须追3。\n")
	}
	for i, champion := range champions {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("\n### %d. %s", i+1, champion.Name))
		if champion.Cost > 0 {
			sb.WriteString(fmt.Sprintf("（%d费", champion.Cost))
			if champion.Role != "" && !isWorkQuery(roleQuery) {
				sb.WriteString("，" + champion.Role)
			}
			sb.WriteString("）\n")
		} else if champion.Role != "" && !isWorkQuery(roleQuery) {
			sb.WriteString(fmt.Sprintf("（%s）\n", champion.Role))
		} else {
			sb.WriteString("\n")
		}
		if champion.BestAvgPlacement > 0 {
			sb.WriteString(fmt.Sprintf("- 最佳平均排名：%.2f\n", champion.BestAvgPlacement))
		}
		if isWorkQuery(roleQuery) && champion.WorkScore > 0 {
			sb.WriteString(fmt.Sprintf("- 打工评分：%.0f/100\n", champion.WorkScore))
		}
		if isWorkQuery(roleQuery) && champion.WorkReason != "" {
			sb.WriteString(fmt.Sprintf("- 打工判断：%s\n", champion.WorkReason))
		}
		if len(champion.Tags) > 0 && !isWorkQuery(roleQuery) {
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
			if isWorkQuery(roleQuery) {
				sb.WriteString(fmt.Sprintf("- 可携带装备数据：%s（仅表示知识库里出现过的携带方案，不等于打工必做装备）\n", strings.Join(champion.BestBuilds[0].Items, " + ")))
			} else {
				sb.WriteString(fmt.Sprintf("- 推荐装备：%s\n", strings.Join(champion.BestBuilds[0].Items, " + ")))
			}
		}
	}
}

func writeTraitSection(sb *strings.Builder, traits []TraitInsight) {
	if len(traits) == 0 {
		return
	}
	sb.WriteString("\n## 羁绊数据\n")
	for i, trait := range traits {
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

func writePatchNotesSection(sb *strings.Builder, notes []PatchNoteInsight) {
	if len(notes) == 0 {
		return
	}
	sb.WriteString("\n## 官方版本环境\n")
	for i, note := range notes {
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

func writeCompSection(sb *strings.Builder, comps []CompSummary, itemMatchIndex map[string][]string) {
	if len(comps) == 0 {
		return
	}
	sb.WriteString("\n## 推荐阵容数据\n")
	for i, comp := range comps {
		if i >= 3 {
			break
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
			writeCompPlan(sb, comp.Plan)
		}
	}
}

func writeNoDataFallback(sb *strings.Builder, input *NluEnrichedContext) {
	if len(input.MatchedComps) > 0 || len(input.MatchedItems) > 0 ||
		len(input.MatchedChampions) > 0 || len(input.MatchedTraits) > 0 {
		return
	}
	sb.WriteString("\n## 数据说明\n")
	sb.WriteString("当前条件未匹配到具体阵容数据。不要编造具体阵容、版本、数值或运营节奏，只能说明当前知识库没有命中，并提示玩家补充英雄、装备、羁绊或重新更新知识库。\n")
}

func writeTaskInstruction(sb *strings.Builder, ctx Context, input *NluEnrichedContext) {
	sb.WriteString("\n## 你的任务\n")
	sb.WriteString("先给玩家明确结论，再给理由和操作；阵容强度结论必须对应上方统计数据，版本原因可以引用官方版本环境。")
	sb.WriteString(buildFeedbackInstruction(input.Feedback))
	sb.WriteString(buildInstruction(ctx))
}
