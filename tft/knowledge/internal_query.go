package knowledge

import "github.com/sagerlabs/awesome/tft/data"

// =============================================================================
// 内部类型（避免引用agent包）
// =============================================================================

// internalContext 内部使用的Context类型（避免引用agent包）
type internalContext struct {
	Intent          string              `json:"intent"`
	Champions       map[string]int8     `json:"champions"`
	Items           []string            `json:"items"`
	Traits          []string            `json:"traits"`
	Augments        []string            `json:"augments"`
	ExplicitLineup  *string             `json:"explicit_lineup"`
	InferredLineup  string              `json:"inferred_lineup"`
	Playstyle       string              `json:"playstyle"`
	GameStage       *string             `json:"game_stage"`
	Gold            *int                `json:"gold"`
	Level           *int                `json:"level"`
	HP              *int                `json:"hp"`
}

// internalNluEnrichedContext 内部使用的NluEnrichedContext类型
type internalNluEnrichedContext struct {
	UserInput    string                `json:"user_input"`
	Ctx          internalContext       `json:"ctx"`
	MatchedComps []data.Comp           `json:"matched_comps"`
	MatchedItems []internalMatchedItemInfo `json:"matched_items"`
}

// internalMatchedItemInfo 内部使用的MatchedItemInfo类型
type internalMatchedItemInfo struct {
	ItemID    string                `json:"item_id"`
	ItemName  string                `json:"item_name"`
	CompInfos []internalItemFitCompInfo `json:"comp_infos"`
}

// internalItemFitCompInfo 内部使用的ItemFitCompInfo类型
type internalItemFitCompInfo struct {
	ClusterID    string  `json:"cluster_id"`
	CompName     string  `json:"comp_name"`
	CompTier     string  `json:"comp_tier"`
	CompAvg      float64 `json:"comp_avg"`
	Carry        string  `json:"carry"`
	CarryName    string  `json:"carry_name"`
	PriorityScore int    `json:"priority_score"`
}

// =============================================================================
// 内部查询逻辑（避免引用agent包）
// =============================================================================

// internalQueryNLUData 内部NLU数据查询逻辑
func internalQueryNLUData(ctx internalContext, store *data.Store) *internalNluEnrichedContext {
	result := &internalNluEnrichedContext{
		Ctx: ctx,
	}

	hasHeroes := len(ctx.Champions) > 0
	hasItems := len(ctx.Items) > 0

	// 1. 处理装备查询
	if hasItems {
		for _, itemName := range ctx.Items {
			if itemID := store.ResolveItemID(itemName); itemID != "" {
				itemInfo := internalMatchedItemInfo{
					ItemID:   itemID,
					ItemName: store.IDToCN(itemID),
				}
				entries := store.GetItemFitEntries(itemID)
				for _, entry := range entries {
					compInfo := internalItemFitCompInfo{
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
		internalSortCompsByAvgPlacement(compsToAdd)

		// 取前3个
		for i, comp := range compsToAdd {
			if i >= 3 {
				break
			}
			result.MatchedComps = append(result.MatchedComps, comp)
		}
	}

	// 3. 将Context中的名称转换为中文（用于展示给LLM）
	result.Ctx = internalNormalizeContext(ctx, store)

	return result
}

// internalSortCompsByAvgPlacement 按平均排名升序排序阵容
func internalSortCompsByAvgPlacement(comps []data.Comp) {
	for i := 1; i < len(comps); i++ {
		for j := i; j > 0 && comps[j].AvgPlacement < comps[j-1].AvgPlacement; j-- {
			comps[j], comps[j-1] = comps[j-1], comps[j]
		}
	}
}

// internalNormalizeContext 尝试将Context中的黑话/昵称转换为标准名称
func internalNormalizeContext(ctx internalContext, store *data.Store) internalContext {
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
