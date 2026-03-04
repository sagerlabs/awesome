package tool

import (
	"context"
	"github.com/sagerlabs/awesome/tft/data"
	"sort"
)

// CompTierTool Tool3：查询阵容强度
type CompTierTool struct {
	store *data.Store
}

func NewCompTierTool(store *data.Store) *CompTierTool {
	return &CompTierTool{store: store}
}

// Query 查询指定 cluster_id 列表的阵容强度
func (t *CompTierTool) Query(ctx context.Context, in *data.CompTierInput) (*data.CompTierOutput, error) {
	var entries []data.CompTierEntry
	for _, cid := range in.ClusterIDs {
		comp, ok := t.store.GetCompByClusterID(cid)
		if !ok {
			continue
		}
		entries = append(entries, data.CompTierEntry{
			ClusterID:    comp.ClusterID,
			Tier:         comp.Tier,
			AvgPlacement: comp.AvgPlacement,
			Top4Rate:     comp.Top4Rate,
			WinRate:      comp.WinRate,
		})
	}
	return &data.CompTierOutput{Tiers: entries}, nil
}

// QueryTopTier 查询当前版本所有 S/A Tier 阵容强度，作为参考基准
// 在 Graph 的并行节点中，CompTier 节点无需等英雄/装备结果，直接提供版本基准数据
func (t *CompTierTool) QueryTopTier(ctx context.Context) (*data.CompTierOutput, error) {
	comps := t.store.GetCompsByTier("S", "A")

	entries := make([]data.CompTierEntry, 0, len(comps))
	for _, c := range comps {
		entries = append(entries, data.CompTierEntry{
			ClusterID:    c.ClusterID,
			Tier:         c.Tier,
			AvgPlacement: c.AvgPlacement,
			Top4Rate:     c.Top4Rate,
			WinRate:      c.WinRate,
		})
	}

	// 按 AvgPlacement 升序（越强越靠前）
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AvgPlacement < entries[j].AvgPlacement
	})

	return &data.CompTierOutput{Tiers: entries}, nil
}

// =============================================================================
// IntersectionCalc 交集计算器
// 把三路 Tool 的输出合并，计算每个阵容的综合置信度，输出排序后的推荐列表
// =============================================================================

// IntersectionCalc 交集计算器
type IntersectionCalc struct {
	store *data.Store
}

func NewIntersectionCalc(store *data.Store) *IntersectionCalc {
	return &IntersectionCalc{store: store}
}

// Compute 核心方法：三路结果求交集，生成置信度排序的推荐列表
//
// 置信度计算规则：
//
//	三路全命中（hero + item + tier）：confidence = 0.9 + avgPlacementBonus
//	两路命中：                        confidence = 0.65
//	仅一路命中：                      confidence = 0.40
//	零命中（无交集）：                 触发兜底，返回当前版本最强阵容
func (c *IntersectionCalc) Compute(in *data.IntersectionInput) (*data.IntersectionOutput, error) {
	// 建立各路命中的 clusterID 集合
	heroHits := buildHeroHitMap(in.HeroComps) // clusterID -> CompMatch
	itemHits := buildItemHitMap(in.ItemFits)  // clusterID -> ItemFitResult
	tierSet := buildTierSet(in.CompTiers)     // clusterID -> CompTierEntry

	// 收集所有候选 clusterID（三路的并集）
	candidates := unionKeys(heroHits, itemHits)

	var recs []data.Recommendation

	for clusterID := range candidates {
		comp, ok := c.store.GetCompByClusterID(clusterID)
		if !ok {
			continue
		}

		rec := data.Recommendation{Comp: *comp}

		// 统计命中来源
		if hm, ok := heroHits[clusterID]; ok {
			rec.HitSources = append(rec.HitSources, "hero")
			rec.MatchedUnits = c.toChineseNames(hm.MatchedUnits)
			rec.MissingUnits = c.toChineseNames(hm.MissingUnits)
		}
		if im, ok := itemHits[clusterID]; ok {
			rec.HitSources = append(rec.HitSources, "item")
			rec.MatchedItems = c.toChineseNames(im.MatchedItems)
			// carry 优先取装备推荐的 carry
			rec.SuggestedCarry = c.store.IDToCN(im.Carry)
		}
		if _, ok := tierSet[clusterID]; ok {
			rec.HitSources = append(rec.HitSources, "tier")
		}

		// 如果没有 carry 建议，取阵容默认的 best_build carry
		if rec.SuggestedCarry == "" && comp.BestBuild.Carry != "" {
			rec.SuggestedCarry = c.store.IDToCN(comp.BestBuild.Carry)
		}

		// 计算置信度
		rec.Confidence = c.calcConfidence(rec.HitSources, comp)
		rec.ConfidenceDesc = rec.CalcConfidenceDesc()

		recs = append(recs, rec)
	}

	// 零交集兜底：返回版本最强的前3个阵容
	if len(recs) == 0 {
		recs = c.fallback(in.CompTiers)
	}

	// 按置信度降序，置信度相同时按 AvgPlacement 升序（越强越靠前）
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Confidence != recs[j].Confidence {
			return recs[i].Confidence > recs[j].Confidence
		}
		return recs[i].Comp.AvgPlacement < recs[j].Comp.AvgPlacement
	})

	// 最多返回 5 条推荐
	if len(recs) > 5 {
		recs = recs[:5]
	}

	return &data.IntersectionOutput{
		Recommendations: recs,
		UserInput:       in.UserInput,
	}, nil
}

// calcConfidence 根据命中来源数量和阵容强度计算置信度
//
//	3路命中：基础 0.90，S Tier +0.05
//	2路命中：基础 0.65，S Tier +0.05，A Tier +0.02
//	1路命中：基础 0.40
func (c *IntersectionCalc) calcConfidence(sources []string, comp *data.Comp) float64 {
	var base float64
	switch len(sources) {
	case 3:
		base = 0.90
	case 2:
		base = 0.65
	default:
		base = 0.40
	}

	// Tier 加成
	switch comp.Tier {
	case "S":
		base += 0.05
	case "A":
		if len(sources) >= 2 {
			base += 0.02
		}
	}

	if base > 1.0 {
		base = 1.0
	}
	return base
}

// fallback 无交集时的兜底：返回版本当前最强的前3个阵容，置信度标记为低
func (c *IntersectionCalc) fallback(tiers data.CompTierOutput) []data.Recommendation {
	var recs []data.Recommendation
	limit := 3
	for _, entry := range tiers.Tiers {
		if len(recs) >= limit {
			break
		}
		comp, ok := c.store.GetCompByClusterID(entry.ClusterID)
		if !ok {
			continue
		}
		recs = append(recs, data.Recommendation{
			Comp:           *comp,
			Confidence:     0.30,
			ConfidenceDesc: "低",
			HitSources:     []string{"fallback"},
			SuggestedCarry: c.store.IDToCN(comp.BestBuild.Carry),
		})
	}
	return recs
}

// toChineseNames 将一组 TFT ID 转换为中文名列表
func (c *IntersectionCalc) toChineseNames(ids []string) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, c.store.IDToCN(id))
	}
	return names
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

func buildHeroHitMap(out data.HeroCompsOutput) map[string]data.CompMatch {
	m := make(map[string]data.CompMatch, len(out.Matches))
	for _, match := range out.Matches {
		m[match.Comp.ClusterID] = match
	}
	return m
}

func buildItemHitMap(out data.ItemFitOutput) map[string]data.ItemFitResult {
	m := make(map[string]data.ItemFitResult, len(out.Results))
	for _, r := range out.Results {
		// 同一阵容可能有多个 carry，取 TotalScore 最高的那条
		if existing, ok := m[r.ClusterID]; !ok || r.TotalScore > existing.TotalScore {
			m[r.ClusterID] = r
		}
	}
	return m
}

func buildTierSet(out data.CompTierOutput) map[string]data.CompTierEntry {
	m := make(map[string]data.CompTierEntry, len(out.Tiers))
	for _, t := range out.Tiers {
		m[t.ClusterID] = t
	}
	return m
}

// unionKeys 返回两个 map 的 key 并集
func unionKeys(
	heroHits map[string]data.CompMatch,
	itemHits map[string]data.ItemFitResult,
) map[string]struct{} {
	union := make(map[string]struct{})
	for k := range heroHits {
		union[k] = struct{}{}
	}
	for k := range itemHits {
		union[k] = struct{}{}
	}
	return union
}
