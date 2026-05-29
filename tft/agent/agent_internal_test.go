package agent

import "testing"

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
