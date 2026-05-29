package tool

import (
	"context"
	"testing"

	"github.com/sagerlabs/awesome/tft/data"
)

func TestItemFitToolWeightByTier(t *testing.T) {
	tool := NewItemFitTool(nil)

	cases := []struct {
		tier string
		want int
	}{
		{"S", 120}, // int(100 * 1.2)
		{"A", 110}, // int(100 * 1.1)
		{"B", 100}, // unchanged
		{"", 100},  // unknown tier unchanged
	}
	for _, tc := range cases {
		t.Run(tc.tier, func(t *testing.T) {
			if got := tool.weightByTier(100, tc.tier); got != tc.want {
				t.Fatalf("weightByTier(100, %q) = %d, want %d", tc.tier, got, tc.want)
			}
		})
	}
}

func TestItemFitToolQueryEmptyItems(t *testing.T) {
	tool := NewItemFitTool(newTestStore())

	out, err := tool.Query(context.Background(), &data.ItemFitInput{Items: nil})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Results) != 0 {
		t.Fatalf("expected no results for empty item list, got %d", len(out.Results))
	}
}

func TestItemFitToolQueryRanksAndAggregates(t *testing.T) {
	tool := NewItemFitTool(newTestStore())

	// guinsoo + rabadon both fit s1/rumble: scores 100+90=190, then S weighting
	// → int(190*1.2)=228. guinsoo also fits a1/kennen: 80, A weighting → 88.
	// s1 must rank first.
	out, err := tool.Query(context.Background(), &data.ItemFitInput{Items: []string{"guinsoo", "rabadon"}})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out.Results))
	}

	first := out.Results[0]
	if first.ClusterID != "s1" {
		t.Fatalf("expected s1 first, got %q", first.ClusterID)
	}
	if first.TotalScore != 228 {
		t.Fatalf("expected aggregated+weighted score 228, got %d", first.TotalScore)
	}
	if len(first.MatchedItems) != 2 {
		t.Fatalf("expected both items aggregated onto s1, got %d", len(first.MatchedItems))
	}
	if out.Results[1].ClusterID != "a1" {
		t.Fatalf("expected a1 second, got %q", out.Results[1].ClusterID)
	}
}
