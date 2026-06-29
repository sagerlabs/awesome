package tool

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

func TestTierToScore(t *testing.T) {
	cases := map[string]int{"S": 4, "A": 3, "B": 2, "C": 1, "": 0, "Z": 0}
	for tier, want := range cases {
		if got := tierToScore(tier); got != want {
			t.Errorf("tierToScore(%q) = %d, want %d", tier, got, want)
		}
	}
}

func TestTierWeight(t *testing.T) {
	cases := map[string]float64{"S": 1.5, "A": 1.2, "B": 1.0, "C": 0.8, "": 1.0}
	for tier, want := range cases {
		if got := tierWeight(tier); got != want {
			t.Errorf("tierWeight(%q) = %v, want %v", tier, got, want)
		}
	}
}

func TestIsHeroMainCarry(t *testing.T) {
	tool := NewPriorityRecommendTool(newTestStore())

	comp := data.Comp{
		BestBuild: data.BuildInfo{Carry: "rumble"},
		AllBuilds: []data.BuildInfo{{Carry: "kennen"}},
	}

	if !tool.isHeroMainCarry(comp, []string{"rumble"}) {
		t.Error("rumble is the best-build carry, should be main carry")
	}
	if !tool.isHeroMainCarry(comp, []string{"kennen"}) {
		t.Error("kennen is an all-builds carry, should count as main carry")
	}
	if tool.isHeroMainCarry(comp, []string{"vex"}) {
		t.Error("vex is not a carry of this comp")
	}
}

func TestMergeAndDedupMatches(t *testing.T) {
	tool := NewPriorityRecommendTool(newTestStore())

	a := []data.CompMatch{{Comp: data.Comp{ClusterID: "s1"}}, {Comp: data.Comp{ClusterID: "a1"}}}
	b := []data.CompMatch{{Comp: data.Comp{ClusterID: "a1"}}, {Comp: data.Comp{ClusterID: "b1"}}}

	merged := tool.mergeAndDedupMatches(a, b)
	if len(merged) != 3 {
		t.Fatalf("expected 3 unique comps after dedup, got %d", len(merged))
	}
	seen := map[string]bool{}
	for _, m := range merged {
		if seen[m.Comp.ClusterID] {
			t.Fatalf("duplicate cluster id %q in merged result", m.Comp.ClusterID)
		}
		seen[m.Comp.ClusterID] = true
	}
}

func TestSelectCompsByHeroWithPriorityPutsMainCarryFirst(t *testing.T) {
	tool := NewPriorityRecommendTool(newTestStore())

	// rumble is the carry of s1; kennen appears in s1 and a1 but is carry of a1.
	// Querying rumble should surface s1 with the main-carry flag set and ranked
	// ahead of any non-carry match.
	out, err := tool.SelectCompsByHeroWithPriority(context.Background(), &HeroPriorityInput{
		HeroIDs: []string{"rumble"},
	})
	if err != nil {
		t.Fatalf("SelectCompsByHeroWithPriority failed: %v", err)
	}
	if len(out.Recommendations) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	top := out.Recommendations[0]
	if top.Comp.ClusterID != "s1" {
		t.Fatalf("expected s1 first, got %q", top.Comp.ClusterID)
	}
	if !top.IsMainCarry {
		t.Error("expected rumble to be flagged as main carry of s1")
	}
	if !out.HasSTier {
		t.Error("expected HasSTier true since s1 is S tier")
	}
}

func TestSelectCompsByHeroWithPriorityCapsAtThree(t *testing.T) {
	tool := NewPriorityRecommendTool(newTestStore())

	// All three fixture comps contain one of these heroes.
	out, err := tool.SelectCompsByHeroWithPriority(context.Background(), &HeroPriorityInput{
		HeroIDs: []string{"rumble", "kennen", "vex"},
	})
	if err != nil {
		t.Fatalf("SelectCompsByHeroWithPriority failed: %v", err)
	}
	if len(out.Recommendations) > 3 {
		t.Fatalf("expected at most 3 recommendations, got %d", len(out.Recommendations))
	}
}

func TestBuildTraitTierIndexAggregatesSTierTraits(t *testing.T) {
	tool := NewPriorityRecommendTool(newTestStore())

	// Only s1 is S tier; its trait is "yordle". Index should contain yordle.
	index := tool.BuildTraitTierIndex()
	if len(index.Tiers) == 0 {
		t.Fatal("expected at least one trait in the index")
	}
	found := false
	for _, entry := range index.Tiers {
		if entry.TraitID == "yordle" {
			found = true
			if entry.Appearance != 1 {
				t.Errorf("expected yordle appearance 1, got %d", entry.Appearance)
			}
		}
	}
	if !found {
		t.Error("expected yordle trait from the S-tier comp in the index")
	}
}
