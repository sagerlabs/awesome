package agent

// buildInstruction returns intent-specific output directives appended to the task section.
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
