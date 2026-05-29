package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
)

const (
	FeedbackAcceptedOrContinued = "advice_accepted_or_continued"
	FeedbackRejected            = "advice_rejected"
	FeedbackNeedsContext        = "advice_needs_context"
	FeedbackUnrelated           = "unrelated"
)

type AdviceFeedbackState struct {
	LastUserInput string
	LastAdvice    string
	LastIntent    string
	LastFeedback  string
}

type FeedbackMemory struct {
	mu    sync.Mutex
	state AdviceFeedbackState
}

func NewFeedbackMemory() *FeedbackMemory {
	return &FeedbackMemory{}
}

func (m *FeedbackMemory) Detect(userInput string) *contracts.AdviceFeedback {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(m.state.LastUserInput) == "" || strings.TrimSpace(m.state.LastAdvice) == "" {
		return nil
	}

	feedbackType, reason := classifyAdviceFeedback(userInput)
	if feedbackType == FeedbackUnrelated {
		return nil
	}

	return &contracts.AdviceFeedback{
		Type:              feedbackType,
		Reason:            reason,
		PreviousUserInput: m.state.LastUserInput,
		LastAdviceSummary: summarizeForFeedback(m.state.LastAdvice, 120),
	}
}

func (m *FeedbackMemory) Record(userInput string, advice string, intent string) {
	if m == nil || strings.TrimSpace(userInput) == "" || strings.TrimSpace(advice) == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = AdviceFeedbackState{
		LastUserInput: strings.TrimSpace(userInput),
		LastAdvice:    strings.TrimSpace(advice),
		LastIntent:    strings.TrimSpace(intent),
	}
}

func (m *FeedbackMemory) Snapshot() AdviceFeedbackState {
	if m == nil {
		return AdviceFeedbackState{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func classifyAdviceFeedback(userInput string) (string, string) {
	normalized := strings.ToLower(strings.TrimSpace(userInput))
	if normalized == "" {
		return FeedbackUnrelated, ""
	}

	rejectedKeywords := []string{
		"不对", "不是", "答非所问", "没用", "你确定", "确定吗", "幻觉", "当前版本不是",
		"这不对", "错了", "不准确", "不地道", "太勉强", "没有回答", "跑题",
	}
	if feedbackContainsAny(normalized, rejectedKeywords) {
		return FeedbackRejected, "用户明确指出上一轮建议没有命中问题"
	}

	contextKeywords := []string{
		"我现在", "如果我有", "我场上", "我血量", "我等级", "我经济", "我装备",
		"现在是", "目前", "场上有", "血量", "等级", "金币", "阶段",
	}
	if feedbackContainsAny(normalized, contextKeywords) {
		return FeedbackNeedsContext, "用户补充了新的局面上下文"
	}

	continuedKeywords := []string{
		"那", "继续", "这套", "几级", "怎么过渡", "装备怎么给", "怎么站位",
		"什么时候", "能不能", "要不要", "优先", "下一步",
	}
	if feedbackContainsAny(normalized, continuedKeywords) {
		return FeedbackAcceptedOrContinued, "用户沿着上一轮建议继续追问"
	}

	return FeedbackUnrelated, ""
}

func feedbackContainsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func summarizeForFeedback(text string, maxRunes int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if maxRunes <= 0 || utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "..."
}

// feedbackRecord is the persistent schema for a single rejected-advice event.
type feedbackRecord struct {
	Timestamp         string `json:"timestamp"`
	Date              string `json:"date"`
	Type              string `json:"type"`
	UserInput         string `json:"user_input"`
	PreviousUserInput string `json:"previous_user_input,omitempty"`
	AdviceSummary     string `json:"advice_summary,omitempty"`
	Reason            string `json:"reason,omitempty"`
}

// AppendFeedbackCase appends a rejected-advice event to a JSONL file.
// path defaults to "data/feedback_cases.jsonl" — one JSON object per line,
// append-only, safe to tail/grep, multi-instance friendly.
func AppendFeedbackCase(path string, userInput string, advice string, feedback *contracts.AdviceFeedback) error {
	if feedback == nil || feedback.Type != FeedbackRejected {
		return nil
	}
	if path == "" {
		path = filepath.Join("data", "feedback_cases.jsonl")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	rec := feedbackRecord{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Date:              time.Now().Format("2006-01-02"),
		Type:              feedback.Type,
		UserInput:         strings.TrimSpace(userInput),
		PreviousUserInput: feedback.PreviousUserInput,
		AdviceSummary:     feedback.LastAdviceSummary,
		Reason:            feedback.Reason,
	}

	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%s\n", line)
	return err
}
