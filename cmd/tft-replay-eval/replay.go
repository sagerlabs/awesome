package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// replayCase 是一条待回放的 rejected 样本。
// 字段与 data/feedback_cases.jsonl 的落盘格式一致（见 tft/agent.feedbackRecord），
// 所以 feedback 文件可以直接作为回放输入，不需要转换。
type replayCase struct {
	Timestamp         string `json:"timestamp"`
	UserInput         string `json:"user_input"`          // 用户的否定，如 "不对"
	PreviousUserInput string `json:"previous_user_input"` // 触发坏回答的原问题
	AdviceSummary     string `json:"advice_summary"`      // 当时的坏回答摘要（对比基线）
	Reason            string `json:"reason,omitempty"`
}

// loadCases 逐行解析 JSONL，跳过坏行；与 tft-feedback-report 同策略。
// 缺少 previous_user_input 的样本无法回放（不知道原问题），也跳过。
func loadCases(r io.Reader) (cases []replayCase, skipped int, err error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var c replayCase
		if json.Unmarshal([]byte(line), &c) != nil ||
			c.PreviousUserInput == "" || c.UserInput == "" {
			skipped++
			continue
		}
		cases = append(cases, c)
	}
	return cases, skipped, scanner.Err()
}

// Runner 抽象"对一个会话发一轮问题并拿到完整回答"。
// 回放需要两轮：先重放原问题建立会话状态，再重放用户的否定，
// 评测的是第二轮（Agent 收到负反馈后的纠偏回答）。
type Runner interface {
	Run(ctx context.Context, sessionID string, input string) (string, error)
}

// ── HTTP Runner：打真实服务的 /v1/tft/nlu/stream ────────────────────────────

type httpRunner struct {
	addr   string
	client *http.Client
}

func newHTTPRunner(addr string) *httpRunner {
	return &httpRunner{
		addr:   strings.TrimRight(addr, "/"),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (r *httpRunner) Run(ctx context.Context, sessionID string, input string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"input":      input,
		"session_id": sessionID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.addr+"/v1/tft/nlu/stream", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return collectSSE(resp.Body)
}

// collectSSE 拼接 SSE 流里的 token 块成完整回答。
// 协议与前端 parseSSEChunk 一致：data: {"type":"token","content":...} / done / error。
func collectSSE(r io.Reader) (string, error) {
	var sb strings.Builder
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "" {
			continue
		}
		var chunk struct {
			Type    string `json:"type"`
			Content string `json:"content"`
			Error   string `json:"error"`
		}
		if json.Unmarshal([]byte(raw), &chunk) != nil {
			continue // 心跳/注释行
		}
		switch chunk.Type {
		case "token":
			sb.WriteString(chunk.Content)
		case "error":
			return "", fmt.Errorf("服务端错误: %s", chunk.Error)
		case "done":
			return sb.String(), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// ── Demo Runner：离线演示，不依赖运行中的服务和 LLM ─────────────────────────

// demoRunner 用固定话术模拟一个"已纠偏"的 Agent：第二轮（收到否定后）
// 会承认未命中并给出不同的回答。用于演示评测器的判定逻辑本身。
type demoRunner struct {
	turns map[string]int // sessionID -> 已进行的轮数
}

func newDemoRunner() *demoRunner { return &demoRunner{turns: make(map[string]int)} }

func (d *demoRunner) Run(_ context.Context, sessionID string, input string) (string, error) {
	d.turns[sessionID]++
	if d.turns[sessionID] == 1 {
		return "（demo）首轮回答：针对「" + input + "」的初始建议。", nil
	}
	return "（demo）上一轮可能没命中你的问题，我重新按知识库证据看：" +
		"建议优先成型核心羁绊，7 级时根据牌面决定是否拉 8。下一步：先稳血量再追三星。", nil
}
