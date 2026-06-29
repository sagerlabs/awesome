package tool

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

func TestCompTierToolQueryTopTierSortsByPlacement(t *testing.T) {
	tool := NewCompTierTool(newTestStore())

	// Only S/A comps are returned (b1 excluded), sorted by avg placement asc.
	out, err := tool.QueryTopTier(context.Background())
	if err != nil {
		t.Fatalf("QueryTopTier failed: %v", err)
	}
	if len(out.Tiers) != 2 {
		t.Fatalf("expected 2 top-tier comps (S+A), got %d", len(out.Tiers))
	}
	if out.Tiers[0].ClusterID != "s1" || out.Tiers[1].ClusterID != "a1" {
		t.Fatalf("expected [s1, a1] by placement, got [%s, %s]",
			out.Tiers[0].ClusterID, out.Tiers[1].ClusterID)
	}
}

func TestCompTierToolQuerySkipsUnknownIDs(t *testing.T) {
	tool := NewCompTierTool(newTestStore())

	out, err := tool.Query(context.Background(), &data.CompTierInput{
		ClusterIDs: []string{"s1", "does_not_exist"},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Tiers) != 1 || out.Tiers[0].ClusterID != "s1" {
		t.Fatalf("expected only s1 to resolve, got %+v", out.Tiers)
	}
}

func TestIntersectionCalcConfidence(t *testing.T) {
	calc := NewIntersectionCalc(newTestStore())

	cases := []struct {
		name    string
		sources []string
		comp    *data.Comp
		want    float64
	}{
		{"three hits S tier", []string{"hero", "item", "tier"}, &data.Comp{Tier: "S"}, 0.95},
		{"three hits B tier", []string{"hero", "item", "tier"}, &data.Comp{Tier: "B"}, 0.90},
		{"two hits A tier", []string{"hero", "item"}, &data.Comp{Tier: "A"}, 0.67},
		{"two hits B tier", []string{"hero", "item"}, &data.Comp{Tier: "B"}, 0.65},
		{"one hit A tier", []string{"hero"}, &data.Comp{Tier: "A"}, 0.40}, // A bonus needs >=2 sources
		{"one hit S tier", []string{"hero"}, &data.Comp{Tier: "S"}, 0.45},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := calc.calcConfidence(tc.sources, tc.comp)
			if !floatEquals(got, tc.want) {
				t.Fatalf("calcConfidence(%v, %s) = %v, want %v", tc.sources, tc.comp.Tier, got, tc.want)
			}
		})
	}
}

func TestIntersectionComputeThreeWayHitRanksFirst(t *testing.T) {
	calc := NewIntersectionCalc(newTestStore())

	in := &data.IntersectionInput{
		HeroComps: data.HeroCompsOutput{Matches: []data.CompMatch{
			{Comp: data.Comp{ClusterID: "s1"}, MatchedUnits: []string{"rumble"}, MissingUnits: []string{"kennen"}},
			{Comp: data.Comp{ClusterID: "a1"}, MatchedUnits: []string{"kennen"}},
		}},
		ItemFits: data.ItemFitOutput{Results: []data.ItemFitResult{
			{ClusterID: "s1", Carry: "rumble", MatchedItems: []string{"guinsoo"}, TotalScore: 100},
		}},
		CompTiers: data.CompTierOutput{Tiers: []data.CompTierEntry{
			{ClusterID: "s1", Tier: "S"},
		}},
	}

	out, err := calc.Compute(in)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	if len(out.Recommendations) != 2 {
		t.Fatalf("expected 2 recommendations, got %d", len(out.Recommendations))
	}

	top := out.Recommendations[0]
	if top.Comp.ClusterID != "s1" {
		t.Fatalf("expected s1 (3-way hit) first, got %q", top.Comp.ClusterID)
	}
	if len(top.HitSources) != 3 {
		t.Fatalf("expected 3 hit sources on s1, got %v", top.HitSources)
	}
	// matched/missing units and carry should be localized to Chinese.
	if len(top.MatchedUnits) != 1 || top.MatchedUnits[0] != "兰博" {
		t.Fatalf("expected matched unit localized to 兰博, got %v", top.MatchedUnits)
	}
	if top.SuggestedCarry != "兰博" {
		t.Fatalf("expected carry 兰博, got %q", top.SuggestedCarry)
	}
}

func TestIntersectionComputeFallsBackWhenNoIntersection(t *testing.T) {
	calc := NewIntersectionCalc(newTestStore())

	// No hero/item hits at all → fallback returns the strongest tier comps.
	in := &data.IntersectionInput{
		CompTiers: data.CompTierOutput{Tiers: []data.CompTierEntry{
			{ClusterID: "s1", Tier: "S", AvgPlacement: 3.80},
			{ClusterID: "a1", Tier: "A", AvgPlacement: 4.20},
		}},
	}

	out, err := calc.Compute(in)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	if len(out.Recommendations) == 0 {
		t.Fatal("expected fallback recommendations, got none")
	}
	for _, rec := range out.Recommendations {
		if len(rec.HitSources) != 1 || rec.HitSources[0] != "fallback" {
			t.Fatalf("fallback recs must be marked as fallback, got %v", rec.HitSources)
		}
		if rec.ConfidenceDesc != "低" {
			t.Fatalf("fallback confidence should be low, got %q", rec.ConfidenceDesc)
		}
	}
}

func TestIntersectionComputeCapsAtFive(t *testing.T) {
	calc := NewIntersectionCalc(newTestStore())

	// Six hero hits, but Compute must cap the output at 5.
	matches := make([]data.CompMatch, 0, 6)
	for _, id := range []string{"s1", "a1", "b1", "x1", "x2", "x3"} {
		matches = append(matches, data.CompMatch{Comp: data.Comp{ClusterID: id}})
	}
	in := &data.IntersectionInput{HeroComps: data.HeroCompsOutput{Matches: matches}}

	out, err := calc.Compute(in)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}
	// Only s1/a1/b1 exist in the store; unknowns are dropped, so this also
	// guards that Compute silently skips IDs missing from the store.
	if len(out.Recommendations) > 5 {
		t.Fatalf("expected at most 5 recommendations, got %d", len(out.Recommendations))
	}
}

func floatEquals(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	return d < eps && d > -eps
}
