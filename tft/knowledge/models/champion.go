package models

// Champion 英雄数据模型
type Champion struct {
	ID          string        `json:"id" yaml:"id"`
	Name        string        `json:"name" yaml:"name"`
	Cost        int           `json:"cost" yaml:"cost"`
	Traits      []string      `json:"traits" yaml:"traits"`
	Stats       ChampionStats `json:"stats" yaml:"stats"`
	Ability     Ability       `json:"ability" yaml:"ability"`
	Analysis    Analysis      `json:"analysis" yaml:"analysis"`
	Version     string        `json:"version" yaml:"version"`
}

// ChampionStats 英雄属性
type ChampionStats struct {
	Health       int     `json:"health" yaml:"health"`
	Mana         int     `json:"mana" yaml:"mana"`
	AttackDamage int     `json:"attack_damage" yaml:"attack_damage"`
	Armor        int     `json:"armor,omitempty" yaml:"armor,omitempty"`
	MagicResist  int     `json:"magic_resist,omitempty" yaml:"magic_resist,omitempty"`
	AttackSpeed  float64 `json:"attack_speed,omitempty" yaml:"attack_speed,omitempty"`
}

// Ability 英雄技能
type Ability struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	ManaCost    int    `json:"mana_cost,omitempty" yaml:"mana_cost,omitempty"`
	Damage      string `json:"damage,omitempty" yaml:"damage,omitempty"`
}

// Analysis 英雄分析
type Analysis struct {
	Strengths     []string `json:"strengths" yaml:"strengths"`
	Weaknesses    []string `json:"weaknesses" yaml:"weaknesses"`
	BestItems     []string `json:"best_items" yaml:"best_items"`
	Synergies     []string `json:"synergies" yaml:"synergies"`
	CounterHeroes []string `json:"counter_heroes,omitempty" yaml:"counter_heroes,omitempty"`
}
