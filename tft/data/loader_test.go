package data

import (
	"testing"
)

func mockStore() *Store {
	comps := []Comp{
		{
			ClusterID:    "1000",
			Name:         "约德尔阵容",
			Tier:         "S",
			AvgPlacement: 3.72,
			Top4Rate:     0.61,
			WinRate:      0.12,
			Units:        []string{"TFT16_Rumble", "TFT16_Kennen", "TFT16_Lulu"},
			BestBuild: BuildInfo{
				Carry: "TFT16_Rumble",
				Items: []string{"TFT_Item_Rageblade", "TFT_Item_Rabadons"},
				PriorityScores: map[string]int{
					"TFT_Item_Rageblade": 100,
					"TFT_Item_Rabadons":  85,
				},
			},
		},
		{
			ClusterID:    "2000",
			Name:         "射手阵容",
			Tier:         "A",
			AvgPlacement: 4.05,
			Top4Rate:     0.52,
			Units:        []string{"TFT16_Jinx", "TFT16_Rumble"},
		},
	}

	items := ItemsFile{
		"TFT_Item_Rageblade": {
			{ClusterID: "1000", CompName: "约德尔阵容", CompTier: "S",
				Carry: "TFT16_Rumble", PriorityScore: 100},
		},
		"TFT_Item_Rabadons": {
			{ClusterID: "1000", CompName: "约德尔阵容", CompTier: "S",
				Carry: "TFT16_Rumble", PriorityScore: 85},
		},
	}

	loc := LocalizationFile{
		IDToCN: map[string]string{
			"TFT16_Rumble":       "兰博",
			"TFT16_Kennen":       "肯能",
			"TFT_Item_Rageblade": "古神狂暴之刃",
			"TFT_Item_Rabadons":  "死亡之帽",
		},
		CNToID: map[string]string{
			"兰博":     "TFT16_Rumble",
			"肯能":     "TFT16_Kennen",
			"古神狂暴之刃": "TFT_Item_Rageblade",
			"死亡之帽":   "TFT_Item_Rabadons",
		},
	}

	return NewStoreFromRaw(comps, items, loc)
}

func TestGetCompsByUnits_MatchScore(t *testing.T) {
	store := mockStore()

	// 输入两个英雄，两个都在约德尔阵容里
	matches := store.GetCompsByUnits([]string{"TFT16_Rumble", "TFT16_Kennen"})

	if len(matches) == 0 {
		t.Fatal("期望有匹配结果，但返回空")
	}

	top := matches[0]
	if top.Comp.ClusterID != "1000" {
		t.Errorf("期望 cluster_id=1000，得到 %s", top.Comp.ClusterID)
	}
	if len(top.MatchedUnits) != 2 {
		t.Errorf("期望命中 2 个英雄，得到 %d", len(top.MatchedUnits))
	}
	if top.MatchScore == 0 {
		t.Error("MatchScore 不应为 0")
	}
}

func TestGetCompsByUnits_MissingUnits(t *testing.T) {
	store := mockStore()

	// 只有兰博，肯能和璐璐缺席
	matches := store.GetCompsByUnits([]string{"TFT16_Rumble"})
	if len(matches) == 0 {
		t.Fatal("期望有匹配结果")
	}

	top := matches[0]
	if len(top.MissingUnits) == 0 {
		t.Error("期望有缺少的英雄，但 MissingUnits 为空")
	}
}

func TestGetItemFitByItems_Merge(t *testing.T) {
	store := mockStore()

	// 两个装备都指向同一阵容，分数应该累加
	results := store.GetItemFitByItems([]string{"TFT_Item_Rageblade", "TFT_Item_Rabadons"})
	if len(results) == 0 {
		t.Fatal("期望有适配结果")
	}

	top := results[0]
	if top.TotalScore != 185 { // 100 + 85
		t.Errorf("期望 TotalScore=185，得到 %d", top.TotalScore)
	}
	if len(top.MatchedItems) != 2 {
		t.Errorf("期望命中 2 个装备，得到 %d", len(top.MatchedItems))
	}
}

func TestResolveUnitID(t *testing.T) {
	store := mockStore()

	cases := []struct {
		input    string
		expected string
	}{
		{"兰博", "TFT16_Rumble"},           // 中文名
		{"TFT16_Rumble", "TFT16_Rumble"}, // 直接 ID
		{"不存在的英雄", ""},                   // 找不到
	}

	for _, c := range cases {
		got := store.ResolveUnitID(c.input)
		if got != c.expected {
			t.Errorf("ResolveUnitID(%q): 期望 %q，得到 %q", c.input, c.expected, got)
		}
	}
}
