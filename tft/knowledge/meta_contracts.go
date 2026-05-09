package knowledge

import (
	"strconv"
	"strings"

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
		Metadata:     metadataFromMetaComp(in),
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
		Plan:         compPlanFromMetaComp(in),
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

func metadataFromMetaComp(in *models.MetaComp) *contracts.KnowledgeMetadata {
	if in == nil {
		return nil
	}
	if in.Metadata != nil {
		return &contracts.KnowledgeMetadata{
			Version:     in.Metadata.Version,
			Source:      in.Metadata.Source,
			UpdatedAt:   in.Metadata.UpdatedAt,
			SampleCount: in.Metadata.SampleCount,
		}
	}

	metadata := &contracts.KnowledgeMetadata{
		Version:     in.TFTSet,
		Source:      "MetaTFT",
		UpdatedAt:   latestTrendDay(in.Trends),
		SampleCount: in.Count,
	}
	if metadata.Version == "" && metadata.Source == "" && metadata.UpdatedAt == "" && metadata.SampleCount == 0 {
		return nil
	}
	return metadata
}

func compPlanFromMetaComp(in *models.MetaComp) *contracts.CompPlan {
	if in == nil {
		return nil
	}
	if in.Plan != nil {
		return toContractCompPlan(in.Plan, in)
	}

	final := buildFinalSnapshot(in)
	if len(final.Units) == 0 && len(final.Traits) == 0 {
		return nil
	}
	return &contracts.CompPlan{
		ClusterID: in.ClusterID,
		Name:      metaCompDisplayName(in),
		Tier:      in.Tier,
		Final:     final,
	}
}

func toContractCompPlan(plan *models.CompPlan, fallback *models.MetaComp) *contracts.CompPlan {
	if plan == nil {
		return nil
	}
	out := &contracts.CompPlan{
		ClusterID: plan.ClusterID,
		Name:      plan.Name,
		Tier:      plan.Tier,
		Final:     toContractBoardSnapshot(plan.Final),
		Early:     toContractBoardSnapshotPtr(plan.Early),
		Middle:    toContractBoardSnapshotPtr(plan.Middle),
	}
	if fallback != nil {
		if out.ClusterID == "" {
			out.ClusterID = fallback.ClusterID
		}
		if out.Name == "" {
			out.Name = metaCompDisplayName(fallback)
		}
		if out.Tier == "" {
			out.Tier = fallback.Tier
		}
		if len(out.Final.Units) == 0 && len(out.Final.Traits) == 0 {
			out.Final = buildFinalSnapshot(fallback)
		}
	}
	return out
}

func toContractBoardSnapshotPtr(snapshot *models.BoardSnapshot) *contracts.BoardSnapshot {
	if snapshot == nil {
		return nil
	}
	out := toContractBoardSnapshot(*snapshot)
	return &out
}

func toContractBoardSnapshot(snapshot models.BoardSnapshot) contracts.BoardSnapshot {
	return contracts.BoardSnapshot{
		Level:  snapshot.Level,
		Units:  toContractBoardUnits(snapshot.Units),
		Traits: toContractTraitMarkers(snapshot.Traits),
	}
}

func toContractBoardUnits(in []models.BoardUnit) []contracts.BoardUnit {
	if len(in) == 0 {
		return nil
	}
	out := make([]contracts.BoardUnit, 0, len(in))
	for _, unit := range in {
		out = append(out, contracts.BoardUnit{
			Name:     unit.Name,
			Items:    cloneStrings(unit.Items),
			IsCore:   unit.IsCore,
			Priority: unit.Priority,
		})
	}
	return out
}

func toContractTraitMarkers(in []models.TraitMarker) []contracts.TraitMarker {
	if len(in) == 0 {
		return nil
	}
	out := make([]contracts.TraitMarker, 0, len(in))
	for _, trait := range in {
		out = append(out, contracts.TraitMarker{
			Name:  trait.Name,
			Count: trait.Count,
		})
	}
	return out
}

func buildFinalSnapshot(in *models.MetaComp) contracts.BoardSnapshot {
	if in == nil {
		return contracts.BoardSnapshot{}
	}

	buildsByUnit := make(map[string]models.CompBuild, len(in.Builds))
	for _, build := range in.Builds {
		name := strings.TrimSpace(build.Unit)
		if name == "" {
			continue
		}
		buildsByUnit[name] = build
	}

	units := make([]contracts.BoardUnit, 0, len(in.Units))
	for index, unitName := range in.Units {
		name := strings.TrimSpace(unitName)
		if name == "" {
			continue
		}
		unit := contracts.BoardUnit{Name: name}
		if build, ok := buildsByUnit[name]; ok {
			unit.Items = cloneStrings(build.Items)
			unit.IsCore = true
			unit.Priority = index + 1
		}
		units = append(units, unit)
	}

	return contracts.BoardSnapshot{
		Level:  finalLevelFromLevelling(in.Levelling),
		Units:  units,
		Traits: traitMarkersFromNames(in.Traits),
	}
}

func traitMarkersFromNames(in []string) []contracts.TraitMarker {
	if len(in) == 0 {
		return nil
	}
	out := make([]contracts.TraitMarker, 0, len(in))
	for _, raw := range in {
		marker := parseTraitMarker(raw)
		if marker.Name != "" {
			out = append(out, marker)
		}
	}
	return out
}

func parseTraitMarker(raw string) contracts.TraitMarker {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return contracts.TraitMarker{}
	}
	marker := contracts.TraitMarker{Name: trimmed}
	openIndex := strings.LastIndex(trimmed, "(")
	closeIndex := strings.LastIndex(trimmed, ")")
	if openIndex < 0 || closeIndex <= openIndex {
		openIndex = strings.LastIndex(trimmed, "（")
		closeIndex = strings.LastIndex(trimmed, "）")
	}
	if openIndex >= 0 && closeIndex > openIndex {
		marker.Name = strings.TrimSpace(trimmed[:openIndex])
		countText := strings.TrimSpace(trimmed[openIndex+1 : closeIndex])
		if count, err := strconv.Atoi(countText); err == nil {
			marker.Count = count
		}
	}
	return marker
}

func finalLevelFromLevelling(levelling string) string {
	lower := strings.ToLower(strings.TrimSpace(levelling))
	switch {
	case strings.Contains(lower, "9"):
		return "9"
	case strings.Contains(lower, "8"):
		return "8"
	case strings.Contains(lower, "7"):
		return "7"
	default:
		return ""
	}
}

func latestTrendDay(trends []models.Trend) string {
	if len(trends) == 0 {
		return ""
	}
	latest := ""
	for _, trend := range trends {
		if strings.TrimSpace(trend.Day) > latest {
			latest = strings.TrimSpace(trend.Day)
		}
	}
	return latest
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
