package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	if feedback := memory.Detect("不对"); feedback != nil {
		t.Fatalf("expected no feedback before previous advice, got %#v", feedback)
	}

	memory.Record("剑魔打工强吗", "能拿来打工，但不要追三。", "champion_query")
	feedback := memory.Detect("不对，这个答非所问")

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
	path := filepath.Join(dir, "FEEDBACK_CASES.md")

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
	for _, text := range []string{"用户原问题：原问题", "Agent 原回答摘要：原回答摘要", "用户反馈：不对"} {
		if !strings.Contains(string(content), text) {
			t.Fatalf("feedback file should contain %q, got:\n%s", text, string(content))
		}
	}
}
