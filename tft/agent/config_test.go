package agent

import "testing"

func TestAgentMaxTokensFromEnv(t *testing.T) {
	t.Setenv("LLM_MAX_TOKENS", "500")

	got := (&Agent{}).maxTokens()
	if got != 500 {
		t.Fatalf("maxTokens() = %d, want 500", got)
	}
}

func TestAgentMaxTokensCapsLargeValue(t *testing.T) {
	t.Setenv("LLM_MAX_TOKENS", "99999")

	got := (&Agent{}).maxTokens()
	if got != 4096 {
		t.Fatalf("maxTokens() = %d, want 4096", got)
	}
}

func TestAgentMaxTokensDefault(t *testing.T) {
	t.Setenv("LLM_MAX_TOKENS", "")

	got := (&Agent{}).maxTokens()
	if got != 1024 {
		t.Fatalf("maxTokens() = %d, want 1024", got)
	}
}
