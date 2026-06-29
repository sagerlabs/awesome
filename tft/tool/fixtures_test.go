package tool

import "github.com/sagerlabs/awesome/tft/data"

// newTestStore builds an in-memory data.Store with a small, hand-checked
// dataset so the recommendation tools can be tested without loading JSON files.
//
// The fixture is deliberately tiny and explicit:
//
//	comp "s1": S tier, units [rumble, kennen], carry rumble, avg 3.80
//	comp "a1": A tier, units [kennen, vex],    carry kennen, avg 4.20
//	comp "b1": B tier, units [vex],            carry vex,    avg 4.60
//
//	item "guinsoo" fits: s1 (carry rumble, 100), a1 (carry kennen, 80)
//	item "rabadon" fits: s1 (carry rumble, 90)
func newTestStore() *data.Store {
	comps := []data.Comp{
		{
			ClusterID:    "s1",
			Name:         "RumbleCarry",
			Tier:         "S",
			AvgPlacement: 3.80,
			Top4Rate:     0.62,
			WinRate:      0.18,
			Units:        []string{"rumble", "kennen"},
			Traits:       []string{"yordle"},
			BestBuild:    data.BuildInfo{Carry: "rumble", Items: []string{"guinsoo", "rabadon"}},
		},
		{
			ClusterID:    "a1",
			Name:         "KennenFlex",
			Tier:         "A",
			AvgPlacement: 4.20,
			Top4Rate:     0.55,
			WinRate:      0.12,
			Units:        []string{"kennen", "vex"},
			Traits:       []string{"yordle"},
			BestBuild:    data.BuildInfo{Carry: "kennen", Items: []string{"guinsoo"}},
		},
		{
			ClusterID:    "b1",
			Name:         "VexSolo",
			Tier:         "B",
			AvgPlacement: 4.60,
			Top4Rate:     0.48,
			WinRate:      0.08,
			Units:        []string{"vex"},
			Traits:       []string{"emo"},
			BestBuild:    data.BuildInfo{Carry: "vex", Items: []string{"rabadon"}},
		},
	}

	items := data.ItemsFile{
		"guinsoo": {
			{ClusterID: "s1", CompName: "RumbleCarry", CompTier: "S", CompAvg: 3.80, Carry: "rumble", PriorityScore: 100},
			{ClusterID: "a1", CompName: "KennenFlex", CompTier: "A", CompAvg: 4.20, Carry: "kennen", PriorityScore: 80},
		},
		"rabadon": {
			{ClusterID: "s1", CompName: "RumbleCarry", CompTier: "S", CompAvg: 3.80, Carry: "rumble", PriorityScore: 90},
		},
	}

	loc := data.LocalizationFile{
		IDToCN: map[string]string{
			"rumble": "兰博",
			"kennen": "凯南",
			"vex":    "薇古丝",
		},
		CNToID: map[string]string{
			"兰博":  "rumble",
			"凯南":  "kennen",
			"薇古丝": "vex",
		},
	}

	return data.NewStoreFromRaw(comps, items, loc)
}
