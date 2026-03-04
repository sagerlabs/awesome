package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/sagerlabs/awesome/tft/data"
)

// Agent TFT Copilot 的对外入口
type Agent struct {
	runnable compose.Runnable[*GraphInput, *schema.Message]
	store    *data.Store
}

// NewAgent 初始化 Agent：创建 LLM + 编译 Graph
func NewAgent(ctx context.Context, store *data.Store) (*Agent, error) {
	chatModel, err := NewChatModel(ctx, DefaultModelConfig())
	if err != nil {
		return nil, fmt.Errorf("init chat model: %w", err)
	}

	runnable, err := BuildGraph(ctx, chatModel, store)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	return &Agent{runnable: runnable, store: store}, nil
}

// Analyze 普通接口：等待 LLM 完整输出后返回
func (a *Agent) Analyze(ctx context.Context, rawInput string) (*GraphOutput, error) {
	msg, err := a.runnable.Invoke(ctx, &GraphInput{RawInput: rawInput})
	if err != nil {
		return nil, fmt.Errorf("graph invoke: %w", err)
	}
	return &GraphOutput{LLMAdvice: msg.Content}, nil
}

// AnalyzeStream 流式接口：用 StreamReaderWithConvert 逐 chunk 转换类型
// Graph 输出 *schema.StreamReader[*schema.Message]
// 在这里转换成 *schema.StreamReader[*GraphOutput]，每个 chunk 单独处理，不聚合
func (a *Agent) AnalyzeStream(ctx context.Context, rawInput string) (
	*schema.StreamReader[*GraphOutput], error,
) {
	// Graph.Stream 返回 *schema.StreamReader[*schema.Message]
	sr, err := a.runnable.Stream(ctx, &GraphInput{RawInput: rawInput})
	if err != nil {
		return nil, fmt.Errorf("graph stream: %w", err)
	}

	// StreamReaderWithConvert：逐 chunk 转换，不等待全部完成
	// 每收到一个 *schema.Message chunk，立刻转成 *GraphOutput 推出去
	converted := schema.StreamReaderWithConvert(sr,
		func(msg *schema.Message) (*GraphOutput, error) {
			if msg == nil || msg.Content == "" {
				// 跳过空 chunk（心跳包、role 字段等）
				return nil, schema.ErrNoValue
			}
			return &GraphOutput{LLMAdvice: msg.Content}, nil
		},
	)

	return converted, nil
}
