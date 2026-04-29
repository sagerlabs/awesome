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
