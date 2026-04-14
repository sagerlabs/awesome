package knowledge

import (
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

func toContractMetaComp(in *models.MetaComp) *contracts.MetaComp {
	if in == nil {
		return nil
	}

	return &contracts.MetaComp{
		ClusterID:    in.ClusterID,
		TFTSet:       in.TFTSet,
		Units:        cloneStrings(in.Units),
		Traits:       cloneStrings(in.Traits),
		Stars:        cloneStrings(in.Stars),
		NameString:   in.NameString,
		DisplayNames: toContractDisplayNames(in.DisplayNames),
		Count:        in.Count,
		AvgPlacement: in.AvgPlacement,
		Top4Rate:     in.Top4Rate,
		WinRate:      in.WinRate,
		Tier:         in.Tier,
		Builds:       toContractCompBuilds(in.Builds),
		BuildItems:   toContractBuildItems(in.BuildItems),
		Trends:       toContractTrends(in.Trends),
		Levelling:    in.Levelling,
		Difficulty:   in.Difficulty,
		Description:  in.Description,
		Limit:        cloneAnyMap(in.Limit),
	}
}

func toContractMetaComps(in []*models.MetaComp) []*contracts.MetaComp {
	out := make([]*contracts.MetaComp, 0, len(in))
	for _, item := range in {
		out = append(out, toContractMetaComp(item))
	}
	return out
}

func paginateMetaComps(in []*contracts.MetaComp, limit int, offset int) []*contracts.MetaComp {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(in) {
		return []*contracts.MetaComp{}
	}

	end := len(in)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return in[offset:end]
}

func toContractMetaChampion(in *models.MetaChampion) *contracts.MetaChampion {
	if in == nil {
		return nil
	}

	return &contracts.MetaChampion{
		Name:          in.Name,
		AppearInComps: toContractCompAppearances(in.AppearInComps),
		Builds:        toContractChampionBuilds(in.Builds),
		Description:   in.Description,
		Limit:         cloneAnyMap(in.Limit),
	}
}

func toContractMetaChampions(in []*models.MetaChampion) []*contracts.MetaChampion {
	out := make([]*contracts.MetaChampion, 0, len(in))
	for _, item := range in {
		out = append(out, toContractMetaChampion(item))
	}
	return out
}

func toContractMetaItem(in *models.MetaItem) *contracts.MetaItem {
	if in == nil {
		return nil
	}

	return &contracts.MetaItem{
		Name:         in.Name,
		PriorityList: toContractItemPriorities(in.PriorityList),
		Description:  in.Description,
		Limit:        cloneAnyMap(in.Limit),
	}
}

func toContractMetaItems(in []*models.MetaItem) []*contracts.MetaItem {
	out := make([]*contracts.MetaItem, 0, len(in))
	for _, item := range in {
		out = append(out, toContractMetaItem(item))
	}
	return out
}

func toContractDisplayNames(in []models.DisplayName) []contracts.DisplayName {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.DisplayName, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.DisplayName{
			Name:  item.Name,
			Type:  item.Type,
			Score: item.Score,
		})
	}
	return out
}

func toContractCompBuilds(in []models.CompBuild) []contracts.CompBuild {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.CompBuild, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.CompBuild{
			Unit:           item.Unit,
			Items:          cloneStrings(item.Items),
			AvgPlacement:   item.AvgPlacement,
			Count:          item.Count,
			Score:          item.Score,
			PlaceChange:    item.PlaceChange,
			PriorityScores: cloneIntMap(item.PriorityScores),
			Description:    item.Description,
			Limit:          cloneAnyMap(item.Limit),
		})
	}
	return out
}

func toContractBuildItems(in map[string]models.BuildItem) map[string]contracts.BuildItem {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]contracts.BuildItem, len(in))
	for key, item := range in {
		out[key] = contracts.BuildItem{
			ItemNames: item.ItemNames,
			Count:     item.Count,
			Avg:       item.Avg,
			Pcnt:      item.Pcnt,
		}
	}
	return out
}

func toContractTrends(in []models.Trend) []contracts.Trend {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.Trend, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.Trend{
			Day:      item.Day,
			Count:    item.Count,
			Avg:      item.Avg,
			PickRate: item.PickRate,
		})
	}
	return out
}

func toContractCompAppearances(in []models.CompAppearance) []contracts.CompAppearance {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.CompAppearance, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.CompAppearance{
			ClusterID:    item.ClusterID,
			CompName:     item.CompName,
			Tier:         item.Tier,
			AvgPlacement: item.AvgPlacement,
		})
	}
	return out
}

func toContractChampionBuilds(in []models.ChampionBuild) []contracts.ChampionBuild {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.ChampionBuild, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.ChampionBuild{
			ClusterID:     item.ClusterID,
			CompName:      item.CompName,
			Items:         cloneStrings(item.Items),
			AvgPlacement:  item.AvgPlacement,
			Count:         item.Count,
			PriorityScore: cloneIntMap(item.PriorityScore),
			Description:   item.Description,
			Limit:         cloneAnyMap(item.Limit),
		})
	}
	return out
}

func toContractItemPriorities(in []models.ItemPriority) []contracts.ItemPriority {
	if len(in) == 0 {
		return nil
	}

	out := make([]contracts.ItemPriority, 0, len(in))
	for _, item := range in {
		out = append(out, contracts.ItemPriority{
			ClusterID:     item.ClusterID,
			CompName:      item.CompName,
			CompTier:      item.CompTier,
			CompAvg:       item.CompAvg,
			Carry:         item.Carry,
			PriorityScore: item.PriorityScore,
		})
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneAnyMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
