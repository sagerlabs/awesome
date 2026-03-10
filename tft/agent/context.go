package agent

type Context struct {
	// Intent User intent classification (e.g., "lineup_recommend", "item_query")
	// must be one of the following:
	// "lineup_recommend", "item_query", "trait_query", "champion_query", "playstyle_query", "augment_query"
	Intent string `json:"intent"`

	// Champions Hero units and hero levels input by the user (default level 1)
	Champions map[string]int8 `json:"champions"`

	// Items Equipment and items (raw strings/slangs from user)
	Items []string `json:"items"`

	// Traits Synergy and trait names mentioned
	Traits []string `json:"traits"`

	// Augments 强化符文/海克斯 (新增：对局决策极其重要)
	Augments []string `json:"augments"`

	// ExplicitLineup Lineup explicitly mentioned by the user (e.g., "梅尔九五")
	ExplicitLineup *string `json:"explicit_lineup"`

	// InferredLineup Lineup inferred from user's context (Populated by Go backend, not LLM)
	InferredLineup string `json:"inferred_lineup"`

	// Playstyle Player's preferred playstyle (survival, fun, high_roll, default auto)
	Playstyle string `json:"playstyle"`

	// GameStage Current game stage (e.g., "4-2")
	GameStage *string `json:"game_stage"`

	// Gold amount available
	Gold *int `json:"gold"`

	// Level Player's current level (1-10)
	Level *int `json:"level"`

	// HP Player's current health (1-100)
	HP *int `json:"hp"`
}
