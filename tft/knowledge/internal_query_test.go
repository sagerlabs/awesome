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
