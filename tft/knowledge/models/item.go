package models

// Item 装备数据模型
type Item struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Type        string   `json:"type" yaml:"type"` // "basic" | "combined"
	Recipe      []string `json:"recipe,omitempty" yaml:"recipe,omitempty"`
	Effects     Effects  `json:"effects" yaml:"effects"`
	Description string   `json:"description" yaml:"description"`
	BestHeroes  []string `json:"best_heroes,omitempty" yaml:"best_heroes,omitempty"`
	Version     string   `json:"version" yaml:"version"`
}

// Effects 装备效果
type Effects struct {
	AbilityPower   int     `json:"ability_power,omitempty" yaml:"ability_power,omitempty"`
	AttackDamage   int     `json:"attack_damage,omitempty" yaml:"attack_damage,omitempty"`
	AttackSpeed    float64 `json:"attack_speed,omitempty" yaml:"attack_speed,omitempty"`
	Health         int     `json:"health,omitempty" yaml:"health,omitempty"`
	Mana           int     `json:"mana,omitempty" yaml:"mana,omitempty"`
	Armor          int     `json:"armor,omitempty" yaml:"armor,omitempty"`
	MagicResist    int     `json:"magic_resist,omitempty" yaml:"magic_resist,omitempty"`
	CritChance     float64 `json:"crit_chance,omitempty" yaml:"crit_chance,omitempty"`
	CritDamage     float64 `json:"crit_damage,omitempty" yaml:"crit_damage,omitempty"`
	Special        string  `json:"special,omitempty" yaml:"special,omitempty"`
}
