package agent

import (
	"github.com/sagerlabs/awesome/tft/data"
)

type NluContext struct {
	UserInput  string
	Ctx        Context
	FinalReply string
}

// NluEnrichedContext NLU enriched context with data lookup results
type NluEnrichedContext struct {
	UserInput    string                 `json:"user_input"`
	Ctx          Context                `json:"ctx"`
	MatchedComps []data.Comp           `json:"matched_comps"` // 匹配的阵容
	MatchedItems []MatchedItemInfo      `json:"matched_items"` // 匹配的装备信息
}

// MatchedItemInfo 匹配的装备信息
type MatchedItemInfo struct {
	ItemID    string              `json:"item_id"`
	ItemName  string              `json:"item_name"`
	CompInfos []ItemFitCompInfo   `json:"comp_infos"` // 适配该装备的阵容
}

// ItemFitCompInfo 装备适配的阵容信息
type ItemFitCompInfo struct {
	ClusterID   string  `json:"cluster_id"`
	CompName    string  `json:"comp_name"`
	CompTier    string  `json:"comp_tier"`
	CompAvg     float64 `json:"comp_avg"`
	Carry       string  `json:"carry"`
	CarryName   string  `json:"carry_name"`
	PriorityScore int   `json:"priority_score"`
}

// QueryNLUData NLU数据查询的单独入口
// 输入：NLU提取的Context
// 输出： enriched context（包含查询到的阵容、装备等数据）
func QueryNLUData(ctx Context, store *data.Store) *NluEnrichedContext {
	result := &NluEnrichedContext{
		Ctx: ctx,
	}

	// TODO: 散件合成逻辑
	// 当用户只输入散件（如女神之泪、反曲之弓等基础装备）时：
	// 1. 需要散件合成列表，知道该散件可以合成哪些成品装备
	// 2. 找到包含了该散件合成的最强装备的阵容
	// 3. 返回最强的三套阵容
	// 当用户同时输入了英雄和散件时：
	// 1. 先找英雄最强的三套阵容
	// 2. 然后看看哪些必须的装备是由该散件合成的
	// 3. 如果没有就是不合适

	hasHeroes := len(ctx.Champions) > 0
	hasItems := len(ctx.Items) > 0

	// 1. 处理装备查询
	if hasItems {
		for _, itemName := range ctx.Items {
			if itemID := store.ResolveItemID(itemName); itemID != "" {
				itemInfo := MatchedItemInfo{
					ItemID:   itemID,
					ItemName: store.IDToCN(itemID),
				}
				entries := store.GetItemFitEntries(itemID)
				for _, entry := range entries {
					compInfo := ItemFitCompInfo{
						ClusterID:    entry.ClusterID,
						CompName:     entry.CompName,
						CompTier:     entry.CompTier,
						CompAvg:      entry.CompAvg,
						Carry:        entry.Carry,
						CarryName:    store.IDToCN(entry.Carry),
						PriorityScore: entry.PriorityScore,
					}
					itemInfo.CompInfos = append(itemInfo.CompInfos, compInfo)
				}
				result.MatchedItems = append(result.MatchedItems, itemInfo)
			}
		}
	}

	// 2. 处理阵容查询
	if hasHeroes {
		// 有英雄输入：根据英雄查询阵容
		heroIDs := make([]string, 0, len(ctx.Champions))
		for name := range ctx.Champions {
			if id := store.ResolveUnitID(name); id != "" {
				heroIDs = append(heroIDs, id)
			}
		}
		if len(heroIDs) > 0 {
			matches := store.GetCompsByUnits(heroIDs)
			// 最多取3个阵容
			for i, match := range matches {
				if i >= 3 {
					break
				}
				result.MatchedComps = append(result.MatchedComps, match.Comp)
			}
		}
	} else if hasItems {
		// 没有英雄输入，但有装备输入：根据装备查询最强三套阵容
		// 直接从所有装备适配的阵容中去重并获取完整阵容
		clusterIDSet := make(map[string]bool)
		var compsToAdd []data.Comp

		for _, item := range result.MatchedItems {
			for _, compInfo := range item.CompInfos {
				if !clusterIDSet[compInfo.ClusterID] {
					clusterIDSet[compInfo.ClusterID] = true
					if comp, ok := store.GetCompByClusterID(compInfo.ClusterID); ok {
						compsToAdd = append(compsToAdd, *comp)
					}
				}
			}
		}

		// 按平均排名升序排序（越小越强）
		sortCompsByAvgPlacement(compsToAdd)

		// 取前3个
		for i, comp := range compsToAdd {
			if i >= 3 {
				break
			}
			result.MatchedComps = append(result.MatchedComps, comp)
		}
	}

	// 3. 将Context中的名称转换为中文（用于展示给LLM）
	result.Ctx = normalizeContext(ctx, store)

	return result
}

// sortItemFitCompInfosByAvg 按平均排名升序排序装备适配阵容信息
func sortItemFitCompInfosByAvg(infos []ItemFitCompInfo) {
	for i := 1; i < len(infos); i++ {
		for j := i; j > 0 && infos[j].CompAvg < infos[j-1].CompAvg; j-- {
			infos[j], infos[j-1] = infos[j-1], infos[j]
		}
	}
}

// sortCompsByAvgPlacement 按平均排名升序排序阵容
func sortCompsByAvgPlacement(comps []data.Comp) {
	for i := 1; i < len(comps); i++ {
		for j := i; j > 0 && comps[j].AvgPlacement < comps[j-1].AvgPlacement; j-- {
			comps[j], comps[j-1] = comps[j-1], comps[j]
		}
	}
}

// normalizeContext 尝试将Context中的黑话/昵称转换为标准名称
func normalizeContext(ctx Context, store *data.Store) Context {
	result := ctx

	// 规范化英雄名称（尝试用store.ResolveUnitID解析）
	if len(ctx.Champions) > 0 {
		normalizedChampions := make(map[string]int8)
		for name, star := range ctx.Champions {
			if id := store.ResolveUnitID(name); id != "" {
				normalizedChampions[store.IDToCN(id)] = star
			} else {
				normalizedChampions[name] = star
			}
		}
		result.Champions = normalizedChampions
	}

	// 规范化装备名称
	if len(ctx.Items) > 0 {
		normalizedItems := make([]string, 0, len(ctx.Items))
		for _, item := range ctx.Items {
			if id := store.ResolveItemID(item); id != "" {
				normalizedItems = append(normalizedItems, store.IDToCN(id))
			} else {
				normalizedItems = append(normalizedItems, item)
			}
		}
		result.Items = normalizedItems
	}

	// 规范化羁绊名称
	if len(ctx.Traits) > 0 {
		normalizedTraits := make([]string, 0, len(ctx.Traits))
		for _, trait := range ctx.Traits {
			normalizedTraits = append(normalizedTraits, store.IDToCN(trait))
		}
		result.Traits = normalizedTraits
	}

	return result
}
