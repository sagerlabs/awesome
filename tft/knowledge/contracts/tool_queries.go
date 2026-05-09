package contracts

type GetCompByIDRequest struct {
	ClusterID string `json:"cluster_id"`
}

type GetCompByIDResponse struct {
	Comp *CompSummary `json:"comp,omitempty"`
}

type GetMetaCompByIDRequest struct {
	ClusterID string `json:"cluster_id"`
}

type GetMetaCompByIDResponse struct {
	Comp *MetaComp `json:"comp,omitempty"`
}

type GetMetaCompByNameRequest struct {
	Name string `json:"name"`
}

type GetMetaCompByNameResponse struct {
	Comp *MetaComp `json:"comp,omitempty"`
}

type SearchMetaCompsRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type SearchMetaCompsResponse struct {
	Comps []*MetaComp `json:"comps"`
}

type GetAllMetaCompsRequest struct{}

type GetAllMetaCompsResponse struct {
	Comps []*MetaComp `json:"comps"`
}

type ListMetaCompsRequest struct {
	Tier                string   `json:"tier,omitempty"`
	Tiers               []string `json:"tiers,omitempty"`
	Limit               int      `json:"limit,omitempty"`
	Offset              int      `json:"offset,omitempty"`
	DesiredOutputFields []string `json:"desired_output_fields,omitempty"`
}

type ListMetaCompsResponse struct {
	Metadata *KnowledgeMetadata `json:"metadata,omitempty"`
	Comps    []map[string]any   `json:"comps"`
}

type GetCompPlanRequest struct {
	ClusterID           string   `json:"cluster_id,omitempty"`
	Name                string   `json:"name,omitempty"`
	DesiredOutputFields []string `json:"desired_output_fields,omitempty"`
}

type GetCompPlanResponse struct {
	Metadata *KnowledgeMetadata `json:"metadata,omitempty"`
	Plan     *CompPlan          `json:"plan,omitempty"`
}

type GetChampionBuildsRequest struct {
	Name                string   `json:"name"`
	Limit               int      `json:"limit,omitempty"`
	DesiredOutputFields []string `json:"desired_output_fields,omitempty"`
}

type GetChampionBuildsResponse struct {
	Metadata *KnowledgeMetadata `json:"metadata,omitempty"`
	Champion map[string]any     `json:"champion,omitempty"`
}

type GetItemFitsRequest struct {
	Name                string   `json:"name"`
	Limit               int      `json:"limit,omitempty"`
	DesiredOutputFields []string `json:"desired_output_fields,omitempty"`
}

type GetItemFitsResponse struct {
	Metadata *KnowledgeMetadata `json:"metadata,omitempty"`
	Item     map[string]any     `json:"item,omitempty"`
}

type GetTraitInsightRequest struct {
	Name                string   `json:"name"`
	Limit               int      `json:"limit,omitempty"`
	DesiredOutputFields []string `json:"desired_output_fields,omitempty"`
}

type GetTraitInsightResponse struct {
	Metadata *KnowledgeMetadata `json:"metadata,omitempty"`
	Trait    map[string]any     `json:"trait,omitempty"`
}

type GetMetaChampionByNameRequest struct {
	Name string `json:"name"`
}

type GetMetaChampionByNameResponse struct {
	Champion *MetaChampion `json:"champion,omitempty"`
}

type GetAllMetaChampionsRequest struct{}

type GetAllMetaChampionsResponse struct {
	Champions []*MetaChampion `json:"champions"`
}

type GetMetaItemByNameRequest struct {
	Name string `json:"name"`
}

type GetMetaItemByNameResponse struct {
	Item *MetaItem `json:"item,omitempty"`
}

type GetAllMetaItemsRequest struct{}

type GetAllMetaItemsResponse struct {
	Items []*MetaItem `json:"items"`
}
