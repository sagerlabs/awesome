package agent

import (
	"fmt"
	"strings"
)

// buildDecisionPolicyHints returns localised coaching hints based on the current game state.
// Returns nil when no GameState fields are present so the section can be omitted.
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
