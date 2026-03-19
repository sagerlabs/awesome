package models

// Trait 羁绊数据模型
type Trait struct {
	ID          string         `json:"id" yaml:"id"`
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Breaks      []BreakPoint   `json:"breaks" yaml:"breaks"`
	BestHeroes  []string       `json:"best_heroes,omitempty" yaml:"best_heroes,omitempty"`
	Analysis    TraitAnalysis  `json:"analysis,omitempty" yaml:"analysis,omitempty"`
	Version     string         `json:"version" yaml:"version"`
}

// BreakPoint 羁绊断点
type BreakPoint struct {
	Count   int    `json:"count" yaml:"count"`
	Effect  string `json:"effect" yaml:"effect"`
	IsMajor bool   `json:"is_major,omitempty" yaml:"is_major,omitempty"`
}

// TraitAnalysis 羁绊分析
type TraitAnalysis struct {
	Strengths  []string `json:"strengths,omitempty" yaml:"strengths,omitempty"`
	Weaknesses []string `json:"weaknesses,omitempty" yaml:"weaknesses,omitempty"`
	Synergies  []string `json:"synergies,omitempty" yaml:"synergies,omitempty"`
}
