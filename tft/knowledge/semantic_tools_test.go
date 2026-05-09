package knowledge

import (
	"encoding/json"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

func TestUnifiedStoreListMetaComps_FieldProjectionAndMetadata(t *testing.T) {
	store := newSemanticToolTestStore(t)

	respBytes, err := store.ListMetaComps(mustJSON(t, contracts.ListMetaCompsRequest{
		Tiers:               []string{"S"},
		Limit:               1,
		DesiredOutputFields: []string{"name", "tier", "avg_placement", "plan"},
	}))
	if err != nil {
		t.Fatalf("ListMetaComps failed: %v", err)
	}

	var resp contracts.ListMetaCompsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Metadata == nil || resp.Metadata.SampleCount != 900 {
		t.Fatalf("expected aggregated metadata sample count, got %#v", resp.Metadata)
	}
	if len(resp.Comps) != 1 {
		t.Fatalf("expected 1 comp, got %d", len(resp.Comps))
	}
	comp := resp.Comps[0]
	if comp["name"] != "狂战士千珏" {
		t.Fatalf("expected best comp name, got %#v", comp["name"])
	}
	if _, ok := comp["count"]; ok {
		t.Fatalf("count should be omitted by field projection: %#v", comp)
	}
	if comp["plan"] == nil {
		t.Fatalf("expected derived comp plan, got %#v", comp)
	}
}

func TestUnifiedStoreGetCompPlan_DerivesFinalBoard(t *testing.T) {
	store := newSemanticToolTestStore(t)

	respBytes, err := store.GetCompPlan(mustJSON(t, contracts.GetCompPlanRequest{ClusterID: "401001"}))
	if err != nil {
		t.Fatalf("GetCompPlan failed: %v", err)
	}

	var resp contracts.GetCompPlanResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Plan == nil {
		t.Fatal("expected comp plan")
	}
	if resp.Plan.Name != "狂战士千珏" {
		t.Fatalf("expected display name, got %q", resp.Plan.Name)
	}
	if len(resp.Plan.Final.Units) != 2 {
		t.Fatalf("expected final units, got %#v", resp.Plan.Final.Units)
	}
	if !resp.Plan.Final.Units[0].IsCore || resp.Plan.Final.Units[0].Items[0] != "鬼索的狂暴之刃" {
		t.Fatalf("expected core unit items, got %#v", resp.Plan.Final.Units[0])
	}
	if len(resp.Plan.Final.Traits) == 0 || resp.Plan.Final.Traits[0].Name != "狂战士" || resp.Plan.Final.Traits[0].Count != 4 {
		t.Fatalf("expected parsed trait marker, got %#v", resp.Plan.Final.Traits)
	}
}

func TestUnifiedStoreGetChampionBuilds_ResolvesAlias(t *testing.T) {
	store := newSemanticToolTestStore(t)

	respBytes, err := store.GetChampionBuilds(mustJSON(t, contracts.GetChampionBuildsRequest{
		Name:  "羊灵",
		Limit: 1,
	}))
	if err != nil {
		t.Fatalf("GetChampionBuilds failed: %v", err)
	}

	var resp contracts.GetChampionBuildsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Champion["name"] != "千珏" {
		t.Fatalf("expected alias to resolve to 千珏, got %#v", resp.Champion)
	}
	builds, ok := resp.Champion["builds"].([]any)
	if !ok || len(builds) != 1 {
		t.Fatalf("expected limited builds, got %#v", resp.Champion["builds"])
	}
}

func TestUnifiedStoreGetItemFits_ResolvesAlias(t *testing.T) {
	store := newSemanticToolTestStore(t)

	respBytes, err := store.GetItemFits(mustJSON(t, contracts.GetItemFitsRequest{Name: "羊刀", Limit: 1}))
	if err != nil {
		t.Fatalf("GetItemFits failed: %v", err)
	}

	var resp contracts.GetItemFitsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Item["name"] != "鬼索的狂暴之刃" {
		t.Fatalf("expected alias to resolve to item, got %#v", resp.Item)
	}
	priorities, ok := resp.Item["priority_list"].([]any)
	if !ok || len(priorities) != 1 {
		t.Fatalf("expected limited priorities, got %#v", resp.Item["priority_list"])
	}
}

func newSemanticToolTestStore(t *testing.T) *UnifiedStore {
	t.Helper()
	knowledgeStore := NewStore()
	knowledgeStore.AddAliases(models.AliasesFile{
		Heroes: map[string]string{"羊灵": "千珏"},
		Items:  map[string]string{"羊刀": "鬼索的狂暴之刃"},
	})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "401001",
		TFTSet:       "TFT17",
		DisplayNames: []models.DisplayName{{Name: "狂战士"}, {Name: "千珏"}},
		Tier:         "S",
		AvgPlacement: 3.02,
		Top4Rate:     0.8042,
		WinRate:      0.2222,
		Count:        500,
		Units:        []string{"千珏", "赛恩"},
		Traits:       []string{"狂战士 (4)", "裁决使 (2)"},
		Levelling:    "fast 8",
		Builds: []models.CompBuild{
			{Unit: "千珏", Items: []string{"鬼索的狂暴之刃", "无尽之刃"}, AvgPlacement: 2.91},
		},
		Trends: []models.Trend{{Day: "2026-05-09", Count: 500}},
	})
	knowledgeStore.AddMetaComp(&models.MetaComp{
		ClusterID:    "401002",
		TFTSet:       "TFT17",
		DisplayNames: []models.DisplayName{{Name: "织命人"}, {Name: "崔斯特"}},
		Tier:         "A",
		AvgPlacement: 3.40,
		Top4Rate:     0.7,
		WinRate:      0.15,
		Count:        400,
		Units:        []string{"崔斯特"},
		Traits:       []string{"织命人 (3)"},
	})
	knowledgeStore.AddMetaChampion(&models.MetaChampion{
		Name: "千珏",
		AppearInComps: []models.CompAppearance{
			{ClusterID: "401001", CompName: "狂战士千珏", Tier: "S", AvgPlacement: 3.02},
		},
		Builds: []models.ChampionBuild{
			{ClusterID: "401001", CompName: "狂战士千珏", Items: []string{"鬼索的狂暴之刃"}, AvgPlacement: 2.91},
			{ClusterID: "401002", CompName: "织命人崔斯特", Items: []string{"无尽之刃"}, AvgPlacement: 3.40},
		},
	})
	knowledgeStore.AddMetaItem(&models.MetaItem{
		Name: "鬼索的狂暴之刃",
		PriorityList: []models.ItemPriority{
			{ClusterID: "401001", CompName: "狂战士千珏", CompTier: "S", CompAvg: 3.02, Carry: "千珏", PriorityScore: 100},
			{ClusterID: "401002", CompName: "织命人崔斯特", CompTier: "A", CompAvg: 3.40, Carry: "崔斯特", PriorityScore: 80},
		},
	})

	store, err := NewUnifiedStore(
		data.NewStoreFromRaw(nil, data.ItemsFile{}, data.LocalizationFile{IDToCN: map[string]string{}, CNToID: map[string]string{}}),
		knowledgeStore,
		&ToolConfig{EnableMeta: true},
	)
	if err != nil {
		t.Fatalf("NewUnifiedStore failed: %v", err)
	}
	return store
}

func mustJSON(t *testing.T, value any) Request {
	t.Helper()
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return Request(out)
}
