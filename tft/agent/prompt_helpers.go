package agent

import (
	"fmt"
	"strings"
)

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

// buildItemMatchIndex maps cluster IDs to formatted item-carry strings for comp sections.
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
