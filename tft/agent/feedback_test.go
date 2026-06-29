package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
)

func TestClassifyAdviceFeedback(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantType  string
		wantEmpty bool
	}{
		{name: "rejected", input: "不对，这个版本不是这样", wantType: FeedbackRejected},
		{name: "needs context", input: "我现在3-2，血量40，装备有羊刀", wantType: FeedbackNeedsContext},
		{name: "continued", input: "那这套几级启动", wantType: FeedbackAcceptedOrContinued},
		{name: "unrelated", input: "今天天气不错", wantType: FeedbackUnrelated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := classifyAdviceFeedback(tc.input)
			if got != tc.wantType {
				t.Fatalf("expected %s, got %s", tc.wantType, got)
			}
		})
	}
}

func TestFeedbackMemoryDetectsRejectedFromPreviousAdvice(t *testing.T) {
	memory := NewFeedbackMemory()
	const sess = "session-a"

	if feedback := memory.Detect(sess, "不对"); feedback != nil {
		t.Fatalf("expected no feedback before previous advice, got %#v", feedback)
	}

	memory.Record(sess, "剑魔打工强吗", "能拿来打工，但不要追三。", "champion_query")
	feedback := memory.Detect(sess, "不对，这个答非所问")

	if feedback == nil {
		t.Fatal("expected feedback")
	}
	if feedback.Type != FeedbackRejected {
		t.Fatalf("expected rejected, got %s", feedback.Type)
	}
	if feedback.PreviousUserInput != "剑魔打工强吗" {
		t.Fatalf("unexpected previous input: %s", feedback.PreviousUserInput)
	}
	if !strings.Contains(feedback.LastAdviceSummary, "能拿来打工") {
		t.Fatalf("unexpected advice summary: %s", feedback.LastAdviceSummary)
	}
}

// TestFeedbackMemoryIsolatesSessions is the regression guard for the cross-talk
// bug: one conversation's previous turn must never surface in another's.
func TestFeedbackMemoryIsolatesSessions(t *testing.T) {
	memory := NewFeedbackMemory()

	memory.Record("session-a", "剑魔打工强吗", "能拿来打工。", "champion_query")

	// A different session asking a follow-up must not see session-a's advice.
	if feedback := memory.Detect("session-b", "不对"); feedback != nil {
		t.Fatalf("session-b should not see session-a's advice, got %#v", feedback)
	}

	// The original session still sees its own previous turn.
	if feedback := memory.Detect("session-a", "不对"); feedback == nil {
		t.Fatal("session-a should still detect feedback against its own advice")
	}
}

// TestFeedbackMemoryEmptySessionIsStateless verifies anonymous requests carry no
// shared state, so they cannot cross-talk through a default bucket.
func TestFeedbackMemoryEmptySessionIsStateless(t *testing.T) {
	memory := NewFeedbackMemory()

	memory.Record("", "剑魔打工强吗", "能拿来打工。", "champion_query")
	if feedback := memory.Detect("", "不对"); feedback != nil {
		t.Fatalf("empty session must stay stateless, got %#v", feedback)
	}
}

// TestFeedbackMemoryEvictsExpiredSessions checks the TTL bound: a turn older
// than the TTL is forgotten so memory does not grow without limit.
func TestFeedbackMemoryEvictsExpiredSessions(t *testing.T) {
	memory := NewFeedbackMemory()
	memory.ttl = time.Minute

	clock := time.Unix(0, 0)
	memory.now = func() time.Time { return clock }

	memory.Record("session-a", "剑魔打工强吗", "能拿来打工。", "champion_query")

	// Advance past the TTL: the previous turn should be gone.
	clock = clock.Add(2 * time.Minute)
	if feedback := memory.Detect("session-a", "不对"); feedback != nil {
		t.Fatalf("expired session should be forgotten, got %#v", feedback)
	}
}

func TestBuildNluFormatPromptIncludesRejectedFeedbackInstruction(t *testing.T) {
	prompt, err := BuildNluFormatPrompt(&NluEnrichedContext{
		UserInput: "不对，这个版本不是这样",
		Ctx: contracts.QueryNLURequest{
			Intent: "lineup_recommend",
		},
		Feedback: &contracts.AdviceFeedback{
			Type:              FeedbackRejected,
			Reason:            "用户明确指出上一轮建议没有命中问题",
			PreviousUserInput: "当前版本最强阵容是什么",
			LastAdviceSummary: "推荐旧版本阵容",
		},
	})
	if err != nil {
		t.Fatalf("BuildNluFormatPrompt failed: %v", err)
	}

	for _, text := range []string{"上轮反馈", "上一轮建议可能没命中", "不要重复上一轮原话"} {
		if !strings.Contains(prompt, text) {
			t.Fatalf("prompt should contain %q, got:\n%s", text, prompt)
		}
	}
}

func TestAppendFeedbackCaseWritesRejectedCase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "feedback_cases.jsonl")

	err := AppendFeedbackCase(path, "不对", "old answer", &contracts.AdviceFeedback{
		Type:              FeedbackRejected,
		PreviousUserInput: "原问题",
		LastAdviceSummary: "原回答摘要",
	})
	if err != nil {
		t.Fatalf("AppendFeedbackCase failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read feedback case failed: %v", err)
	}

	line := strings.TrimSpace(string(content))
	var rec map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("feedback file should be valid JSONL, got:\n%s\nerr: %v", line, err)
	}
	for key, want := range map[string]string{
		"type":                FeedbackRejected,
		"user_input":          "不对",
		"previous_user_input": "原问题",
		"advice_summary":      "原回答摘要",
	} {
		if got, _ := rec[key].(string); got != want {
			t.Errorf("field %q: want %q, got %q", key, want, got)
		}
	}
	if rec["timestamp"] == "" {
		t.Error("timestamp should be non-empty")
	}
}
