package agent

import (
	"fmt"
	"github.com/sagerlabs/awesome/tft/data"
)

// ── Prompt 构建 ───────────────────────────────────────────────────────────────

const systemPrompt = `/no_think 你是一位云顶之弈高段位教练，擅长根据玩家当前棋盘状态给出精准的阵容推荐和运营建议。
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
