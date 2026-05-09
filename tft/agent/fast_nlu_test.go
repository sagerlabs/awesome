package agent

import (
	"testing"
)

func TestFastNLUExtractorParsesChampionWorkQuery(t *testing.T) {
	extractor := &FastNLUExtractor{champions: []string{"剑魔"}}

	ctx, ok := extractor.TryParse("剑魔打工强吗？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "champion_query" {
		t.Fatalf("expected champion_query, got %q", ctx.Intent)
	}
	if ctx.RoleQuery != "work" {
		t.Fatalf("expected work role, got %q", ctx.RoleQuery)
	}
	if _, ok := ctx.Champions["剑魔"]; !ok {
		t.Fatalf("expected champion 剑魔, got %#v", ctx.Champions)
	}
}

func TestFastNLUExtractorParsesItemQuery(t *testing.T) {
	extractor := &FastNLUExtractor{items: []string{"羊刀"}}

	ctx, ok := extractor.TryParse("羊刀给谁？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "item_query" {
		t.Fatalf("expected item_query, got %q", ctx.Intent)
	}
	if len(ctx.Items) != 1 || ctx.Items[0] != "羊刀" {
		t.Fatalf("expected item 羊刀, got %#v", ctx.Items)
	}
}

func TestFastNLUExtractorTreatsItemLineupQuestionAsItemQuery(t *testing.T) {
	extractor := &FastNLUExtractor{items: []string{"珠光护手"}}

	ctx, ok := extractor.TryParse("我有珠光护手，可以玩什么阵容？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "item_query" {
		t.Fatalf("expected item_query, got %q", ctx.Intent)
	}
	if len(ctx.Items) != 1 || ctx.Items[0] != "珠光护手" {
		t.Fatalf("expected item 珠光护手, got %#v", ctx.Items)
	}
}

func TestFastNLUExtractorParsesVerticalQuery(t *testing.T) {
	extractor := &FastNLUExtractor{}

	ctx, ok := extractor.TryParse("四费卡谁能C？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "vertical_query" {
		t.Fatalf("expected vertical_query, got %q", ctx.Intent)
	}
	if ctx.UnitCost == nil || *ctx.UnitCost != 4 {
		t.Fatalf("expected 4-cost query, got %#v", ctx.UnitCost)
	}
	if ctx.RoleQuery != "carry" {
		t.Fatalf("expected carry role, got %q", ctx.RoleQuery)
	}
}

func TestFastNLUExtractorParsesGameState(t *testing.T) {
	extractor := &FastNLUExtractor{
		champions: []string{"千珏"},
		items:     []string{"羊刀", "水银"},
	}

	ctx, ok := extractor.TryParse("我现在3-2，6级，40血，50金币，场上千珏两星，有羊刀水银，能冲吗？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "lineup_recommend" {
		t.Fatalf("expected lineup_recommend, got %q", ctx.Intent)
	}
	if ctx.GameStage == nil || *ctx.GameStage != "3-2" {
		t.Fatalf("expected stage 3-2, got %#v", ctx.GameStage)
	}
	if ctx.Level == nil || *ctx.Level != 6 {
		t.Fatalf("expected level 6, got %#v", ctx.Level)
	}
	if ctx.HP == nil || *ctx.HP != 40 {
		t.Fatalf("expected hp 40, got %#v", ctx.HP)
	}
	if ctx.Gold == nil || *ctx.Gold != 50 {
		t.Fatalf("expected gold 50, got %#v", ctx.Gold)
	}
	if _, ok := ctx.Champions["千珏"]; !ok {
		t.Fatalf("expected champion 千珏, got %#v", ctx.Champions)
	}
	if len(ctx.Items) != 2 {
		t.Fatalf("expected two items, got %#v", ctx.Items)
	}
}

func TestFastNLUExtractorParsesTraitQuery(t *testing.T) {
	extractor := &FastNLUExtractor{traits: []string{"海魔人"}}

	ctx, ok := extractor.TryParse("海魔人能玩吗？")

	if !ok {
		t.Fatal("expected fast nlu hit")
	}
	if ctx.Intent != "trait_query" {
		t.Fatalf("expected trait_query, got %q", ctx.Intent)
	}
	if len(ctx.Traits) != 1 || ctx.Traits[0] != "海魔人" {
		t.Fatalf("expected trait 海魔人, got %#v", ctx.Traits)
	}
}

func TestFastNLUExtractorFallsBackForAmbiguousText(t *testing.T) {
	extractor := &FastNLUExtractor{}

	_, ok := extractor.TryParse("这把怎么运营比较好？")

	if ok {
		t.Fatal("expected ambiguous query to fall back to LLM NLU")
	}
}
