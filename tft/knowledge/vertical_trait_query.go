package knowledge

import (
	"sort"
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

var carryItemNames = map[string]struct{}{
	"珠光护手":     {},
	"鬼索的狂暴之刃":  {},
	"无尽之刃":     {},
	"朔极之矛":     {},
	"纳什之牙":     {},
	"大天使之杖":    {},
	"灭世者的死亡之帽": {},
	"蓝霸符":      {},
	"巨人杀手":     {},
	"最后的轻语":    {},
	"死亡之刃":     {},
	"海克斯科技枪刃":  {},
	"正义之手":     {},
	"泰坦的坚决":    {},
	"饮血剑":      {},
	"红霸符":      {},
	"莫雷洛秘典":    {},
	"虚空之杖":     {},
	"夜之锋刃":     {},
	"水银":       {},
}

var tankItemNames = map[string]struct{}{
	"日炎斗篷":   {},
	"狂徒铠甲":   {},
	"棘刺背心":   {},
	"巨龙之爪":   {},
	"石像鬼石板甲": {},
	"坚定之心":   {},
	"薄暮法袍":   {},
	"圣盾使的誓约": {},
	"救赎":     {},
	"离子火花":   {},
	"振奋盔甲":   {},
	"适应性头盔":  {},
}

func (s *UnifiedStore) enrichVerticalAndTraitQueries(result *contracts.QueryNLUResponse, ctx contracts.QueryNLURequest) {
	if shouldBuildTraitInsights(ctx) {
		result.MatchedTraits = s.buildTraitInsights(ctx.Traits)
		if len(result.MatchedComps) == 0 {
			result.MatchedComps = compsFromTraitInsights(result.MatchedTraits, 3)
		}
	}

	if shouldBuildChampionInsights(ctx) {
		result.MatchedChampions = s.buildChampionInsights(ctx)
		if len(result.MatchedComps) == 0 {
			result.MatchedComps = compsFromChampionInsights(result.MatchedChampions, 3)
		}
	}
}

func shouldBuildTraitInsights(ctx contracts.QueryNLURequest) bool {
	return ctx.Intent == "trait_query" || len(ctx.Traits) > 0
}

func shouldBuildChampionInsights(ctx contracts.QueryNLURequest) bool {
	return ctx.Intent == "vertical_query" || ctx.Intent == "champion_query" || len(ctx.Champions) > 0 || ctx.UnitCost != nil || strings.TrimSpace(ctx.RoleQuery) != ""
}

func (s *UnifiedStore) buildTraitInsights(traits []string) []contracts.TraitInsight {
	if s.knowledgeStore == nil {
		return nil
	}

	querySet := make(map[string]struct{}, len(traits))
	for _, trait := range traits {
		base := normalizeTraitName(trait)
		if base != "" {
			querySet[base] = struct{}{}
		}
	}

	type traitGroup struct {
		name        string
		activations map[string]struct{}
		units       map[string]struct{}
		comps       []contracts.CompSummary
	}
	groups := make(map[string]*traitGroup)

	for _, comp := range s.knowledgeStore.GetAllMetaComps() {
		for _, trait := range comp.Traits {
			base := normalizeTraitName(trait)
			if base == "" {
				continue
			}
			if len(querySet) > 0 && !traitMatchesAny(base, querySet) {
				continue
			}

			group := groups[base]
			if group == nil {
				group = &traitGroup{
					name:        base,
					activations: make(map[string]struct{}),
					units:       make(map[string]struct{}),
				}
				groups[base] = group
			}
			group.activations[strings.TrimSpace(trait)] = struct{}{}
			for _, unit := range comp.Units {
				if strings.TrimSpace(unit) != "" {
					group.units[unit] = struct{}{}
				}
			}
			group.comps = append(group.comps, enrichCompSummaryWithMeta(contracts.CompSummary{ClusterID: comp.ClusterID}, comp))
		}
	}

	insights := make([]contracts.TraitInsight, 0, len(groups))
	for _, group := range groups {
		sortCompSummariesByAvg(group.comps)
		insight := contracts.TraitInsight{
			Name:        group.name,
			Activations: sortedKeys(group.activations),
			Units:       limitStrings(sortedKeys(group.units), 10),
			BestComps:   limitCompSummaries(group.comps, 3),
		}
		insights = append(insights, insight)
	}

	sort.Slice(insights, func(i, j int) bool {
		return bestTraitAvg(insights[i]) < bestTraitAvg(insights[j])
	})
	if len(querySet) == 0 {
		insights = limitTraitInsights(insights, 3)
	}
	return insights
}

func (s *UnifiedStore) buildChampionInsights(ctx contracts.QueryNLURequest) []contracts.ChampionInsight {
	if s.knowledgeStore == nil {
		return nil
	}

	role := normalizeRoleQuery(ctx.RoleQuery)
	querySet := championQuerySet(ctx.Champions)
	insights := make([]contracts.ChampionInsight, 0)
	for _, champion := range s.knowledgeStore.GetAllMetaChampions() {
		if len(querySet) > 0 {
			if _, ok := querySet[normalizeChampionName(champion.Name)]; !ok {
				continue
			}
		}
		cost := 0
		if profile, ok := s.knowledgeStore.GetChampionProfileByName(champion.Name); ok {
			cost = profile.Cost
		}
		if ctx.UnitCost != nil {
			if cost == 0 || cost != *ctx.UnitCost {
				continue
			}
		}

		insight := s.buildChampionInsight(champion, cost)
		if role == "carry" && insight.CarryScore <= 0 {
			continue
		}
		if role == "tank" && insight.TankScore <= 0 {
			continue
		}
		insights = append(insights, insight)
	}

	sort.Slice(insights, func(i, j int) bool {
		return championInsightLess(insights[i], insights[j], role)
	})
	return limitChampionInsights(insights, 5)
}

func championQuerySet(champions map[string]int8) map[string]struct{} {
	if len(champions) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(champions))
	for name := range champions {
		normalized := normalizeChampionName(name)
		if normalized != "" {
			result[normalized] = struct{}{}
		}
	}
	return result
}

func normalizeChampionName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (s *UnifiedStore) buildChampionInsight(champion *models.MetaChampion, cost int) contracts.ChampionInsight {
	insight := contracts.ChampionInsight{
		Name:             champion.Name,
		Cost:             cost,
		BestAvgPlacement: bestChampionAvg(champion),
	}

	builds := make([]contracts.BuildInfo, 0, len(champion.Builds))
	for _, build := range champion.Builds {
		carryPoints, tankPoints := scoreItemsByRole(build.Items)
		weight := placementWeight(build.AvgPlacement)
		insight.CarryScore += float64(carryPoints) * weight
		insight.TankScore += float64(tankPoints) * weight
		builds = append(builds, contracts.BuildInfo{
			Carry:          champion.Name,
			Items:          cloneStrings(build.Items),
			PriorityScores: cloneIntMap(build.PriorityScore),
			AvgPlacement:   build.AvgPlacement,
		})
	}
	sort.Slice(builds, func(i, j int) bool {
		return builds[i].AvgPlacement < builds[j].AvgPlacement
	})
	insight.BestBuilds = limitBuildInfos(builds, 3)
	insight.BestComps = s.bestCompsForChampion(champion, 3)

	if insight.CarryScore >= insight.TankScore && insight.CarryScore > 0 {
		insight.Role = "主C"
		insight.Tags = append(insight.Tags, "能C")
	}
	if insight.TankScore > insight.CarryScore && insight.TankScore > 0 {
		insight.Role = "前排"
		insight.Tags = append(insight.Tags, "能抗")
	}
	if insight.CarryScore > 0 && insight.TankScore > 0 {
		insight.Tags = append(insight.Tags, "可摇摆")
	}

	return insight
}

func (s *UnifiedStore) bestCompsForChampion(champion *models.MetaChampion, limit int) []contracts.CompSummary {
	comps := make([]contracts.CompSummary, 0, len(champion.AppearInComps))
	seen := make(map[string]struct{})
	for _, app := range champion.AppearInComps {
		if app.ClusterID == "" {
			continue
		}
		if _, ok := seen[app.ClusterID]; ok {
			continue
		}
		seen[app.ClusterID] = struct{}{}
		if metaComp, ok := s.knowledgeStore.GetMetaCompByID(app.ClusterID); ok {
			comps = append(comps, enrichCompSummaryWithMeta(contracts.CompSummary{ClusterID: app.ClusterID}, metaComp))
			continue
		}
		comps = append(comps, contracts.CompSummary{
			ClusterID:    app.ClusterID,
			Name:         app.CompName,
			Tier:         app.Tier,
			AvgPlacement: app.AvgPlacement,
		})
	}
	sortCompSummariesByAvg(comps)
	return limitCompSummaries(comps, limit)
}

func traitMatchesAny(base string, querySet map[string]struct{}) bool {
	for query := range querySet {
		if base == query || strings.Contains(base, query) || strings.Contains(query, base) {
			return true
		}
	}
	return false
}

func normalizeTraitName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, "("); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	if idx := strings.Index(trimmed, "（"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	return trimmed
}

func normalizeRoleQuery(role string) string {
	lower := strings.ToLower(strings.TrimSpace(role))
	switch lower {
	case "carry", "c", "主c", "输出", "能c":
		return "carry"
	case "tank", "frontline", "前排", "坦克", "肉", "能抗":
		return "tank"
	case "work", "worker", "打工", "过渡", "前期", "二阶段":
		return "work"
	case "all", "综合":
		return "all"
	default:
		return lower
	}
}

func scoreItemsByRole(items []string) (carry int, tank int) {
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if _, ok := carryItemNames[trimmed]; ok {
			carry++
		}
		if _, ok := tankItemNames[trimmed]; ok {
			tank++
		}
	}
	return carry, tank
}

func placementWeight(avg float64) float64 {
	if avg <= 0 {
		return 1
	}
	return 1 + (5.5-avg)/5
}

func bestChampionAvg(champion *models.MetaChampion) float64 {
	best := 0.0
	for _, app := range champion.AppearInComps {
		best = minPositive(best, app.AvgPlacement)
	}
	for _, build := range champion.Builds {
		best = minPositive(best, build.AvgPlacement)
	}
	return best
}

func minPositive(current float64, candidate float64) float64 {
	if candidate <= 0 {
		return current
	}
	if current <= 0 || candidate < current {
		return candidate
	}
	return current
}

func championInsightLess(a contracts.ChampionInsight, b contracts.ChampionInsight, role string) bool {
	switch role {
	case "carry":
		if a.CarryScore != b.CarryScore {
			return a.CarryScore > b.CarryScore
		}
	case "tank":
		if a.TankScore != b.TankScore {
			return a.TankScore > b.TankScore
		}
	default:
		aScore := a.CarryScore + a.TankScore
		bScore := b.CarryScore + b.TankScore
		if aScore != bScore {
			return aScore > bScore
		}
	}
	if a.BestAvgPlacement != b.BestAvgPlacement {
		return a.BestAvgPlacement < b.BestAvgPlacement
	}
	return a.Name < b.Name
}

func sortCompSummariesByAvg(comps []contracts.CompSummary) {
	sort.Slice(comps, func(i, j int) bool {
		if comps[i].AvgPlacement != comps[j].AvgPlacement {
			return comps[i].AvgPlacement < comps[j].AvgPlacement
		}
		return comps[i].Name < comps[j].Name
	})
}

func bestTraitAvg(insight contracts.TraitInsight) float64 {
	if len(insight.BestComps) == 0 {
		return 99
	}
	return insight.BestComps[0].AvgPlacement
}

func compsFromTraitInsights(insights []contracts.TraitInsight, limit int) []contracts.CompSummary {
	var comps []contracts.CompSummary
	seen := make(map[string]struct{})
	for _, insight := range insights {
		for _, comp := range insight.BestComps {
			if comp.ClusterID == "" {
				continue
			}
			if _, ok := seen[comp.ClusterID]; ok {
				continue
			}
			seen[comp.ClusterID] = struct{}{}
			comps = append(comps, comp)
			if len(comps) >= limit {
				return comps
			}
		}
	}
	return comps
}

func compsFromChampionInsights(insights []contracts.ChampionInsight, limit int) []contracts.CompSummary {
	var comps []contracts.CompSummary
	seen := make(map[string]struct{})
	for _, insight := range insights {
		for _, comp := range insight.BestComps {
			if comp.ClusterID == "" {
				continue
			}
			if _, ok := seen[comp.ClusterID]; ok {
				continue
			}
			seen[comp.ClusterID] = struct{}{}
			comps = append(comps, comp)
			if len(comps) >= limit {
				return comps
			}
		}
	}
	return comps
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func limitCompSummaries(values []contracts.CompSummary, limit int) []contracts.CompSummary {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func limitBuildInfos(values []contracts.BuildInfo, limit int) []contracts.BuildInfo {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func limitChampionInsights(values []contracts.ChampionInsight, limit int) []contracts.ChampionInsight {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func limitTraitInsights(values []contracts.TraitInsight, limit int) []contracts.TraitInsight {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}
