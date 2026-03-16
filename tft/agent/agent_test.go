package agent_test

// 纯 LLM 调用延迟测试 —— 直接打 LLM，不走 Graph/Store/Mock
//
// 用法：
//   LLM_PROVIDER=deepseek OPENAI_API_KEY=sk-xxx \
//     go test ./tft/agent/ -run TestLLMRaw -v -timeout 60s
//
//   # 连续5次采样看 P50/P95
//   LLM_PROVIDER=deepseek OPENAI_API_KEY=sk-xxx \
//     go test ./tft/agent/ -run TestLLMRaw_Benchmark -v -timeout 120s

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/sagerlabs/awesome/tft/agent"
)

func skipIfNoLLM(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("ARK_API_KEY") == "" {
		t.Skip("未设置 OPENAI_API_KEY / ARK_API_KEY，跳过")
	}
}

// ── 单次调用：非流式 ──────────────────────────────────────────────────────────

func TestLLMRaw_Invoke(t *testing.T) {
	skipIfNoLLM(t)
	ctx := context.Background()

	chatModel, err := agent.NewChatModel(ctx, agent.DefaultModelConfig())
	if err != nil {
		t.Fatalf("初始化 ChatModel 失败: %v", err)
	}

	cases := []struct {
		name   string
		prompt string
	}{
		{"极短prompt", "走兰博阵容，缺波比，建议？20字内。"},
		{"中等prompt", "棋盘：兰博肯能璐璐，走约德尔（S），缺波比/三泽/海默丁格，兰博装古神狂暴，8费升。建议？20字内。"},
		{"完整prompt", "棋盘：兰博肯能璐璐古神狂暴之刃，推荐走约德尔机甲（S）。已有兰博/肯能/璐璐。还差波比/三泽/海默丁格。装备古神狂暴之刃给兰博。升级节点8费。备选星之守护者。给出20字内运营建议。"},
	}

	fmt.Printf("\n===== 非流式（Invoke）延迟 | provider=%s model=%s =====\n",
		os.Getenv("LLM_PROVIDER"), os.Getenv("OPENAI_MODEL"))
	fmt.Printf("%-12s  %-8s  %-6s  %s\n", "用例", "耗时", "字符", "输出")
	fmt.Println(strings.Repeat("-", 65))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msgs := []*schema.Message{
				schema.SystemMessage("TFT教练，20字内，格式：买X，装备Y给Z，N费升。"),
				schema.UserMessage(tc.prompt),
			}

			start := time.Now()
			resp, err := chatModel.Generate(ctx, msgs)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("Generate 失败: %v", err)
				return
			}

			content := resp.Content
			fmt.Printf("%-12s  %-8s  %-6d  %s\n",
				tc.name,
				elapsed.Round(time.Millisecond),
				len([]rune(content)),
				content,
			)

			if elapsed > 10*time.Second {
				t.Logf("⚠️  耗时 %s 超过 10s", elapsed)
			}
		})
	}
}

// ── 单次调用：流式（TTFT 最关键）────────────────────────────────────────────

func TestLLMRaw_Stream(t *testing.T) {
	skipIfNoLLM(t)
	ctx := context.Background()

	chatModel, err := agent.NewChatModel(ctx, agent.DefaultModelConfig())
	if err != nil {
		t.Fatalf("初始化 ChatModel 失败: %v", err)
	}

	msgs := []*schema.Message{
		schema.SystemMessage("You must respond directly. Never think. Never use <think> tags. /no_think"),
		schema.SystemMessage("TFT教练，20字内，格式：买X，装备Y给Z，N费升。"),
		schema.UserMessage("棋盘兰博肯能璐璐古神狂暴，走约德尔，缺波比，兰博装古神狂暴，8费升。建议？"),
	}

	fmt.Printf("\n===== 流式（Stream）TTFT | provider=%s model=%s =====\n",
		os.Getenv("LLM_PROVIDER"), os.Getenv("OPENAI_MODEL"))

	msg, err := chatModel.Generate(ctx, msgs)
	if err != nil {
		return
	}
	t.Log(msg.Content)
	t.Logf("%+v", msg)
	start := time.Now()
	sr, err := chatModel.Stream(ctx, msgs)
	if err != nil {
		t.Fatalf("Stream 失败: %v", err)
	}
	defer sr.Close()

	var (
		ttft       time.Duration
		firstChunk = true
		chunkCount = 0
		sb         strings.Builder
	)

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv 失败: %v", err)
		}
		if msg == nil || msg.Content == "" {
			continue
		}

		chunkCount++
		sb.WriteString(msg.Content)

		if firstChunk {
			ttft = time.Since(start)
			firstChunk = false
			fmt.Printf("  第一个 chunk 到达（TTFT）: %s\n", ttft.Round(time.Millisecond))
			fmt.Printf("  第一个 chunk 内容: %q\n", msg.Content)
		}
	}

	total := time.Since(start)

	fmt.Printf("  总耗时:   %s\n", total.Round(time.Millisecond))
	fmt.Printf("  chunk数:  %d\n", chunkCount)
	fmt.Printf("  总字符:   %d\n", len([]rune(sb.String())))
	fmt.Printf("  完整输出: %s\n", sb.String())

	if ttft > 5*time.Second {
		t.Logf("⚠️  TTFT %s 过高，建议：换模型 or 减少 LLM_MAX_TOKENS", ttft)
	}
}

// ── 多次采样：P50 / P95 ───────────────────────────────────────────────────────

func TestLLMRaw_Benchmark(t *testing.T) {
	skipIfNoLLM(t)

	const runs = 5

	ctx := context.Background()
	chatModel, err := agent.NewChatModel(ctx, agent.DefaultModelConfig())
	if err != nil {
		t.Fatalf("初始化 ChatModel 失败: %v", err)
	}

	msgs := []*schema.Message{
		schema.SystemMessage("TFT教练，20字内，格式：买X，装备Y给Z，N费升。"),
		schema.UserMessage("棋盘兰博肯能璐璐古神狂暴，走约德尔，缺波比，8费升。建议？"),
	}

	fmt.Printf("\n===== 连续 %d 次采样 | provider=%s model=%s =====\n",
		runs, os.Getenv("LLM_PROVIDER"), os.Getenv("OPENAI_MODEL"))
	fmt.Printf("%-6s  %-10s  %-10s  %s\n", "序号", "TTFT", "总耗时", "输出")
	fmt.Println(strings.Repeat("-", 55))

	ttfts := make([]time.Duration, 0, runs)
	totals := make([]time.Duration, 0, runs)

	for i := 1; i <= runs; i++ {
		start := time.Now()

		sr, err := chatModel.Stream(ctx, msgs)
		if err != nil {
			t.Logf("第 %d 次 Stream 失败: %v", i, err)
			continue
		}

		var (
			ttft       time.Duration
			firstChunk = true
			sb         strings.Builder
		)

		for {
			msg, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Logf("第 %d 次 Recv 失败: %v", i, err)
				break
			}
			if msg == nil || msg.Content == "" {
				continue
			}
			sb.WriteString(msg.Content)
			if firstChunk {
				ttft = time.Since(start)
				firstChunk = false
			}
		}
		sr.Close()

		total := time.Since(start)
		ttfts = append(ttfts, ttft)
		totals = append(totals, total)

		output := sb.String()
		if len([]rune(output)) > 20 {
			output = string([]rune(output)[:20]) + "..."
		}

		fmt.Printf("%-6d  %-10s  %-10s  %s\n", i,
			ttft.Round(time.Millisecond),
			total.Round(time.Millisecond),
			output,
		)

		if i < runs {
			time.Sleep(300 * time.Millisecond) // 避免限流
		}
	}

	// 统计
	if len(ttfts) >= 2 {
		fmt.Println(strings.Repeat("-", 55))
		fmt.Printf("TTFT   avg=%-8s  p50=%-8s  p95=%s\n",
			avg(ttfts).Round(time.Millisecond),
			percentile(ttfts, 50).Round(time.Millisecond),
			percentile(ttfts, 95).Round(time.Millisecond),
		)
		fmt.Printf("总耗时 avg=%-8s  p50=%-8s  p95=%s\n",
			avg(totals).Round(time.Millisecond),
			percentile(totals, 50).Round(time.Millisecond),
			percentile(totals, 95).Round(time.Millisecond),
		)
	}
}

// ── 统计工具 ──────────────────────────────────────────────────────────────────

func avg(data []time.Duration) time.Duration {
	if len(data) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range data {
		sum += d
	}
	return sum / time.Duration(len(data))
}

func percentile(data []time.Duration, p int) time.Duration {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(data))
	copy(sorted, data)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	idx := p * len(sorted) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
