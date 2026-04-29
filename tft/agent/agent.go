package agent

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/trace"
	"github.com/sirupsen/logrus"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// AgentConfig Agent 运行时配置，通过 NewAgentWithConfig 传入
type AgentConfig struct {
	// LLMTimeout LLM 推理超时，0 表示用环境变量 LLM_TIMEOUT_SEC（默认 60s）
	LLMTimeout time.Duration
	// Logger 业务日志，传 nil 时用 logrus.StandardLogger()
	Logger *logrus.Logger
	// EnableTrace 是否开启节点级链路追踪日志（Debug 级别）
	EnableTrace bool
	// GraphRuntime 注入静态 Graph 的可变依赖，nil 时使用默认 runtime
	GraphRuntime *GraphRuntime
}

// defaultLLMTimeout 从环境变量读超时，兜底 60s
func defaultLLMTimeout() time.Duration {
	if v := os.Getenv("LLM_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}

// Agent TFT Copilot 的对外入口
type Agent struct {
	runnable          compose.Runnable[*GraphInput, *schema.Message]
	nluRunnable       compose.Runnable[*NluContext, *NluEnrichedContext]
	nluStreamRunnable compose.Runnable[*NluContext, *schema.Message]
	store             *data.Store
	llmTimeout        time.Duration
	logger            *logrus.Logger
	traceOpts         []compose.Option // 链路追踪 callback，每次调用时注入
	graphRuntime      *GraphRuntime
}

// NewAgent 使用默认配置初始化 Agent
func NewAgent(ctx context.Context, store *data.Store) (*Agent, error) {
	return NewAgentWithConfig(ctx, store, &AgentConfig{})
}

// NewAgentWithConfig 使用自定义配置初始化 Agent
//
// 示例：
//
//	agent.NewAgentWithConfig(ctx, store, &agent.AgentConfig{
//	    LLMTimeout:  30 * time.Second,
//	    EnableTrace: true,
//	    Logger:      logger,
//	})
func NewAgentWithConfig(ctx context.Context, store *data.Store, cfg *AgentConfig) (*Agent, error) {
	if cfg == nil {
		cfg = &AgentConfig{}
	}

	logger := cfg.Logger
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// ── LLM 超时 ─────────────────────────────────────────────────────
	llmTimeout := cfg.LLMTimeout
	if llmTimeout <= 0 {
		llmTimeout = defaultLLMTimeout()
	}
	logger.WithField("llm_timeout", llmTimeout.String()).Info("Agent 配置")

	// ── 初始化 ChatModel ──────────────────────────────────────────────
	chatModel, err := NewChatModel(ctx, DefaultModelConfig())
	if err != nil {
		return nil, fmt.Errorf("init chat model: %w", err)
	}

	// ── 注册链路追踪 Callback ─────────────────────────────────────────
	// Callback 通过 compose.WithCallbacks 在每次 Invoke/Stream 时传入
	var traceOpts []compose.Option
	if cfg.EnableTrace {
		tracer := NewTraceCallback(logger)
		traceOpts = append(traceOpts, compose.WithCallbacks(tracer))
		logger.Info("链路追踪已开启（节点级耗时日志）")
	}

	graphRuntime := cfg.GraphRuntime
	if graphRuntime == nil {
		graphRuntime = NewDefaultGraphRuntime(store)
	}

	// ── 编译 Graph ────────────────────────────────────────────────────
	runnable, err := BuildGraphWithRuntime(ctx, chatModel, graphRuntime)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	knowledgeAdapter, err := newKnowledgeAdapterFromStore(store, logger)
	if err != nil {
		return nil, fmt.Errorf("init knowledge adapter: %w", err)
	}

	nluRunnable, err := BuildNluGraph(ctx, chatModel, knowledgeAdapter)
	if err != nil {
		return nil, fmt.Errorf("build nlu graph: %w", err)
	}

	nluStreamRunnable, err := BuildNluStreamGraph(ctx, chatModel, knowledgeAdapter)
	if err != nil {
		return nil, fmt.Errorf("build nlu stream graph: %w", err)
	}

	return &Agent{
		nluRunnable:       nluRunnable,
		nluStreamRunnable: nluStreamRunnable,
		runnable:          runnable,
		store:             store,
		llmTimeout:        llmTimeout,
		logger:            logger,
		traceOpts:         traceOpts,
		graphRuntime:      graphRuntime,
	}, nil
}

// maxTokens 返回每次调用的 token 上限
// 优先读环境变量 LLM_MAX_TOKENS，兜底 60
func (a *Agent) maxTokens() int {
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 150 {
			return n
		}
	}
	return 1024
}

// withLLMTimeout 在 ctx 上套一层 LLM 专属超时
// 和外层 handler 的 context 独立，防止 handler 超时影响 LLM 正在输出的内容
func (a *Agent) withLLMTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, a.llmTimeout)
}

// Analyze 普通接口：等待 LLM 完整输出后返回
func (a *Agent) Analyze(ctx context.Context, rawInput string) (*GraphOutput, error) {
	traceID, _ := trace.TraceIDFromContext(ctx)
	llmCtx, cancel := a.withLLMTimeout(ctx)
	defer cancel()

	start := time.Now()
	a.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
		"input":    rawInput,
	}).Debug("开始推理")

	recommendations, err := a.computeRecommendations(ctx, rawInput)
	if err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"trace_id": traceID,
			"elapsed":  time.Since(start).String(),
		}).Error("结构化推荐计算失败")
		return nil, fmt.Errorf("compute recommendations: %w", err)
	}

	opts := append(a.traceOpts, compose.WithChatModelOption(model.WithMaxTokens(a.maxTokens())))
	msg, err := a.runnable.Invoke(llmCtx, &GraphInput{RawInput: rawInput}, opts...)
	if err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"trace_id": traceID,
			"elapsed":  time.Since(start).String(),
		}).Error("推理失败")
		return nil, fmt.Errorf("graph invoke: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"trace_id":     traceID,
		"elapsed":      time.Since(start).Round(time.Millisecond).String(),
		"output_chars": len([]rune(msg.Content)),
	}).Debug("推理完成")

	return &GraphOutput{
		Recommendations: recommendations,
		LLMAdvice:       msg.Content,
	}, nil
}

// AnalyzeStream 流式接口：返回 StreamReader，由 handler 逐 chunk 推送
func (a *Agent) AnalyzeStream(ctx context.Context, rawInput string) (
	*schema.StreamReader[*GraphOutput], error,
) {
	llmCtx, cancel := a.withLLMTimeout(ctx)

	start := time.Now()

	//opts := append(a.traceOpts, compose.WithChatModelOption(model.WithMaxTokens(a.maxTokens())))
	opts := append(a.traceOpts, compose.WithChatModelOption(
		ark.WithThinking(&ark.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		})))
	sr, err := a.runnable.Stream(llmCtx, &GraphInput{RawInput: rawInput}, opts...)
	if err != nil {
		cancel()
		a.logger.WithError(err).WithField("elapsed", time.Since(start).String()).Error("流式推理启动失败")
		return nil, fmt.Errorf("graph stream: %w", err)
	}

	// 把 *schema.Message 流转换成 *GraphOutput 流，同时在流结束时打印总耗时
	tokenCount := 0
	converted := schema.StreamReaderWithConvert(sr,
		func(msg *schema.Message) (*GraphOutput, error) {
			if msg == nil || msg.Content == "" {
				return nil, schema.ErrNoValue
			}
			tokenCount++
			return &GraphOutput{LLMAdvice: msg.Content}, nil
		},
	)

	_ = cancel
	return converted, nil
}

// NluAnalyze NLU分析接口：提取用户输入的结构化信息并查询数据
func (a *Agent) NluAnalyze(ctx context.Context, rawInput string) (
	*NluEnrichedContext, error,
) {
	traceID, _ := trace.TraceIDFromContext(ctx)
	llmCtx, cancel := a.withLLMTimeout(ctx)
	defer cancel()

	start := time.Now()

	opts := append(a.traceOpts, compose.WithChatModelOption(
		ark.WithThinking(&ark.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		})))
	result, err := a.nluRunnable.Invoke(llmCtx, &NluContext{UserInput: rawInput}, opts...)
	if err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"trace_id": traceID,
			"elapsed":  time.Since(start).String(),
		}).Error("NLU分析失败")
		return nil, fmt.Errorf("nlu analyze: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"trace_id":      traceID,
		"elapsed":       time.Since(start).Round(time.Millisecond).String(),
		"intent":        result.Ctx.Intent,
		"matched_comps": len(result.MatchedComps),
		"matched_items": len(result.MatchedItems),
	}).Debug("NLU分析完成")

	return result, nil
}

// NluAnalyzeStream NLU流式分析接口：提取用户输入的结构化信息并查询数据，然后流式返回LLM润色结果
func (a *Agent) NluAnalyzeStream(ctx context.Context, rawInput string) (
	*schema.StreamReader[*GraphOutput], error,
) {
	llmCtx, cancel := a.withLLMTimeout(ctx)

	start := time.Now()

	opts := append(a.traceOpts, compose.WithChatModelOption(
		ark.WithThinking(&ark.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		})))
	sr, err := a.nluStreamRunnable.Stream(llmCtx, &NluContext{UserInput: rawInput}, opts...)
	if err != nil {
		cancel()
		a.logger.WithError(err).WithField("elapsed", time.Since(start).String()).Error("NLU流式推理启动失败")
		return nil, fmt.Errorf("nlu stream graph: %w", err)
	}

	// 把 *schema.Message 流转换成 *GraphOutput 流
	tokenCount := 0
	converted := schema.StreamReaderWithConvert(sr,
		func(msg *schema.Message) (*GraphOutput, error) {
			if msg == nil || msg.Content == "" {
				return nil, schema.ErrNoValue
			}
			tokenCount++
			return &GraphOutput{LLMAdvice: msg.Content}, nil
		},
	)

	_ = cancel
	return converted, nil
}

func (a *Agent) computeRecommendations(ctx context.Context, rawInput string) ([]data.Recommendation, error) {
	out, err := a.graphRuntime.Executor.Execute(ctx, rawInput)
	if err != nil {
		return nil, err
	}

	return out.Recommendations, nil
}
