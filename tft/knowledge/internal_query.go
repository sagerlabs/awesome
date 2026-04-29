package knowledge

import (
	"strings"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
)

// internalQueryNLUData 使用共享 contract 作为 knowledge 内部 service 的输入输出。
// 这样 knowledge 仍然是强类型实现，但不再维护一份与 agent 平行演进的镜像 schema。
func internalQueryNLUData(ctx contracts.QueryNLURequest, store *data.Store) *contracts.QueryNLUResponse {
	result := &contracts.QueryNLUResponse{
		Ctx: ctx,
	}

	hasHeroes := len(ctx.Champions) > 0
	hasItems := len(ctx.Items) > 0

	if hasItems {
		for _, itemName := range ctx.Items {
			if itemID := store.ResolveItemID(itemName); itemID != "" {
				itemInfo := contracts.MatchedItemInfo{
					ItemID:   itemID,
					ItemName: store.IDToCN(itemID),
				}
				entries := store.GetItemFitEntries(itemID)
				for _, entry := range entries {
					itemInfo.CompInfos = append(itemInfo.CompInfos, contracts.ItemFitCompInfo{
						ClusterID:     entry.ClusterID,
						CompName:      localizeCompName(entry.CompName, store),
						CompTier:      entry.CompTier,
						CompAvg:       entry.CompAvg,
						Carry:         entry.Carry,
						CarryName:     store.IDToCN(entry.Carry),
						PriorityScore: entry.PriorityScore,
					})
				}
				result.MatchedItems = append(result.MatchedItems, itemInfo)
			}
		}
	}

	if hasHeroes {
		heroIDs := make([]string, 0, len(ctx.Champions))
		for name := range ctx.Champions {
			if id := store.ResolveUnitID(name); id != "" {
				heroIDs = append(heroIDs, id)
			}
		}
		if len(heroIDs) > 0 {
			matches := store.GetCompsByUnits(heroIDs)
			for i, match := range matches {
				if i >= 3 {
					break
				}
				result.MatchedComps = append(result.MatchedComps, toCompSummary(match.Comp, store))
			}
		}
	} else if hasItems {
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

		internalSortCompsByAvgPlacement(compsToAdd)

		for i, comp := range compsToAdd {
			if i >= 3 {
				break
			}
			result.MatchedComps = append(result.MatchedComps, toCompSummary(comp, store))
		}
	} else if shouldReturnTopComps(ctx) {
		topComps := store.GetCompsByTier("S", "A")
		internalSortCompPointersByAvgPlacement(topComps)
		for i, comp := range topComps {
			if i >= 3 {
				break
			}
			result.MatchedComps = append(result.MatchedComps, toCompSummary(*comp, store))
		}
	}

	result.Ctx = internalNormalizeContext(ctx, store)
	return result
}

func shouldReturnTopComps(ctx contracts.QueryNLURequest) bool {
	if len(ctx.Traits) > 0 || len(ctx.Augments) > 0 {
		return false
	}
	if ctx.ExplicitLineup != nil && strings.TrimSpace(*ctx.ExplicitLineup) != "" {
		return false
	}
	switch ctx.Intent {
	case "", "lineup_recommend", "playstyle_query":
		return true
	default:
		return false
	}
}

func internalSortCompsByAvgPlacement(comps []data.Comp) {
	for i := 1; i < len(comps); i++ {
		for j := i; j > 0 && comps[j].AvgPlacement < comps[j-1].AvgPlacement; j-- {
			comps[j], comps[j-1] = comps[j-1], comps[j]
		}
	}
}

func internalSortCompPointersByAvgPlacement(comps []*data.Comp) {
	for i := 1; i < len(comps); i++ {
		for j := i; j > 0 && comps[j].AvgPlacement < comps[j-1].AvgPlacement; j-- {
			comps[j], comps[j-1] = comps[j-1], comps[j]
		}
	}
}

func internalNormalizeContext(ctx contracts.QueryNLURequest, store *data.Store) contracts.QueryNLURequest {
	result := ctx

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

	if len(ctx.Traits) > 0 {
		normalizedTraits := make([]string, 0, len(ctx.Traits))
		for _, trait := range ctx.Traits {
			normalizedTraits = append(normalizedTraits, store.IDToCN(trait))
		}
		result.Traits = normalizedTraits
	}

	return result
}

func toCompSummary(comp data.Comp, store *data.Store) contracts.CompSummary {
	return contracts.CompSummary{
		ClusterID:    comp.ClusterID,
		Name:         localizeCompName(comp.Name, store),
		Tier:         comp.Tier,
		AvgPlacement: comp.AvgPlacement,
		Top4Rate:     comp.Top4Rate,
		WinRate:      comp.WinRate,
		Count:        comp.Count,
		Units:        localizeStrings(comp.Units, store),
		Traits:       localizeStrings(comp.Traits, store),
		Stars:        localizeStrings(comp.Stars, store),
		Levelling:    comp.Levelling,
		Difficulty:   comp.Difficulty,
		BestBuild:    toBuildInfo(comp.BestBuild, store),
		AllBuilds:    toBuildInfos(comp.AllBuilds, store),
	}
}

func ptrCompSummary(comp data.Comp, store *data.Store) *contracts.CompSummary {
	summary := toCompSummary(comp, store)
	return &summary
}

func toBuildInfo(build data.BuildInfo, store *data.Store) contracts.BuildInfo {
	return contracts.BuildInfo{
		Carry:          store.IDToCN(build.Carry),
		Items:          localizeStrings(build.Items, store),
		PriorityScores: localizePriorityScores(build.PriorityScores, store),
		AvgPlacement:   build.AvgPlacement,
		PlaceChange:    build.PlaceChange,
		Score:          build.Score,
	}
}

func toBuildInfos(builds []data.BuildInfo, store *data.Store) []contracts.BuildInfo {
	if len(builds) == 0 {
		return nil
	}

	out := make([]contracts.BuildInfo, 0, len(builds))
	for _, build := range builds {
		out = append(out, toBuildInfo(build, store))
	}
	return out
}

func localizeCompName(name string, store *data.Store) string {
	if store == nil || name == "" {
		return name
	}
	parts := strings.Split(name, ",")
	out := make([]string, 0, len(parts))
	changed := false
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		localized := store.IDToCN(trimmed)
		if localized != trimmed {
			changed = true
		}
		out = append(out, localized)
	}
	if !changed {
		return name
	}
	return strings.Join(out, "、")
}

func localizeStrings(values []string, store *data.Store) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, store.IDToCN(value))
	}
	return out
}

func localizePriorityScores(in map[string]int, store *data.Store) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for key, value := range in {
		out[store.IDToCN(key)] = value
	}
	return out
}

func clonePriorityScores(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
