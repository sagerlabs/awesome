package contracts

// QueryNLURequest 定义 agent 与 knowledge 之间共享的查询请求结构。
// 它描述的是边界协议，而不是某个包内部的上下文模型。
type QueryNLURequest struct {
	Intent         string          `json:"intent"`
	Champions      map[string]int8 `json:"champions"`
	Items          []string        `json:"items"`
	Traits         []string        `json:"traits"`
	Augments       []string        `json:"augments"`
	ExplicitLineup *string         `json:"explicit_lineup"`
	InferredLineup string          `json:"inferred_lineup"`
	Playstyle      string          `json:"playstyle"`
	UnitCost       *int            `json:"unit_cost"`
	RoleQuery      string          `json:"role_query"`
	GameStage      *string         `json:"game_stage"`
	Gold           *int            `json:"gold"`
	Level          *int            `json:"level"`
	HP             *int            `json:"hp"`
	// PatchHint 用户显式提及的补丁版本（如问"17.7 改了什么"时填 "17.7"）。
	// DATA-03：非空时公告检索进入历史模式；为空一律只看当前补丁，
	// 旧公告不再混入当前回答（NLU 抽取端接入由 CORE-02 完成，
	// 字段缺席时零值走 current 模式，行为向后兼容）。
	PatchHint string `json:"patch_hint,omitempty"`
}

// QueryNLUResponse 是 knowledge 对外返回的标准查询结果。
type QueryNLUResponse struct {
	UserInput        string             `json:"user_input"`
	Ctx              QueryNLURequest    `json:"ctx"`
	Metadata         *KnowledgeMetadata `json:"metadata,omitempty"`
	Feedback         *AdviceFeedback    `json:"feedback,omitempty"`
	NormalizedTerms  []NormalizedTerm   `json:"normalized_terms,omitempty"`
	MatchedComps     []CompSummary      `json:"matched_comps"`
	MatchedItems     []MatchedItemInfo  `json:"matched_items"`
	MatchedChampions []ChampionInsight  `json:"matched_champions,omitempty"`
	MatchedTraits    []TraitInsight     `json:"matched_traits,omitempty"`
	PatchNotes       []PatchNoteInsight `json:"patch_notes,omitempty"`
}

// AdviceFeedback describes whether the previous advice appears accepted, rejected, or context-starved.
type AdviceFeedback struct {
	Type              string `json:"type"`
	Reason            string `json:"reason,omitempty"`
	PreviousUserInput string `json:"previous_user_input,omitempty"`
	LastAdviceSummary string `json:"last_advice_summary,omitempty"`
}

// NormalizedTerm records how player slang was mapped before querying knowledge.
type NormalizedTerm struct {
	Type       string `json:"type"`
	Raw        string `json:"raw"`
	Normalized string `json:"normalized"`
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
	ClusterID    string             `json:"cluster_id"`
	Metadata     *KnowledgeMetadata `json:"metadata,omitempty"`
	Name         string             `json:"name"`
	Tier         string             `json:"tier"`
	AvgPlacement float64            `json:"avg_placement"`
	Top4Rate     float64            `json:"top4_rate"`
	WinRate      float64            `json:"win_rate"`
	Count        int                `json:"count"`
	Units        []string           `json:"units"`
	Traits       []string           `json:"traits"`
	Stars        []string           `json:"stars"`
	Levelling    string             `json:"levelling"`
	Difficulty   float64            `json:"difficulty"`
	BestBuild    BuildInfo          `json:"best_build"`
	AllBuilds    []BuildInfo        `json:"all_builds,omitempty"`
	Plan         *CompPlan          `json:"plan,omitempty"`
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

// ChampionInsight 是垂直英雄查询返回给 agent 的事实摘要。
type ChampionInsight struct {
	Name             string        `json:"name"`
	Cost             int           `json:"cost,omitempty"`
	Role             string        `json:"role,omitempty"`
	Tags             []string      `json:"tags,omitempty"`
	BestAvgPlacement float64       `json:"best_avg_placement,omitempty"`
	CarryScore       float64       `json:"carry_score,omitempty"`
	TankScore        float64       `json:"tank_score,omitempty"`
	WorkScore        float64       `json:"work_score,omitempty"`
	WorkReason       string        `json:"work_reason,omitempty"`
	BestComps        []CompSummary `json:"best_comps,omitempty"`
	BestBuilds       []BuildInfo   `json:"best_builds,omitempty"`
}

// TraitInsight 是羁绊查询返回给 agent 的事实摘要。
type TraitInsight struct {
	Name        string        `json:"name"`
	Activations []string      `json:"activations,omitempty"`
	Units       []string      `json:"units,omitempty"`
	BestComps   []CompSummary `json:"best_comps,omitempty"`
}

// PatchNoteInsight 是官方版本公告中与本次查询相关的环境信息。
type PatchNoteInsight struct {
	Patch        string   `json:"patch"`
	Title        string   `json:"title"`
	Source       string   `json:"source"`
	SourceURL    string   `json:"source_url"`
	PublishedAt  string   `json:"published_at"`
	SectionType  string   `json:"section_type"`
	SectionTitle string   `json:"section_title"`
	Summary      string   `json:"summary"`
	ImpactTags   []string `json:"impact_tags,omitempty"`
	Details      []string `json:"details,omitempty"`
	// IsHistorical 该条公告是否属于历史（非当前生效）补丁。
	// DATA-03：current 模式恒为 false；历史模式（PatchHint 指定旧版本）
	// 为 true，agent 应显式向用户说明这是历史版本内容。
	// 零值语义：false = 当前（未显式设置的旧调用方默认按当前处理）。
	IsHistorical bool `json:"is_historical,omitempty"`
}
