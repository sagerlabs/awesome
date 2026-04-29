package knowledge

import (
	"encoding/json"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

func TestInternalQueryNLUData_TopCompsFallbackForOpenLineupRecommend(t *testing.T) {
	store := data.NewStoreFromRaw(
		[]data.Comp{
			testComp("394001", "TFT16_A", "S", 3.40),
			testComp("394002", "TFT16_Sorcerer, TFT16_Taric", "S", 3.10),
			testComp("394003", "TFT16_B", "A", 3.30),
			testComp("394004", "TFT16_C", "B", 3.00),
		},
		data.ItemsFile{},
		data.LocalizationFile{
			IDToCN: map[string]string{
				"TFT16_Sorcerer":        "法师",
				"TFT16_Taric":           "塔里克",
				"TFT_Item_Rabadons":     "灭世者的死亡之帽",
				"TFT_Item_NashorsTooth": "纳什之牙",
			},
			CNToID: map[string]string{
				"法师":       "TFT16_Sorcerer",
				"塔里克":      "TFT16_Taric",
				"灭世者的死亡之帽": "TFT_Item_Rabadons",
				"纳什之牙":     "TFT_Item_NashorsTooth",
			},
		},
	)

	result := internalQueryNLUData(contracts.QueryNLURequest{Intent: "lineup_recommend"}, store)

	if len(result.MatchedComps) != 3 {
		t.Fatalf("expected 3 top comps, got %d", len(result.MatchedComps))
	}
	if result.MatchedComps[0].ClusterID != "394002" {
		t.Fatalf("expected best S/A comp first, got %s", result.MatchedComps[0].ClusterID)
	}
	if result.MatchedComps[0].Name != "法师、塔里克" {
		t.Fatalf("expected localized comp name, got %q", result.MatchedComps[0].Name)
	}
	if result.MatchedComps[0].BestBuild.Carry != "塔里克" {
		t.Fatalf("expected localized carry, got %q", result.MatchedComps[0].BestBuild.Carry)
	}
	if got := result.MatchedComps[0].BestBuild.Items[0]; got != "灭世者的死亡之帽" {
		t.Fatalf("expected localized item, got %q", got)
	}
	if result.MatchedComps[2].ClusterID == "394004" {
		t.Fatalf("did not expect B tier comp in top S/A fallback")
	}
}

func TestInternalQueryNLUData_NoTopCompsFallbackForExplicitLineup(t *testing.T) {
	store := data.NewStoreFromRaw(
		[]data.Comp{testComp("394001", "TFT16_A", "S", 3.40)},
		data.ItemsFile{},
		data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}},
	)
	explicitLineup := "夜幽锐雯"

	result := internalQueryNLUData(contracts.QueryNLURequest{
		Intent:         "lineup_recommend",
		ExplicitLineup: &explicitLineup,
	}, store)

	if len(result.MatchedComps) != 0 {
		t.Fatalf("expected no fallback for explicit lineup query, got %d comps", len(result.MatchedComps))
	}
}

func TestUnifiedStoreQueryNLU_EnrichesCompSummaryWithMetaDisplayName(t *testing.T) {
	dataStore := data.NewStoreFromRaw(
		[]data.Comp{testComp("394014", "TFT16_Augment_RumbleCarry", "S", 3.84)},
		data.ItemsFile{
			"TFT_Item_GuinsoosRageblade": {
				{
					ClusterID:     "394014",
					CompName:      "TFT16_Augment_RumbleCarry",
					CompTier:      "S",
					CompAvg:       3.8451,
					Carry:         "TFT16_Rumble",
					PriorityScore: 100,
				},
			},
		},
		data.LocalizationFile{
			IDToCN: map[string]string{
				"TFT16_Augment_RumbleCarry":  "TFT16_Augment_RumbleCarry",
				"TFT16_Rumble":               "兰博",
				"TFT_Item_GuinsoosRageblade": "鬼索的狂暴之刃",
			},
			CNToID: map[string]string{
				"鬼索的狂暴之刃": "TFT_Item_GuinsoosRageblade",
			},
		},
	)
	knowledgeStore := NewStore()
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "394014",
		NameString:   "TFT16_Augment_RumbleCarry",
		DisplayNames: []models.DisplayName{{Name: "约德尔人"}, {Name: "兰博"}},
		Tier:         "S",
		AvgPlacement: 3.8451,
		Top4Rate:     0.6202,
		WinRate:      0.1791,
		Count:        25060,
		Units:        []string{"兰博", "凯南", "璐璐"},
		Traits:       []string{"约德尔人 (4)"},
		Stars:        []string{"兰博", "凯南", "厄斐琉斯"},
		Levelling:    "lvl 5",
		Builds: []models.CompBuild{
			{
				Unit:  "兰博",
				Items: []string{"鬼索的狂暴之刃", "灭世者的死亡之帽"},
				PriorityScores: map[string]int{
					"鬼索的狂暴之刃": 100,
				},
				AvgPlacement: 3.38,
			},
		},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent: "item_query",
		Items:  []string{"鬼索的狂暴之刃"},
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.MatchedComps) != 1 {
		t.Fatalf("expected 1 matched comp, got %d", len(resp.MatchedComps))
	}
	comp := resp.MatchedComps[0]
	if comp.Name != "约德尔人兰博" {
		t.Fatalf("expected meta display name, got %q", comp.Name)
	}
	if comp.Name == "TFT16_Augment_RumbleCarry" {
		t.Fatalf("internal name leaked into response")
	}
	if comp.Levelling != "lvl 5" {
		t.Fatalf("expected meta levelling, got %q", comp.Levelling)
	}
	if len(comp.Stars) != 2 {
		t.Fatalf("expected stars filtered to units, got %#v", comp.Stars)
	}
	if len(resp.MatchedItems) != 1 || len(resp.MatchedItems[0].CompInfos) != 1 {
		t.Fatalf("expected enriched matched item comp info, got %#v", resp.MatchedItems)
	}
	itemComp := resp.MatchedItems[0].CompInfos[0]
	if itemComp.CompName != "约德尔人兰博" {
		t.Fatalf("expected item comp display name, got %q", itemComp.CompName)
	}
	if itemComp.CarryName != "兰博" {
		t.Fatalf("expected localized item carry name, got %q", itemComp.CarryName)
	}
}

func TestUnifiedStoreQueryNLU_NormalizesAliasesBeforeQuery(t *testing.T) {
	dataStore := data.NewStoreFromRaw(
		[]data.Comp{testComp("394014", "TFT16_Augment_RumbleCarry", "S", 3.84)},
		data.ItemsFile{
			"TFT_Item_GuinsoosRageblade": {
				{
					ClusterID:     "394014",
					CompName:      "TFT16_Augment_RumbleCarry",
					CompTier:      "S",
					CompAvg:       3.8451,
					Carry:         "TFT16_Rumble",
					PriorityScore: 100,
				},
			},
		},
		data.LocalizationFile{
			IDToCN: map[string]string{
				"TFT16_Augment_RumbleCarry":  "TFT16_Augment_RumbleCarry",
				"TFT16_Rumble":               "兰博",
				"TFT_Item_GuinsoosRageblade": "鬼索的狂暴之刃",
			},
			CNToID: map[string]string{
				"鬼索的狂暴之刃": "TFT_Item_GuinsoosRageblade",
			},
		},
	)
	knowledgeStore := NewStore()
	knowledgeStore.AddAliases(models.AliasesFile{
		Items: map[string]string{
			"羊刀": "鬼索的狂暴之刃",
		},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent: "item_query",
		Items:  []string{"羊刀"},
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.NormalizedTerms) != 1 {
		t.Fatalf("expected normalized term, got %#v", resp.NormalizedTerms)
	}
	term := resp.NormalizedTerms[0]
	if term.Type != "item" || term.Raw != "羊刀" || term.Normalized != "鬼索的狂暴之刃" {
		t.Fatalf("unexpected normalized term: %#v", term)
	}
	if len(resp.Ctx.Items) != 1 || resp.Ctx.Items[0] != "鬼索的狂暴之刃" {
		t.Fatalf("expected normalized ctx item, got %#v", resp.Ctx.Items)
	}
	if len(resp.MatchedItems) != 1 {
		t.Fatalf("expected alias to hit item knowledge, got %#v", resp.MatchedItems)
	}
}

func TestUnifiedStoreQueryNLU_ChampionAliasBuildsChampionInsight(t *testing.T) {
	dataStore := data.NewStoreFromRaw(nil, data.ItemsFile{}, data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}})
	knowledgeStore := NewStore()
	knowledgeStore.AddAliases(models.AliasesFile{
		Heroes: map[string]string{
			"剑魔": "亚托克斯",
		},
	})
	knowledgeStore.AddChampionProfile("亚托克斯", &models.ChampionProfile{Cost: 1})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "401007",
		DisplayNames: []models.DisplayName{{Name: "海魔人"}, {Name: "卑尔维斯"}},
		Tier:         "S",
		AvgPlacement: 4.18,
		Top4Rate:     0.58,
		WinRate:      0.10,
		Units:        []string{"亚托克斯", "卑尔维斯"},
		Traits:       []string{"海魔人 (3)", "堡垒卫士 (2)"},
	})
	knowledgeStore.AddMetaChampion(&models.MetaChampion{
		Name: "亚托克斯",
		AppearInComps: []models.CompAppearance{
			{ClusterID: "401007", CompName: "海魔人、卑尔维斯", Tier: "S", AvgPlacement: 4.18},
		},
		Builds: []models.ChampionBuild{
			{ClusterID: "401007", Items: []string{"饮血剑", "泰坦的坚决"}, AvgPlacement: 4.10},
		},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent:    "champion_query",
		Champions: map[string]int8{"剑魔": 1},
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.NormalizedTerms) != 1 || resp.NormalizedTerms[0].Raw != "剑魔" || resp.NormalizedTerms[0].Normalized != "亚托克斯" {
		t.Fatalf("expected sword demon alias to be recorded, got %#v", resp.NormalizedTerms)
	}
	if _, ok := resp.Ctx.Champions["亚托克斯"]; !ok {
		t.Fatalf("expected normalized champion in ctx, got %#v", resp.Ctx.Champions)
	}
	if len(resp.MatchedChampions) != 1 || resp.MatchedChampions[0].Name != "亚托克斯" {
		t.Fatalf("expected champion insight for 亚托克斯, got %#v", resp.MatchedChampions)
	}
	if len(resp.MatchedChampions[0].BestComps) == 0 || resp.MatchedChampions[0].BestComps[0].Name != "海魔人卑尔维斯" {
		t.Fatalf("expected best comp for 亚托克斯, got %#v", resp.MatchedChampions[0].BestComps)
	}
}

func TestUnifiedStoreQueryNLU_BuildsTraitInsights(t *testing.T) {
	dataStore := data.NewStoreFromRaw(nil, data.ItemsFile{}, data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}})
	knowledgeStore := NewStore()
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "394100",
		DisplayNames: []models.DisplayName{{Name: "法师"}, {Name: "拉克丝"}},
		Tier:         "S",
		AvgPlacement: 3.20,
		Top4Rate:     0.66,
		WinRate:      0.22,
		Units:        []string{"拉克丝", "安妮", "奥瑞利安·索尔"},
		Traits:       []string{"法师 (3)", "巨神峰 (1)"},
		Builds: []models.CompBuild{
			{Unit: "拉克丝", Items: []string{"珠光护手", "朔极之矛"}, AvgPlacement: 3.10},
		},
	})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "394101",
		DisplayNames: []models.DisplayName{{Name: "法师"}, {Name: "安妮"}},
		Tier:         "A",
		AvgPlacement: 3.60,
		Top4Rate:     0.60,
		WinRate:      0.18,
		Units:        []string{"安妮", "佐伊"},
		Traits:       []string{"法师 (5)"},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent: "trait_query",
		Traits: []string{"法师"},
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.MatchedTraits) != 1 {
		t.Fatalf("expected 1 matched trait, got %#v", resp.MatchedTraits)
	}
	trait := resp.MatchedTraits[0]
	if trait.Name != "法师" {
		t.Fatalf("expected trait name 法师, got %q", trait.Name)
	}
	if len(trait.BestComps) != 2 || trait.BestComps[0].Name != "法师拉克丝" {
		t.Fatalf("expected sorted best comps, got %#v", trait.BestComps)
	}
	if len(resp.MatchedComps) == 0 || resp.MatchedComps[0].ClusterID != "394100" {
		t.Fatalf("expected trait query to expose representative comps, got %#v", resp.MatchedComps)
	}
}

func TestUnifiedStoreQueryNLU_BuildsVerticalChampionInsights(t *testing.T) {
	dataStore := data.NewStoreFromRaw(nil, data.ItemsFile{}, data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}})
	knowledgeStore := NewStore()
	knowledgeStore.AddChampionProfile("拉克丝", &models.ChampionProfile{Cost: 4})
	knowledgeStore.AddChampionProfile("布隆", &models.ChampionProfile{Cost: 4})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "394200",
		DisplayNames: []models.DisplayName{{Name: "法师"}, {Name: "拉克丝"}},
		Tier:         "S",
		AvgPlacement: 3.20,
		Top4Rate:     0.66,
		WinRate:      0.22,
		Units:        []string{"拉克丝", "安妮"},
		Traits:       []string{"法师 (3)"},
	})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "394201",
		DisplayNames: []models.DisplayName{{Name: "护卫"}, {Name: "布隆"}},
		Tier:         "S",
		AvgPlacement: 3.30,
		Top4Rate:     0.64,
		WinRate:      0.18,
		Units:        []string{"布隆", "盖伦"},
		Traits:       []string{"护卫 (2)"},
	})
	knowledgeStore.AddMetaChampion(&models.MetaChampion{
		Name: "拉克丝",
		AppearInComps: []models.CompAppearance{
			{ClusterID: "394200", CompName: "法师、拉克丝", Tier: "S", AvgPlacement: 3.20},
		},
		Builds: []models.ChampionBuild{
			{ClusterID: "394200", Items: []string{"珠光护手", "朔极之矛"}, AvgPlacement: 3.10},
		},
	})
	knowledgeStore.AddMetaChampion(&models.MetaChampion{
		Name: "布隆",
		AppearInComps: []models.CompAppearance{
			{ClusterID: "394201", CompName: "护卫、布隆", Tier: "S", AvgPlacement: 3.30},
		},
		Builds: []models.ChampionBuild{
			{ClusterID: "394201", Items: []string{"狂徒铠甲", "棘刺背心"}, AvgPlacement: 3.25},
		},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	cost := 4
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent:    "vertical_query",
		UnitCost:  &cost,
		RoleQuery: "all",
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.MatchedChampions) != 2 {
		t.Fatalf("expected 2 matched champions, got %#v", resp.MatchedChampions)
	}
	if resp.MatchedChampions[0].Cost != 4 {
		t.Fatalf("expected cost filter to keep 4-cost champions, got %#v", resp.MatchedChampions[0])
	}
	if resp.MatchedChampions[0].Role == "" {
		t.Fatalf("expected role inferred from items, got %#v", resp.MatchedChampions[0])
	}
	if len(resp.MatchedComps) == 0 {
		t.Fatalf("expected vertical query to expose representative comps")
	}
}

func TestUnifiedStoreQueryNLU_AttachesRelevantPatchNotes(t *testing.T) {
	dataStore := data.NewStoreFromRaw(nil, data.ItemsFile{}, data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}})
	knowledgeStore := NewStore()
	knowledgeStore.AddPatchNote(&models.PatchNote{
		Patch:       "17.1",
		Title:       "17.1云顶之弈版本更新公告",
		Source:      "Tencent LOL",
		PublishedAt: "2026-04-15 19:04:36",
		Sections: []models.PatchNoteSection{
			{
				Type:       "item",
				Title:      "装备",
				Summary:    "坦克装备整体削弱",
				ImpactTags: []string{"itemization", "frontline"},
				Details:    []string{"【棘刺背心】额外生命值：9%->6%"},
			},
		},
	})

	store, err := NewUnifiedStore(dataStore, knowledgeStore, &ToolConfig{EnableMeta: true})
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	req, err := json.Marshal(contracts.QueryNLURequest{
		Intent: "item_query",
		Items:  []string{"棘刺背心"},
	})
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}

	respBytes, err := store.QueryNLU(req)
	if err != nil {
		t.Fatalf("QueryNLU failed: %v", err)
	}
	var resp contracts.QueryNLUResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}

	if len(resp.PatchNotes) != 1 {
		t.Fatalf("expected 1 patch note insight, got %#v", resp.PatchNotes)
	}
	if resp.PatchNotes[0].SectionTitle != "装备" {
		t.Fatalf("expected equipment patch note, got %#v", resp.PatchNotes[0])
	}
}

func testComp(clusterID string, name string, tier string, avg float64) data.Comp {
	return data.Comp{
		ClusterID:    clusterID,
		Name:         name,
		Tier:         tier,
		AvgPlacement: avg,
		Top4Rate:     0.60,
		WinRate:      0.20,
		Count:        1000,
		Units:        []string{"TFT16_Taric"},
		Traits:       []string{"TFT16_Sorcerer_1"},
		BestBuild: data.BuildInfo{
			Carry: "TFT16_Taric",
			Items: []string{"TFT_Item_Rabadons", "TFT_Item_NashorsTooth"},
			PriorityScores: map[string]int{
				"TFT_Item_Rabadons": 100,
			},
			AvgPlacement: 3.0,
		},
	}
}
