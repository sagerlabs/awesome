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

	// 1. 查询匹配的阵容（根据英雄）- 直接用原始名称去Resolve
	if len(ctx.Champions) > 0 {
		heroIDs := make([]string, 0, len(ctx.Champions))
		for name := range ctx.Champions {
			if id := store.ResolveUnitID(name); id != "" {
				heroIDs = append(heroIDs, id)
			}
		}
		if len(heroIDs) > 0 {
			matches := store.GetCompsByUnits(heroIDs)
			for _, match := range matches {
				result.MatchedComps = append(result.MatchedComps, match.Comp)
			}
		}
	}

	// 2. 查询匹配的装备 - 直接用原始名称去Resolve
	if len(ctx.Items) > 0 {
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

	// 3. 将Context中的名称转换为中文（用于展示给LLM）
	result.Ctx = normalizeContext(ctx, store)

	return result
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
