package tool

import (
	"github.com/sagerlabs/awesome/tft/data"
	"testing"
)

func mockStore() *data.Store {
	comps := []data.Comp{
		{
			ClusterID:    "1000",
			Name:         "约德尔阵容",
			Tier:         "S",
			AvgPlacement: 3.72,
			Top4Rate:     0.61,
			WinRate:      0.12,
			Units:        []string{"TFT16_Rumble", "TFT16_Kennen", "TFT16_Lulu"},
			BestBuild: data.BuildInfo{
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

	items := data.ItemsFile{
		"TFT_Item_Rageblade": {
			{ClusterID: "1000", CompName: "约德尔阵容", CompTier: "S",
				Carry: "TFT16_Rumble", PriorityScore: 100},
		},
		"TFT_Item_Rabadons": {
			{ClusterID: "1000", CompName: "约德尔阵容", CompTier: "S",
				Carry: "TFT16_Rumble", PriorityScore: 85},
		},
	}

	loc := data.LocalizationFile{
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

	return data.NewStoreFromRaw(comps, items, loc)
}

func TestIntersectionCalc_ThreeWayHit(t *testing.T) {
	store := mockStore()
	calc := NewIntersectionCalc(store)

	in := data.IntersectionInput{
		HeroComps: data.HeroCompsOutput{
			Matches: []data.CompMatch{
				{Comp: data.Comp{ClusterID: "1000", Tier: "S", AvgPlacement: 3.72},
					MatchedUnits: []string{"TFT16_Rumble"}},
			},
		},
		ItemFits: data.ItemFitOutput{
			Results: []data.ItemFitResult{
				{ClusterID: "1000", Carry: "TFT16_Rumble",
					MatchedItems: []string{"TFT_Item_Rageblade"}, TotalScore: 100},
			},
		},
		CompTiers: data.CompTierOutput{
			Tiers: []data.CompTierEntry{
				{ClusterID: "1000", Tier: "S"},
			},
		},
	}

	out, err := calc.Compute(&in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Recommendations) == 0 {
		t.Fatal("期望有推荐结果")
	}

	top := out.Recommendations[0]

	// 三路全命中 + S Tier，置信度应该 >= 0.90
	if top.Confidence < 0.90 {
		t.Errorf("三路命中期望置信度 >= 0.90，得到 %.2f", top.Confidence)
	}
	if len(top.HitSources) != 3 {
		t.Errorf("期望 3 个命中来源，得到 %v", top.HitSources)
	}
}

func TestIntersectionCalc_Fallback(t *testing.T) {
	store := mockStore()
	calc := NewIntersectionCalc(store)

	// 三路全空，触发兜底
	in := data.IntersectionInput{
		HeroComps: data.HeroCompsOutput{},
		ItemFits:  data.ItemFitOutput{},
		CompTiers: data.CompTierOutput{
			Tiers: []data.CompTierEntry{
				{ClusterID: "1000", Tier: "S", AvgPlacement: 3.72},
			},
		},
	}

	out, _ := calc.Compute(&in)

	if len(out.Recommendations) == 0 {
		t.Fatal("兜底时期望返回版本强势阵容")
	}
	if out.Recommendations[0].Confidence != 0.30 {
		t.Errorf("兜底置信度应为 0.30，得到 %.2f", out.Recommendations[0].Confidence)
	}
}
