package knowledge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

// ListMetaComps exposes a compact, field-selectable meta comp list for tool callers.
func (s *UnifiedStore) ListMetaComps(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.ListMetaCompsRequest
	if len(req) > 0 {
		if err := json.Unmarshal(req, &request); err != nil {
			return nil, fmt.Errorf("unmarshal list meta comps request: %w", err)
		}
	}
	if s.knowledgeStore == nil {
		return marshalResponse(contracts.ListMetaCompsResponse{Comps: []map[string]any{}})
	}

	comps := s.filteredSortedMetaComps(request.Tier, request.Tiers)
	comps = paginateModelMetaComps(comps, request.Limit, request.Offset)
	projected := make([]map[string]any, 0, len(comps))
	for _, comp := range comps {
		projected = append(projected, projectMetaComp(comp, request.DesiredOutputFields))
	}

	return marshalResponse(contracts.ListMetaCompsResponse{
		Metadata: s.buildKnowledgeMetadata(),
		Comps:    projected,
	})
}

// GetCompPlan returns the board plan for a comp. Current generated data can derive final boards,
// while future pipelines can fill early/middle snapshots directly.
func (s *UnifiedStore) GetCompPlan(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetCompPlanRequest
	if err := json.Unmarshal(req, &request); err != nil {
		return nil, fmt.Errorf("unmarshal get comp plan request: %w", err)
	}
	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	comp, ok := s.findMetaComp(request.ClusterID, request.Name)
	if !ok {
		return nil, fmt.Errorf("meta comp not found: cluster_id=%s name=%s", request.ClusterID, request.Name)
	}

	resp := contracts.GetCompPlanResponse{
		Metadata: metadataFromMetaComp(comp),
		Plan:     projectCompPlan(compPlanFromMetaComp(comp), request.DesiredOutputFields),
	}
	return marshalResponse(resp)
}

// GetChampionBuilds returns a compact champion view for vertical queries.
func (s *UnifiedStore) GetChampionBuilds(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetChampionBuildsRequest
	if err := json.Unmarshal(req, &request); err != nil {
		return nil, fmt.Errorf("unmarshal get champion builds request: %w", err)
	}
	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := s.resolveKnowledgeAlias("champion", request.Name)
	champion, ok := s.knowledgeStore.GetMetaChampionByName(name)
	if !ok {
		return nil, fmt.Errorf("meta champion not found: %s", request.Name)
	}

	return marshalResponse(contracts.GetChampionBuildsResponse{
		Metadata: s.buildKnowledgeMetadata(),
		Champion: projectChampion(
			toContractMetaChampion(limitChampionModel(champion, request.Limit)),
			request.DesiredOutputFields,
		),
	})
}

// GetItemFits returns item-to-comp fit rows with priority scores.
func (s *UnifiedStore) GetItemFits(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetItemFitsRequest
	if err := json.Unmarshal(req, &request); err != nil {
		return nil, fmt.Errorf("unmarshal get item fits request: %w", err)
	}
	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := s.resolveKnowledgeAlias("item", request.Name)
	item, ok := s.knowledgeStore.GetMetaItemByName(name)
	if !ok {
		return nil, fmt.Errorf("meta item not found: %s", request.Name)
	}

	return marshalResponse(contracts.GetItemFitsResponse{
		Metadata: s.buildKnowledgeMetadata(),
		Item: projectItem(
			toContractMetaItem(limitItemModel(item, request.Limit)),
			request.DesiredOutputFields,
		),
	})
}

// GetTraitInsight returns representative comps and units for a trait.
func (s *UnifiedStore) GetTraitInsight(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetTraitInsightRequest
	if err := json.Unmarshal(req, &request); err != nil {
		return nil, fmt.Errorf("unmarshal get trait insight request: %w", err)
	}
	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := s.resolveKnowledgeAlias("trait", request.Name)
	insights := s.buildTraitInsights([]string{name})
	if len(insights) == 0 {
		return nil, fmt.Errorf("trait insight not found: %s", request.Name)
	}
	trait := insights[0]
	if request.Limit > 0 && len(trait.BestComps) > request.Limit {
		trait.BestComps = trait.BestComps[:request.Limit]
	}

	return marshalResponse(contracts.GetTraitInsightResponse{
		Metadata: s.buildKnowledgeMetadata(),
		Trait:    projectTrait(trait, request.DesiredOutputFields),
	})
}

func (s *UnifiedStore) filteredSortedMetaComps(tier string, tiers []string) []*models.MetaComp {
	tierSet := make(map[string]struct{})
	addTier := func(value string) {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if normalized != "" {
			tierSet[normalized] = struct{}{}
		}
	}
	addTier(tier)
	for _, value := range tiers {
		addTier(value)
	}

	comps := make([]*models.MetaComp, 0)
	for _, comp := range s.knowledgeStore.GetAllMetaComps() {
		if len(tierSet) > 0 {
			if _, ok := tierSet[strings.ToUpper(strings.TrimSpace(comp.Tier))]; !ok {
				continue
			}
		}
		comps = append(comps, comp)
	}

	sort.Slice(comps, func(i, j int) bool {
		if comps[i].AvgPlacement != comps[j].AvgPlacement {
			return comps[i].AvgPlacement < comps[j].AvgPlacement
		}
		return metaCompDisplayName(comps[i]) < metaCompDisplayName(comps[j])
	})
	return comps
}

func (s *UnifiedStore) findMetaComp(clusterID string, name string) (*models.MetaComp, bool) {
	if strings.TrimSpace(clusterID) != "" {
		return s.knowledgeStore.GetMetaCompByID(strings.TrimSpace(clusterID))
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false
	}
	if comp, ok := s.knowledgeStore.GetMetaCompByName(name); ok {
		return comp, true
	}
	matches := s.knowledgeStore.SearchMetaComps(name)
	if len(matches) > 0 {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].AvgPlacement < matches[j].AvgPlacement
		})
		return matches[0], true
	}
	return nil, false
}

func (s *UnifiedStore) resolveKnowledgeAlias(kind string, raw string) string {
	raw = strings.TrimSpace(raw)
	if s.knowledgeStore == nil {
		return raw
	}
	if normalized, ok := s.knowledgeStore.ResolveAlias(kind, raw); ok {
		return normalized
	}
	return raw
}

func (s *UnifiedStore) buildKnowledgeMetadata() *contracts.KnowledgeMetadata {
	if s == nil || s.knowledgeStore == nil {
		return nil
	}

	metadata := &contracts.KnowledgeMetadata{Source: "MetaTFT"}
	for _, comp := range s.knowledgeStore.GetAllMetaComps() {
		if comp == nil {
			continue
		}
		metadata.SampleCount += comp.Count
		if metadata.Version == "" {
			metadata.Version = comp.TFTSet
		}
		if comp.Metadata != nil {
			if metadata.Version == "" {
				metadata.Version = comp.Metadata.Version
			}
			if comp.Metadata.Source != "" {
				metadata.Source = comp.Metadata.Source
			}
			if comp.Metadata.UpdatedAt > metadata.UpdatedAt {
				metadata.UpdatedAt = comp.Metadata.UpdatedAt
			}
		}
		if latest := latestTrendDay(comp.Trends); latest > metadata.UpdatedAt {
			metadata.UpdatedAt = latest
		}
	}
	if metadata.Version == "" && metadata.Source == "" && metadata.UpdatedAt == "" && metadata.SampleCount == 0 {
		return nil
	}
	return metadata
}

func paginateModelMetaComps(in []*models.MetaComp, limit int, offset int) []*models.MetaComp {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(in) {
		return []*models.MetaComp{}
	}
	end := len(in)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return in[offset:end]
}

func projectMetaComp(comp *models.MetaComp, requested []string) map[string]any {
	meta := toContractMetaComp(comp)
	fields := fieldSet(requested, []string{
		"cluster_id", "name", "tier", "avg_placement", "top4_rate", "win_rate",
		"count", "levelling", "best_build", "plan",
	})
	out := make(map[string]any)
	if fieldWanted(fields, "cluster_id") {
		out["cluster_id"] = meta.ClusterID
	}
	if fieldWanted(fields, "name") {
		out["name"] = metaCompDisplayName(comp)
	}
	if fieldWanted(fields, "tier") {
		out["tier"] = meta.Tier
	}
	if fieldWanted(fields, "avg_placement") {
		out["avg_placement"] = meta.AvgPlacement
	}
	if fieldWanted(fields, "top4_rate") {
		out["top4_rate"] = meta.Top4Rate
	}
	if fieldWanted(fields, "win_rate") {
		out["win_rate"] = meta.WinRate
	}
	if fieldWanted(fields, "count") {
		out["count"] = meta.Count
	}
	if fieldWanted(fields, "units") {
		out["units"] = meta.Units
	}
	if fieldWanted(fields, "traits") {
		out["traits"] = meta.Traits
	}
	if fieldWanted(fields, "stars") {
		out["stars"] = meta.Stars
	}
	if fieldWanted(fields, "levelling") {
		out["levelling"] = meta.Levelling
	}
	if fieldWanted(fields, "difficulty") {
		out["difficulty"] = meta.Difficulty
	}
	if fieldWanted(fields, "metadata") {
		out["metadata"] = meta.Metadata
	}
	if fieldWanted(fields, "best_build") && len(meta.Builds) > 0 {
		out["best_build"] = meta.Builds[0]
	}
	if fieldWanted(fields, "builds") {
		out["builds"] = meta.Builds
	}
	if fieldWanted(fields, "plan") {
		out["plan"] = meta.Plan
	}
	return out
}

func projectCompPlan(plan *contracts.CompPlan, requested []string) *contracts.CompPlan {
	if plan == nil || len(requested) == 0 {
		return plan
	}
	fields := fieldSet(requested, nil)
	out := &contracts.CompPlan{}
	if fieldWanted(fields, "cluster_id") {
		out.ClusterID = plan.ClusterID
	}
	if fieldWanted(fields, "name") {
		out.Name = plan.Name
	}
	if fieldWanted(fields, "tier") {
		out.Tier = plan.Tier
	}
	if fieldWanted(fields, "final") {
		out.Final = plan.Final
	}
	if fieldWanted(fields, "early") {
		out.Early = plan.Early
	}
	if fieldWanted(fields, "middle") {
		out.Middle = plan.Middle
	}
	return out
}

func projectChampion(champion *contracts.MetaChampion, requested []string) map[string]any {
	fields := fieldSet(requested, []string{"name", "appear_in_comps", "builds"})
	out := make(map[string]any)
	if champion == nil {
		return out
	}
	if fieldWanted(fields, "name") {
		out["name"] = champion.Name
	}
	if fieldWanted(fields, "appear_in_comps") {
		out["appear_in_comps"] = champion.AppearInComps
	}
	if fieldWanted(fields, "builds") {
		out["builds"] = champion.Builds
	}
	return out
}

func projectItem(item *contracts.MetaItem, requested []string) map[string]any {
	fields := fieldSet(requested, []string{"name", "priority_list"})
	out := make(map[string]any)
	if item == nil {
		return out
	}
	if fieldWanted(fields, "name") {
		out["name"] = item.Name
	}
	if fieldWanted(fields, "priority_list") {
		out["priority_list"] = item.PriorityList
	}
	return out
}

func projectTrait(trait contracts.TraitInsight, requested []string) map[string]any {
	fields := fieldSet(requested, []string{"name", "activations", "units", "best_comps"})
	out := make(map[string]any)
	if fieldWanted(fields, "name") {
		out["name"] = trait.Name
	}
	if fieldWanted(fields, "activations") {
		out["activations"] = trait.Activations
	}
	if fieldWanted(fields, "units") {
		out["units"] = trait.Units
	}
	if fieldWanted(fields, "best_comps") {
		out["best_comps"] = trait.BestComps
	}
	return out
}

func fieldSet(requested []string, defaults []string) map[string]struct{} {
	values := requested
	if len(values) == 0 {
		values = defaults
	}
	out := make(map[string]struct{}, len(values))
	for _, field := range values {
		field = strings.ToLower(strings.TrimSpace(field))
		if field != "" {
			out[field] = struct{}{}
		}
	}
	return out
}

func fieldWanted(fields map[string]struct{}, field string) bool {
	_, ok := fields[field]
	return ok
}

func limitChampionModel(in *models.MetaChampion, limit int) *models.MetaChampion {
	if in == nil || limit <= 0 {
		return in
	}
	out := *in
	if len(out.Builds) > limit {
		out.Builds = append([]models.ChampionBuild(nil), out.Builds[:limit]...)
	}
	if len(out.AppearInComps) > limit {
		out.AppearInComps = append([]models.CompAppearance(nil), out.AppearInComps[:limit]...)
	}
	return &out
}

func limitItemModel(in *models.MetaItem, limit int) *models.MetaItem {
	if in == nil || limit <= 0 {
		return in
	}
	out := *in
	if len(out.PriorityList) > limit {
		out.PriorityList = append([]models.ItemPriority(nil), out.PriorityList[:limit]...)
	}
	return &out
}

func marshalResponse(value any) (Response, error) {
	resp, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return Response(resp), nil
}
