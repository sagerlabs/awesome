package tool

import (
	"context"
	"github.com/sagerlabs/awesome/tft/data"
	"sort"
)

// ItemFitTool Tool2：根据装备列表推荐最适合的阵容
type ItemFitTool struct {
	store *data.Store
}

func NewItemFitTool(store *data.Store) *ItemFitTool {
	return &ItemFitTool{store: store}
}

// Query 输入装备 ID 列表，返回最适配的阵容
//
// 核心逻辑：
//  1. 查询每个装备适配的阵容列表（带 priority_score）
//  2. 同一阵容被多个装备命中时，分数累加合并
//  3. Tier 加权后排序，取前 TopN
func (t *ItemFitTool) Query(ctx context.Context, in *data.ItemFitInput) (*data.ItemFitOutput, error) {
	if len(in.Items) == 0 {
		return &data.ItemFitOutput{}, nil
	}

	// 1. 批量查询并聚合：同阵容同 carry 的多个装备合并累加分数
	results := t.store.GetItemFitByItems(in.Items)

	// 2. 补全阵容详情（Store 返回的 ItemFitResult 只有 cluster_id，需要补 top4_rate 等）
	results = t.enrichWithCompDetail(results)

	// 3. Tier 加权
	for i := range results {
		results[i].TotalScore = t.weightByTier(results[i].TotalScore, results[i].CompTier)
	}

	// 4. 按 TotalScore 降序
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	// 5. 取前 5 个
	if len(results) > 5 {
		results = results[:5]
	}

	return &data.ItemFitOutput{Results: results}, nil
}

// enrichWithCompDetail 从 Store 补全 ItemFitResult 里缺失的 CompTier/CompAvg
// items_priority.json 已经存了 comp_tier 和 comp_avg，这里主要做二次校验和修正
func (t *ItemFitTool) enrichWithCompDetail(results []data.ItemFitResult) []data.ItemFitResult {
	for i, r := range results {
		comp, ok := t.store.GetCompByClusterID(r.ClusterID)
		if !ok {
			continue
		}
		// 以 Store 中的实时数据为准（比 items_priority.json 里的更准确）
		results[i].CompTier = comp.Tier
		results[i].CompAvg = comp.AvgPlacement
	}
	return results
}

// weightByTier 装备适配的 Tier 加权，逻辑同 HeroCompsTool
func (t *ItemFitTool) weightByTier(score int, tier string) int {
	switch tier {
	case "S":
		return int(float64(score) * 1.2)
	case "A":
		return int(float64(score) * 1.1)
	default:
		return score
	}
}
