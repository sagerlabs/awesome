package tool

import (
	"context"
	"sort"

	"github.com/sagerlabs/awesome/tft/data"
)

// PriorityRecommendTool 优先级推荐工具
type PriorityRecommendTool struct {
	store *data.Store
}

func NewPriorityRecommendTool(store *data.Store) *PriorityRecommendTool {
	return &PriorityRecommendTool{store: store}
}

// =============================================================================
// 1. 通过英雄选择阵容的优先级
// =============================================================================

// HeroPriorityInput 英雄优先级输入
type HeroPriorityInput struct {
	HeroIDs       []string // 用户输入的英雄ID
	EnableUnlock  bool     // 是否启用解锁机制（S16）
	UnlockHeroIDs []string // 能解锁高费卡的英雄ID
}

// HeroPriorityOutput 英雄优先级输出
type HeroPriorityOutput struct {
	Recommendations []ScoredComp // 推荐的阵容（按优先级排序）
	HasSTier        bool          // 是否包含S梯队阵容
}

// ScoredComp 带评分的阵容
type ScoredComp struct {
	Comp        data.Comp // 阵容信息
	Score       float64   // 综合评分
	IsMainCarry bool      // 输入英雄是否为主C
	Source      string    // 来源："hero" / "item"
}

// SelectCompsByHeroWithPriority 根据英雄优先级选择阵容
func (t *PriorityRecommendTool) SelectCompsByHeroWithPriority(ctx context.Context, in *HeroPriorityInput) (*HeroPriorityOutput, error) {
	var allMatches []data.CompMatch

	// 1. 查询包含输入英雄的阵容
	if len(in.HeroIDs) > 0 {
		allMatches = t.store.GetCompsByUnits(in.HeroIDs)
	}

	// 2. 如果启用了解锁机制，查询解锁英雄相关的阵容
	var unlockMatches []data.CompMatch
	if in.EnableUnlock && len(in.UnlockHeroIDs) > 0 {
		unlockMatches = t.store.GetCompsByUnits(in.UnlockHeroIDs)
	}

	// 3. 合并并去重
	allMatches = t.mergeAndDedupMatches(allMatches, unlockMatches)

	// 4. 评分和排序
	var scoredComps []ScoredComp
	for _, match := range allMatches {
		score := t.calcHeroScore(match, in.HeroIDs)
		isMainCarry := t.isHeroMainCarry(match.Comp, in.HeroIDs)

		scoredComps = append(scoredComps, ScoredComp{
			Comp:        match.Comp,
			Score:       score,
			IsMainCarry: isMainCarry,
			Source:      "hero",
		})
	}

	// 5. 按优先级排序：主C优先 > 评分 > 阵容等级
	sort.Slice(scoredComps, func(i, j int) bool {
		// 主C优先
		if scoredComps[i].IsMainCarry != scoredComps[j].IsMainCarry {
			return scoredComps[i].IsMainCarry
		}
		// 评分高的优先
		if scoredComps[i].Score != scoredComps[j].Score {
			return scoredComps[i].Score > scoredComps[j].Score
		}
		// 阵容等级高的优先
		return tierToScore(scoredComps[i].Comp.Tier) > tierToScore(scoredComps[j].Comp.Tier)
	})

	// 6. 检查是否有S梯队
	hasSTier := false
	for _, sc := range scoredComps {
		if sc.Comp.Tier == "S" {
			hasSTier = true
			break
		}
	}

	// 7. 如果启用解锁机制且没有S梯队，则只保留当前英雄筛选的阵容
	if in.EnableUnlock && !hasSTier {
		scoredComps = t.filterByHeroIDs(scoredComps, in.HeroIDs)
	}

	// 8. 取前三套
	if len(scoredComps) > 3 {
		scoredComps = scoredComps[:3]
	}

	return &HeroPriorityOutput{
		Recommendations: scoredComps,
		HasSTier:        hasSTier,
	}, nil
}

// calcHeroScore 计算英雄匹配评分
func (t *PriorityRecommendTool) calcHeroScore(match data.CompMatch, heroIDs []string) float64 {
	score := match.MatchScore

	// 阵容等级加权
	score *= tierWeight(match.Comp.Tier)

	// 主C额外加分
	if t.isHeroMainCarry(match.Comp, heroIDs) {
		score += 0.3
	}

	return score
}

// isHeroMainCarry 检查输入英雄是否是阵容的主C
func (t *PriorityRecommendTool) isHeroMainCarry(comp data.Comp, heroIDs []string) bool {
	for _, heroID := range heroIDs {
		if comp.BestBuild.Carry == heroID {
			return true
		}
		// 检查all_builds中是否有该英雄作为carry
		for _, build := range comp.AllBuilds {
			if build.Carry == heroID {
				return true
			}
		}
	}
	return false
}

// mergeAndDedupMatches 合并并去重匹配结果
func (t *PriorityRecommendTool) mergeAndDedupMatches(matches1, matches2 []data.CompMatch) []data.CompMatch {
	seen := make(map[string]bool)
	var result []data.CompMatch

	for _, m := range matches1 {
		if !seen[m.Comp.ClusterID] {
			seen[m.Comp.ClusterID] = true
			result = append(result, m)
		}
	}
	for _, m := range matches2 {
		if !seen[m.Comp.ClusterID] {
			seen[m.Comp.ClusterID] = true
			result = append(result, m)
		}
	}
	return result
}

// filterByHeroIDs 只保留包含指定英雄的阵容
func (t *PriorityRecommendTool) filterByHeroIDs(comps []ScoredComp, heroIDs []string) []ScoredComp {
	var result []ScoredComp
	for _, sc := range comps {
		for _, heroID := range heroIDs {
			for _, unit := range sc.Comp.Units {
				if unit == heroID {
					result = append(result, sc)
					goto next
				}
			}
		}
	next:
	}
	return result
}

// =============================================================================
// 2. 通过装备选择阵容的优先级
// =============================================================================

// ItemPriorityInput 装备优先级输入
type ItemPriorityInput struct {
	ItemIDs          []string // 用户输入的装备ID
	HeroHasSTier     bool     // 英雄筛选是否有S梯队
	HeroComps        []ScoredComp // 英雄筛选出的阵容
}

// ItemPriorityOutput 装备优先级输出
type ItemPriorityOutput struct {
	Recommendations []ScoredComp // 推荐的阵容（按优先级排序）
}

// SelectCompsByItemWithPriority 根据装备优先级选择阵容
func (t *PriorityRecommendTool) SelectCompsByItemWithPriority(ctx context.Context, in *ItemPriorityInput) (*ItemPriorityOutput, error) {
	var itemComps []ScoredComp

	// 1. 查询装备适配的阵容
	if len(in.ItemIDs) > 0 {
		itemResults := t.store.GetItemFitByItems(in.ItemIDs)

		for _, itemResult := range itemResults {
			if comp, ok := t.store.GetCompByClusterID(itemResult.ClusterID); ok {
				// 根据情况筛选
				if in.HeroHasSTier && comp.Tier != "S" {
					continue // 英雄有S梯队，只看S梯队装备适配
				}

				score := float64(itemResult.TotalScore) / 100.0
				itemComps = append(itemComps, ScoredComp{
					Comp:   *comp,
					Score:  score,
					Source: "item",
				})
			}
		}
	}

	// 2. 合并英雄和装备推荐，英雄优先级 > 装备S梯队优先级
	var allComps []ScoredComp

	// 先加英雄推荐
	allComps = append(allComps, in.HeroComps...)

	// 再加装备推荐（但要去重）
	seen := make(map[string]bool)
	for _, sc := range in.HeroComps {
		seen[sc.Comp.ClusterID] = true
	}
	for _, sc := range itemComps {
		if !seen[sc.Comp.ClusterID] {
			seen[sc.Comp.ClusterID] = true
			allComps = append(allComps, sc)
		}
	}

	// 3. 排序：英雄来源 > 装备来源 > 评分
	sort.Slice(allComps, func(i, j int) bool {
		// 英雄来源优先
		if allComps[i].Source != allComps[j].Source {
			return allComps[i].Source == "hero"
		}
		// 评分高的优先
		return allComps[i].Score > allComps[j].Score
	})

	// 4. 取前三套
	if len(allComps) > 3 {
		allComps = allComps[:3]
	}

	return &ItemPriorityOutput{
		Recommendations: allComps,
	}, nil
}

// =============================================================================
// 3. 羁绊梯队索引
// =============================================================================

// TraitTierEntry 羁绊梯队条目
type TraitTierEntry struct {
	TraitID    string  // 羁绊ID
	TierScore  float64 // 梯队评分
	Appearance int     // 出现次数
	AvgWinRate float64 // 平均胜率
	AvgTop4Rate float64 // 平均前4率
}

// TraitTierIndex 羁绊梯队索引
type TraitTierIndex struct {
	Tiers []TraitTierEntry // 按评分排序的羁绊列表
}

// BuildTraitTierIndex 构建羁绊梯队索引
func (t *PriorityRecommendTool) BuildTraitTierIndex() *TraitTierIndex {
	// 1. 获取所有S梯队阵容
	sComps := t.store.GetCompsByTier("S")

	// 2. 统计每个羁绊的信息
	traitStats := make(map[string]*struct {
		count     int
		totalWin  float64
		totalTop4 float64
		tierSum   int
	})

	for _, comp := range sComps {
		tierScore := tierToScore(comp.Tier)
		for _, trait := range comp.Traits {
			stats, ok := traitStats[trait]
			if !ok {
				stats = &struct {
					count     int
					totalWin  float64
					totalTop4 float64
					tierSum   int
				}{}
				traitStats[trait] = stats
			}
			stats.count++
			stats.totalWin += comp.WinRate
			stats.totalTop4 += comp.Top4Rate
			stats.tierSum += int(tierScore)
		}
	}

	// 3. 构建结果
	var index TraitTierIndex
	for traitID, stats := range traitStats {
		entry := TraitTierEntry{
			TraitID:     traitID,
			Appearance:  stats.count,
			AvgWinRate:  stats.totalWin / float64(stats.count),
			AvgTop4Rate: stats.totalTop4 / float64(stats.count),
		}
		// 综合评分 = 出现次数 * 0.4 + 胜率 * 0.4 + 前4率 * 0.2
		entry.TierScore = float64(stats.count)*0.4 + entry.AvgWinRate*0.4 + entry.AvgTop4Rate*0.2
		index.Tiers = append(index.Tiers, entry)
	}

	// 4. 排序：梯队评分 > 胜率 > 前4率
	sort.Slice(index.Tiers, func(i, j int) bool {
		if index.Tiers[i].TierScore != index.Tiers[j].TierScore {
			return index.Tiers[i].TierScore > index.Tiers[j].TierScore
		}
		if index.Tiers[i].AvgWinRate != index.Tiers[j].AvgWinRate {
			return index.Tiers[i].AvgWinRate > index.Tiers[j].AvgWinRate
		}
		return index.Tiers[i].AvgTop4Rate > index.Tiers[j].AvgTop4Rate
	})

	return &index
}

// =============================================================================
// 辅助函数
// =============================================================================

// tierToScore 阵容等级转数值
func tierToScore(tier string) int {
	switch tier {
	case "S":
		return 4
	case "A":
		return 3
	case "B":
		return 2
	case "C":
		return 1
	default:
		return 0
	}
}

// tierWeight 阵容等级权重
func tierWeight(tier string) float64 {
	switch tier {
	case "S":
		return 1.5
	case "A":
		return 1.2
	case "B":
		return 1.0
	case "C":
		return 0.8
	default:
		return 1.0
	}
}
