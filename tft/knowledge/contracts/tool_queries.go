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
