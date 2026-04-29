package contracts

// MetaComp 是 knowledge 对外暴露的版本阵容协议。
type MetaComp struct {
	ClusterID    string                 `json:"cluster_id" yaml:"cluster_id"`
	TFTSet       string                 `json:"tft_set" yaml:"tft_set"`
	Units        []string               `json:"units" yaml:"units"`
	Traits       []string               `json:"traits" yaml:"traits"`
	Stars        []string               `json:"stars" yaml:"stars"`
	NameString   string                 `json:"name_string" yaml:"name_string"`
	DisplayNames []DisplayName          `json:"display_names" yaml:"display_names"`
	Count        int                    `json:"count" yaml:"count"`
	AvgPlacement float64                `json:"avg_placement" yaml:"avg_placement"`
	Top4Rate     float64                `json:"top4_rate" yaml:"top4_rate"`
	WinRate      float64                `json:"win_rate" yaml:"win_rate"`
	Tier         string                 `json:"tier" yaml:"tier"`
	Builds       []CompBuild            `json:"builds" yaml:"builds"`
	BuildItems   map[string]BuildItem   `json:"build_items" yaml:"build_items"`
	Trends       []Trend                `json:"trends" yaml:"trends"`
	Levelling    string                 `json:"levelling" yaml:"levelling"`
	Difficulty   float64                `json:"difficulty" yaml:"difficulty"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit        map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

type DisplayName struct {
	Name  string  `json:"name" yaml:"name"`
	Type  string  `json:"type" yaml:"type"`
	Score float64 `json:"score" yaml:"score"`
}

type CompBuild struct {
	Unit           string                 `json:"unit" yaml:"unit"`
	Items          []string               `json:"items" yaml:"items"`
	AvgPlacement   float64                `json:"avg_placement" yaml:"avg_placement"`
	Count          int                    `json:"count" yaml:"count"`
	Score          float64                `json:"score" yaml:"score"`
	PlaceChange    float64                `json:"place_change" yaml:"place_change"`
	PriorityScores map[string]int         `json:"priority_scores" yaml:"priority_scores"`
	Description    string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit          map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

type BuildItem struct {
	ItemNames string  `json:"itemNames" yaml:"itemNames"`
	Count     int     `json:"count" yaml:"count"`
	Avg       float64 `json:"avg" yaml:"avg"`
	Pcnt      float64 `json:"pcnt" yaml:"pcnt"`
}

type Trend struct {
	Day      string  `json:"day" yaml:"day"`
	Count    int     `json:"count" yaml:"count"`
	Avg      float64 `json:"avg" yaml:"avg"`
	PickRate float64 `json:"pick_rate" yaml:"pick_rate"`
}

// MetaChampion 是 knowledge 对外暴露的版本英雄协议。
type MetaChampion struct {
	Name          string                 `json:"name" yaml:"name"`
	AppearInComps []CompAppearance       `json:"appear_in_comps" yaml:"appear_in_comps"`
	Builds        []ChampionBuild        `json:"builds" yaml:"builds"`
	Description   string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit         map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

type CompAppearance struct {
	ClusterID    string  `json:"cluster_id" yaml:"cluster_id"`
	CompName     string  `json:"comp_name" yaml:"comp_name"`
	Tier         string  `json:"tier" yaml:"tier"`
	AvgPlacement float64 `json:"avg_placement" yaml:"avg_placement"`
}

type ChampionBuild struct {
	ClusterID     string                 `json:"cluster_id" yaml:"cluster_id"`
	CompName      string                 `json:"comp_name" yaml:"comp_name"`
	Items         []string               `json:"items" yaml:"items"`
	AvgPlacement  float64                `json:"avg_placement" yaml:"avg_placement"`
	Count         int                    `json:"count" yaml:"count"`
	PriorityScore map[string]int         `json:"priority_scores,omitempty" yaml:"priority_scores,omitempty"`
	Description   string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit         map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

// MetaItem 是 knowledge 对外暴露的版本装备协议。
type MetaItem struct {
	Name         string                 `json:"name" yaml:"name"`
	PriorityList []ItemPriority         `json:"priority_list" yaml:"priority_list"`
	Description  string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Limit        map[string]interface{} `json:"limit,omitempty" yaml:"limit,omitempty"`
}

type ItemPriority struct {
	ClusterID     string  `json:"cluster_id" yaml:"cluster_id"`
	CompName      string  `json:"comp_name" yaml:"comp_name"`
	CompTier      string  `json:"comp_tier" yaml:"comp_tier"`
	CompAvg       float64 `json:"comp_avg" yaml:"comp_avg"`
	Carry         string  `json:"carry" yaml:"carry"`
	PriorityScore int     `json:"priority_score" yaml:"priority_score"`
}
