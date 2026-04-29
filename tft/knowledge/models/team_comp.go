package models

// TeamComp 阵容策略模型
type TeamComp struct {
	ID          string        `json:"id" yaml:"id"`
	Name        string        `json:"name" yaml:"name"`
	Version     string        `json:"version" yaml:"version"`
	Tier        string        `json:"tier" yaml:"tier"` // "S" | "A" | "B" | "C"
	Champions   []CompChampion `json:"champions" yaml:"champions"`
	Traits      []CompTrait   `json:"traits" yaml:"traits"`
	Strategy    Strategy      `json:"strategy" yaml:"strategy"`
	Positioning string        `json:"positioning" yaml:"positioning"`
	Meta        CompMeta      `json:"meta,omitempty" yaml:"meta,omitempty"`
}

// CompChampion 阵容中的英雄
type CompChampion struct {
	Name     string   `json:"name" yaml:"name"`
	Role     string   `json:"role" yaml:"role"` // "main_carry" | "sub_carry" | "tank" | "support"
	Items    []string `json:"items" yaml:"items"`
	Priority int      `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// CompTrait 阵容中的羁绊
type CompTrait struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

// Strategy 阵容策略
type Strategy struct {
	EarlyGame string `json:"early_game" yaml:"early_game"`
	MidGame   string `json:"mid_game" yaml:"mid_game"`
	LateGame  string `json:"late_game" yaml:"late_game"`
	Economy   string `json:"economy,omitempty" yaml:"economy,omitempty"`
}

// CompMeta 阵容元数据
type CompMeta struct {
	AvgPlacement float64 `json:"avg_placement,omitempty" yaml:"avg_placement,omitempty"`
	Top4Rate     float64 `json:"top4_rate,omitempty" yaml:"top4_rate,omitempty"`
	WinRate      float64 `json:"win_rate,omitempty" yaml:"win_rate,omitempty"`
	PlayRate     float64 `json:"play_rate,omitempty" yaml:"play_rate,omitempty"`
	Difficulty   string  `json:"difficulty,omitempty" yaml:"difficulty,omitempty"`
}
