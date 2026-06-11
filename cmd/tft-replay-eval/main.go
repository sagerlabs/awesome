// tft-replay-eval 把教练反馈闭环落盘的 rejected 样本（data/feedback_cases.jsonl）
// 重新回放一遍链路，验证修复后的 Agent 在收到负反馈时是否真的纠偏了。
// 这是 ADR-010 Phase 4（Eval 集成）的 demo 实现，设计见 docs/replay-eval-design.md。
//
// 每条样本回放两轮（同一个新 session）：
//
//	第 1 轮：重放原问题 previous_user_input，建立会话状态
//	第 2 轮：重放用户的否定 user_input，评测这一轮的纠偏回答
//
// 用法：
//
//	go run ./cmd/tft-replay-eval -mode demo                       # 离线演示，不依赖服务
//	go run ./cmd/tft-replay-eval -mode http -addr http://localhost:8080
//	go run ./cmd/tft-replay-eval -file path.jsonl -max 20
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	file := flag.String("file", "data/feedback_cases.jsonl", "rejected 样本 JSONL 路径")
	mode := flag.String("mode", "demo", "demo（离线演示）或 http（打真实服务回放）")
	addr := flag.String("addr", "http://localhost:8080", "http 模式下的服务地址")
	maxCases := flag.Int("max", 50, "最多回放多少条样本（控制 LLM 成本）")
	flag.Parse()

	f, err := os.Open(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tft-replay-eval: 打开 %s 失败: %v\n", *file, err)
		fmt.Fprintln(os.Stderr, "提示：先用 -mode demo -file testdata/demo_cases.jsonl 跑离线演示。")
		os.Exit(1)
	}
	cases, skipped, err := loadCases(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tft-replay-eval: 读取失败: %v\n", err)
		os.Exit(1)
	}
	if len(cases) > *maxCases {
		cases = cases[len(cases)-*maxCases:] // 取最新的 N 条
	}

	var runner Runner
	switch *mode {
	case "demo":
		runner = newDemoRunner()
	case "http":
		runner = newHTTPRunner(*addr)
	default:
		fmt.Fprintf(os.Stderr, "tft-replay-eval: 未知 mode %q\n", *mode)
		os.Exit(1)
	}

	verdicts := replayAll(context.Background(), runner, cases)
	fmt.Print(buildReplayReport(verdicts, skipped))
}

// replayAll 顺序回放（不并发：http 模式下避免打爆 LLM 配额和限流中间件）。
func replayAll(ctx context.Context, runner Runner, cases []replayCase) []verdict {
	verdicts := make([]verdict, 0, len(cases))
	for i, c := range cases {
		sessionID := fmt.Sprintf("replay-%d-%d", time.Now().Unix(), i)
		// 第 1 轮只为建立会话状态，回答内容不评测，失败则整条记为回放失败。
		if _, err := runner.Run(ctx, sessionID, c.PreviousUserInput); err != nil {
			verdicts = append(verdicts, judge(c, "", fmt.Errorf("首轮回放失败: %w", err)))
			continue
		}
		answer, err := runner.Run(ctx, sessionID, c.UserInput)
		verdicts = append(verdicts, judge(c, answer, err))
	}
	return verdicts
}
