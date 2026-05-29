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

// defaultSessionTTL bounds how long a conversation's last-turn state is kept.
// A turn older than this is treated as a fresh conversation, and the entry is
// evicted to keep memory bounded for a long-running server.
const defaultSessionTTL = 30 * time.Minute

// AdviceFeedbackState is the last-turn memory for a single conversation.
type AdviceFeedbackState struct {
	LastUserInput string
	LastAdvice    string
	LastIntent    string
	LastFeedback  string
}

type sessionEntry struct {
	state     AdviceFeedbackState
	updatedAt time.Time
}

// FeedbackMemory holds last-turn advice state keyed by session ID, so that
// concurrent conversations never read each other's context. Requests without a
// session ID are treated as stateless: Detect returns nil and Record is a
// no-op, which is the safe default (no cross-talk) when the caller does not
// identify a conversation.
type FeedbackMemory struct {
	mu       sync.Mutex
	sessions map[string]*sessionEntry
	ttl      time.Duration
	now      func() time.Time // injectable so tests can control expiry
}

func NewFeedbackMemory() *FeedbackMemory {
	return &FeedbackMemory{
		sessions: make(map[string]*sessionEntry),
		ttl:      defaultSessionTTL,
		now:      time.Now,
	}
}

// Detect compares the new input against the session's previous turn and returns
// a feedback signal, or nil when there is no prior turn or no session scope.
func (m *FeedbackMemory) Detect(sessionID string, userInput string) *contracts.AdviceFeedback {
	if m == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry := m.liveEntryLocked(sessionID)
	if entry == nil {
		return nil
	}
	if strings.TrimSpace(entry.state.LastUserInput) == "" || strings.TrimSpace(entry.state.LastAdvice) == "" {
		return nil
	}

	feedbackType, reason := classifyAdviceFeedback(userInput)
	if feedbackType == FeedbackUnrelated {
		return nil
	}

	return &contracts.AdviceFeedback{
		Type:              feedbackType,
		Reason:            reason,
		PreviousUserInput: entry.state.LastUserInput,
		LastAdviceSummary: summarizeForFeedback(entry.state.LastAdvice, 120),
	}
}

// Record stores the latest turn for a session. It no-ops when the session ID,
// input, or advice is empty.
func (m *FeedbackMemory) Record(sessionID string, userInput string, advice string, intent string) {
	if m == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	if strings.TrimSpace(userInput) == "" || strings.TrimSpace(advice) == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.evictExpiredLocked()
	m.sessions[sessionID] = &sessionEntry{
		state: AdviceFeedbackState{
			LastUserInput: strings.TrimSpace(userInput),
			LastAdvice:    strings.TrimSpace(advice),
			LastIntent:    strings.TrimSpace(intent),
		},
		updatedAt: m.now(),
	}
}

// Snapshot returns a copy of a session's current state, for tests and debugging.
func (m *FeedbackMemory) Snapshot(sessionID string) AdviceFeedbackState {
	if m == nil {
		return AdviceFeedbackState{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry := m.liveEntryLocked(sessionID); entry != nil {
		return entry.state
	}
	return AdviceFeedbackState{}
}

// liveEntryLocked returns a non-expired entry for the session, deleting it if
// it has aged past the TTL. Callers must hold m.mu.
func (m *FeedbackMemory) liveEntryLocked(sessionID string) *sessionEntry {
	entry, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	if m.now().Sub(entry.updatedAt) > m.ttl {
		delete(m.sessions, sessionID)
		return nil
	}
	return entry
}

// evictExpiredLocked drops all aged-out sessions. Callers must hold m.mu.
// Eviction runs on each Record, which is cheap for the expected session count.
func (m *FeedbackMemory) evictExpiredLocked() {
	cutoff := m.now()
	for id, entry := range m.sessions {
		if cutoff.Sub(entry.updatedAt) > m.ttl {
			delete(m.sessions, id)
		}
	}
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
