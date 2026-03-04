package tool

import (
	"context"
	"sort"

	"github.com/sagerlabs/awesome/tft/data"
)

// HeroCompsTool Tool1：根据英雄列表推荐最匹配的阵容
type HeroCompsTool struct {
	store *data.Store
}

func NewHeroCompsTool(store *data.Store) *HeroCompsTool {
	return &HeroCompsTool{store: store}
}

// Query 输入英雄 ID 列表，返回匹配度最高的 TopN 个阵容
func (t *HeroCompsTool) Query(ctx context.Context, in *data.HeroCompsInput) (*data.HeroCompsOutput, error) {
	topN := in.TopN
	if topN <= 0 {
		topN = 5
	}

	// 1. 从 Store 查询包含这些英雄的阵容，已按 MatchScore 降序
	matches := t.store.GetCompsByUnits(in.Heroes)

	// 2. Tier 加权：S Tier 阵容额外加分，避免低强度阵容因英雄命中多而排在前面
	for i := range matches {
		matches[i].MatchScore = t.weightByTier(matches[i].MatchScore, matches[i].Comp.Tier)
	}

	// 3. 加权后重新排序
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].MatchScore > matches[j].MatchScore
	})

	// 4. 截取 TopN
	if len(matches) > topN {
		matches = matches[:topN]
	}

	return &data.HeroCompsOutput{Matches: matches}, nil
}

// weightByTier 根据阵容 Tier 对 matchScore 加权
// S Tier * 1.2，A Tier * 1.1，B/C Tier 不加权
// 保证强势阵容在匹配度相近时优先展示
func (t *HeroCompsTool) weightByTier(score float64, tier string) float64 {
	switch tier {
	case "S":
		return score * 1.2
	case "A":
		return score * 1.1
	default:
		return score
	}
}
