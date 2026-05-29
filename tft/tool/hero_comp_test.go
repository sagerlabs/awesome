package tool

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

func TestHeroCompsToolWeightByTier(t *testing.T) {
	tool := NewHeroCompsTool(nil)

	cases := []struct {
		tier string
		want float64
	}{
		{"S", 12.0}, // 10 * 1.2
		{"A", 11.0}, // 10 * 1.1
		{"B", 10.0}, // unchanged
		{"C", 10.0}, // unchanged
		{"", 10.0},  // unknown tier unchanged
	}
	for _, tc := range cases {
		t.Run(tc.tier, func(t *testing.T) {
			if got := tool.weightByTier(10.0, tc.tier); got != tc.want {
				t.Fatalf("weightByTier(10, %q) = %v, want %v", tc.tier, got, tc.want)
			}
		})
	}
}

func TestHeroCompsToolQueryRanksByWeightedScore(t *testing.T) {
	tool := NewHeroCompsTool(newTestStore())

	// kennen belongs to both s1 (S tier) and a1 (A tier), each with one of two
	// core units matched → raw match score 0.5 for both. After tier weighting,
	// s1 (0.5*1.2=0.60) must outrank a1 (0.5*1.1=0.55).
	out, err := tool.Query(context.Background(), &data.HeroCompsInput{Heroes: []string{"kennen"}})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(out.Matches))
	}
	if out.Matches[0].Comp.ClusterID != "s1" {
		t.Fatalf("expected S-tier comp first, got %q", out.Matches[0].Comp.ClusterID)
	}
	if out.Matches[1].Comp.ClusterID != "a1" {
		t.Fatalf("expected A-tier comp second, got %q", out.Matches[1].Comp.ClusterID)
	}
}

func TestHeroCompsToolQueryRespectsTopN(t *testing.T) {
	tool := NewHeroCompsTool(newTestStore())

	out, err := tool.Query(context.Background(), &data.HeroCompsInput{
		Heroes: []string{"kennen"},
		TopN:   1,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Matches) != 1 {
		t.Fatalf("expected TopN to cap results at 1, got %d", len(out.Matches))
	}
	if out.Matches[0].Comp.ClusterID != "s1" {
		t.Fatalf("expected the strongest comp to survive the cap, got %q", out.Matches[0].Comp.ClusterID)
	}
}

func TestHeroCompsToolQueryNoMatch(t *testing.T) {
	tool := NewHeroCompsTool(newTestStore())

	out, err := tool.Query(context.Background(), &data.HeroCompsInput{Heroes: []string{"unknown_hero"}})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Matches) != 0 {
		t.Fatalf("expected no matches for unknown hero, got %d", len(out.Matches))
	}
}
