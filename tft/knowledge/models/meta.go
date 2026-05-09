package models

// MetaComp Meta阵容数据（来自MetaTFT）
type MetaComp struct {
	ClusterID    string               `json:"cluster_id" yaml:"cluster_id"`
	TFTSet       string               `json:"tft_set" yaml:"tft_set"`
	Metadata     *KnowledgeMetadata   `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Units        []string             `json:"units" yaml:"units"`
	Traits       []string             `json:"traits" yaml:"traits"`
	Stars        []string             `json:"stars" yaml:"stars"`
	NameString   string               `json:"name_string" yaml:"name_string"`
	DisplayNames []DisplayName        `json:"display_names" yaml:"display_names"`
	Count        int                  `json:"count" yaml:"count"`
	AvgPlacement float64              `json:"avg_placement" yaml:"avg_placement"`
	Top4Rate     float64              `json:"top4_rate" yaml:"top4_rate"`
	WinRate      float64              `json:"win_rate" yaml:"win_rate"`
	Tier         string               `json:"tier" yaml:"tier"`
	Builds       []CompBuild          `json:"builds" yaml:"builds"`
	BuildItems   map[string]BuildItem `json:"build_items" yaml:"build_items"`
	Trends       []Trend              `json:"trends" yaml:"trends"`
	Levelling    string               `json:"levelling" yaml:"levelling"`
	Difficulty   float64              `json:"difficulty" yaml:"difficulty"`
	Plan         *CompPlan            `json:"plan,omitempty" yaml:"plan,omitempty"`

	// 预留字段
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit       map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// KnowledgeMetadata 记录本条知识的版本、来源和样本可信度。
type KnowledgeMetadata struct {
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Source      string `json:"source,omitempty" yaml:"source,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	SampleCount int    `json:"sample_count,omitempty" yaml:"sample_count,omitempty"`
}

// CompPlan 描述阵容从过渡到成型的棋盘计划。
type CompPlan struct {
	ClusterID string         `json:"cluster_id,omitempty" yaml:"cluster_id,omitempty"`
	Name      string         `json:"name,omitempty" yaml:"name,omitempty"`
	Tier      string         `json:"tier,omitempty" yaml:"tier,omitempty"`
	Final     BoardSnapshot  `json:"final" yaml:"final"`
	Early     *BoardSnapshot `json:"early,omitempty" yaml:"early,omitempty"`
	Middle    *BoardSnapshot `json:"middle,omitempty" yaml:"middle,omitempty"`
}

// BoardSnapshot 是某个阶段的棋盘快照。
type BoardSnapshot struct {
	Level  string        `json:"level,omitempty" yaml:"level,omitempty"`
	Units  []BoardUnit   `json:"units,omitempty" yaml:"units,omitempty"`
	Traits []TraitMarker `json:"traits,omitempty" yaml:"traits,omitempty"`
}

// BoardUnit 是棋盘上的单位及其装备。
type BoardUnit struct {
	Name     string   `json:"name" yaml:"name"`
	Items    []string `json:"items,omitempty" yaml:"items,omitempty"`
	IsCore   bool     `json:"is_core,omitempty" yaml:"is_core,omitempty"`
	Priority int      `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// TraitMarker 是棋盘快照里的羁绊档位。
type TraitMarker struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count,omitempty" yaml:"count,omitempty"`
}

// DisplayName 显示名称
type DisplayName struct {
	Name  string  `json:"name" yaml:"name"`
	Type  string  `json:"type" yaml:"type"`
	Score float64 `json:"score" yaml:"score"`
}

// CompBuild 阵容出装
type CompBuild struct {
	Unit           string         `json:"unit" yaml:"unit"`
	Items          []string       `json:"items" yaml:"items"`
	AvgPlacement   float64        `json:"avg_placement" yaml:"avg_placement"`
	Count          int            `json:"count" yaml:"count"`
	Score          float64        `json:"score" yaml:"score"`
	PlaceChange    float64        `json:"place_change" yaml:"place_change"`
	PriorityScores map[string]int `json:"priority_scores" yaml:"priority_scores"`

	// 预留字段
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit       map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// BuildItem 出装物品统计
type BuildItem struct {
	ItemNames string  `json:"itemNames" yaml:"itemNames"`
	Count     int     `json:"count" yaml:"count"`
	Avg       float64 `json:"avg" yaml:"avg"`
	Pcnt      float64 `json:"pcnt" yaml:"pcnt"`
}

// Trend 趋势数据
type Trend struct {
	Day      string  `json:"day" yaml:"day"`
	Count    int     `json:"count" yaml:"count"`
	Avg      float64 `json:"avg" yaml:"avg"`
	PickRate float64 `json:"pick_rate" yaml:"pick_rate"`
}

// MetaChampion Meta英雄数据
type MetaChampion struct {
	Name          string           `json:"name" yaml:"name"`
	AppearInComps []CompAppearance `json:"appear_in_comps" yaml:"appear_in_comps"`
	Builds        []ChampionBuild  `json:"builds" yaml:"builds"`

	// 预留字段
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit       map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// CompAppearance 英雄在阵容中的出现
type CompAppearance struct {
	ClusterID    string  `json:"cluster_id" yaml:"cluster_id"`
	CompName     string  `json:"comp_name" yaml:"comp_name"`
	Tier         string  `json:"tier" yaml:"tier"`
	AvgPlacement float64 `json:"avg_placement" yaml:"avg_placement"`
}

// ChampionBuild 英雄出装
type ChampionBuild struct {
	ClusterID     string         `json:"cluster_id" yaml:"cluster_id"`
	CompName      string         `json:"comp_name" yaml:"comp_name"`
	Items         []string       `json:"items" yaml:"items"`
	AvgPlacement  float64        `json:"avg_placement" yaml:"avg_placement"`
	Count         int            `json:"count" yaml:"count"`
	PriorityScore map[string]int `json:"priority_scores,omitempty" yaml:"priority_scores,omitempty"`

	// 预留字段
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit       map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// MetaItem Meta装备数据
type MetaItem struct {
	Name         string         `json:"name" yaml:"name"`
	PriorityList []ItemPriority `json:"priority_list" yaml:"priority_list"`

	// 预留字段
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit       map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// ItemPriority 装备优先级
type ItemPriority struct {
	ClusterID     string  `json:"cluster_id" yaml:"cluster_id"`
	CompName      string  `json:"comp_name" yaml:"comp_name"`
	CompTier      string  `json:"comp_tier" yaml:"comp_tier"`
	CompAvg       float64 `json:"comp_avg" yaml:"comp_avg"`
	Carry         string  `json:"carry" yaml:"carry"`
	PriorityScore int     `json:"priority_score" yaml:"priority_score"`
}
