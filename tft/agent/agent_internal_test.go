package agent

import (
	"context"
	"errors"
	"testing"
)

// TestDisabledLegacyGraphReturnsSentinel verifies that when the legacy graph is
// not compiled, the legacy entry points fail fast with ErrLegacyGraphDisabled
// instead of panicking on a nil runnable.
func TestDisabledLegacyGraphReturnsSentinel(t *testing.T) {
	a := &Agent{} // runnable left nil, as it would be when DisableLegacyGraph=true

	if _, err := a.Analyze(context.Background(), "剑魔强吗"); !errors.Is(err, ErrLegacyGraphDisabled) {
		t.Fatalf("Analyze: want ErrLegacyGraphDisabled, got %v", err)
	}
	if _, err := a.AnalyzeStream(context.Background(), "剑魔强吗"); !errors.Is(err, ErrLegacyGraphDisabled) {
		t.Fatalf("AnalyzeStream: want ErrLegacyGraphDisabled, got %v", err)
	}
}

func TestMaxTokens(t *testing.T) {
	a := &Agent{}

	cases := []struct {
		name string
		env  string
		want int
	}{
		{"unset falls back to default", "", defaultMaxTokens},
		{"valid override", "256", 256},
		{"zero is ignored", "0", defaultMaxTokens},
		{"negative is ignored", "-5", defaultMaxTokens},
		{"non-numeric is ignored", "abc", defaultMaxTokens},
		{"large value is honoured", "4096", 4096}, // no artificial upper cap
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LLM_MAX_TOKENS", tc.env)
			if got := a.maxTokens(); got != tc.want {
				t.Fatalf("maxTokens() with env %q = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}
