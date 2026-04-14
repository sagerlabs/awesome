package contracts

// QueryNLURequest 定义 agent 与 knowledge 之间共享的查询请求结构。
// 它描述的是边界协议，而不是某个包内部的上下文模型。
type QueryNLURequest struct {
	Intent         string           `json:"intent"`
	Champions      map[string]int8  `json:"champions"`
	Items          []string         `json:"items"`
	Traits         []string         `json:"traits"`
	Augments       []string         `json:"augments"`
	ExplicitLineup *string          `json:"explicit_lineup"`
	InferredLineup string           `json:"inferred_lineup"`
	Playstyle      string           `json:"playstyle"`
	GameStage      *string          `json:"game_stage"`
	Gold           *int             `json:"gold"`
	Level          *int             `json:"level"`
	HP             *int             `json:"hp"`
}

// QueryNLUResponse 是 knowledge 对外返回的标准查询结果。
type QueryNLUResponse struct {
	UserInput    string            `json:"user_input"`
	Ctx          QueryNLURequest   `json:"ctx"`
	MatchedComps []CompSummary     `json:"matched_comps"`
	MatchedItems []MatchedItemInfo `json:"matched_items"`
}

// MatchedItemInfo 表示某件装备对应的推荐阵容信息。
type MatchedItemInfo struct {
	ItemID    string            `json:"item_id"`
	ItemName  string            `json:"item_name"`
	CompInfos []ItemFitCompInfo `json:"comp_infos"`
}

// ItemFitCompInfo 描述某件装备适配到的阵容及 carry 信息。
type ItemFitCompInfo struct {
	ClusterID     string  `json:"cluster_id"`
	CompName      string  `json:"comp_name"`
	CompTier      string  `json:"comp_tier"`
	CompAvg       float64 `json:"comp_avg"`
	Carry         string  `json:"carry"`
	CarryName     string  `json:"carry_name"`
	PriorityScore int     `json:"priority_score"`
}

// CompSummary 是跨 knowledge 边界暴露给 agent 的阵容摘要。
// 这里保留 QueryNLU 当前真正会消费的字段，避免把 data.Comp 直接泄漏到 contract 层。
type CompSummary struct {
	ClusterID    string      `json:"cluster_id"`
	Name         string      `json:"name"`
	Tier         string      `json:"tier"`
	AvgPlacement float64     `json:"avg_placement"`
	Top4Rate     float64     `json:"top4_rate"`
	WinRate      float64     `json:"win_rate"`
	Count        int         `json:"count"`
	Units        []string    `json:"units"`
	Traits       []string    `json:"traits"`
	Stars        []string    `json:"stars"`
	Levelling    string      `json:"levelling"`
	Difficulty   float64     `json:"difficulty"`
	BestBuild    BuildInfo   `json:"best_build"`
	AllBuilds    []BuildInfo `json:"all_builds,omitempty"`
}

// BuildInfo 是阵容内某个核心英雄的装备方案。
type BuildInfo struct {
	Carry          string         `json:"carry"`
	Items          []string       `json:"items"`
	PriorityScores map[string]int `json:"priority_scores"`
	AvgPlacement   float64        `json:"avg_placement"`
	PlaceChange    float64        `json:"place_change"`
	Score          float64        `json:"score,omitempty"`
}
