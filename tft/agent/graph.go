package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/sagerlabs/awesome/tft/prompt"
	"github.com/sagerlabs/awesome/tft/trace"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/tft/data"
)

// ── 节点 key 常量，避免魔法字符串 ─────────────────────────────────────────────

const (
	nodeParser       = "parser"       // InputParser：标准化用户输入
	nodeHeroComps    = "hero_comps"   // Tool1：英雄 → 推荐阵容
	nodeItemFit      = "item_fit"     // Tool2：装备 → 适配阵容
	nodeCompTier     = "comp_tier"    // Tool3：阵容强度查询
	nodeIntersection = "intersection" // 交集计算：三路结果合并
	nodeLLM          = "llm"          // LLM 推理：生成自然语言建议
	nodeFormat       = "format"       // 格式化输出
)

// ── Graph 输入输出类型 ─────────────────────────────────────────────────────────

// GraphInput 整个 Graph 的入口，用户原始输入
type GraphInput struct {
	RawInput string `json:"raw_input"` // 如："兰博 肯能 鬼索的狂暴之刃"
}

// GraphOutput 整个 Graph 的出口，最终推荐结果
type GraphOutput struct {
	Recommendations []data.Recommendation `json:"recommendations"`
	LLMAdvice       string                `json:"llm_advice"` // LLM 生成的自然语言建议
}

// ── 并行 Fan-out 节点的中间状态 ───────────────────────────────────────────────
// Eino Fan-out 要求多个并行节点的输出 merge 时，类型必须是 map[string]any
// 且各节点输出的 key 不能重复，通过 WithOutputKey 指定

// ── BuildGraph 构建并编译 TFT Copilot Graph ───────────────────────────────────

func BuildGraph(ctx context.Context, chatModel model.ChatModel, store *data.Store) (
	compose.Runnable[*GraphInput, *schema.Message], error,
) {
	return BuildGraphWithRuntime(ctx, chatModel, NewDefaultGraphRuntime(store))
}

func BuildGraphWithRuntime(ctx context.Context, chatModel model.ChatModel, runtime *GraphRuntime) (
	compose.Runnable[*GraphInput, *schema.Message], error,
) {
	if err := runtime.Validate(); err != nil {
		return nil, fmt.Errorf("validate graph runtime: %w", err)
	}

	// NewGraph[输入类型, 输出类型]
	g := compose.NewGraph[*GraphInput, *schema.Message]()

	// ── 1. InputParser 节点 ───────────────────────────────────────────────────
	// 把用户原始输入标准化为 UserInput{Heroes, Items}
	parserFn := compose.InvokableLambda(
		func(ctx context.Context, in *GraphInput) (*data.UserInput, error) {
			return runtime.Parser.Parse(in.RawInput)
		},
	)
	if err := g.AddLambdaNode(nodeParser, parserFn); err != nil {
		return nil, fmt.Errorf("add parser node: %w", err)
	}

	// ── 2. 三个并行 Tool 节点 ─────────────────────────────────────────────────
	// Eino Fan-out：一个节点连接多个下游节点，自动并发执行
	// 每个节点通过 WithOutputKey 把输出包装成 map，供 Fan-in 合并

	// Tool1: 英雄 → 推荐阵容
	heroCompsFn := compose.InvokableLambda(
		func(ctx context.Context, in *data.UserInput) (*data.HeroCompsOutput, error) {
			return runtime.Tools.HeroComps.Query(ctx, &data.HeroCompsInput{
				Heroes: in.Heroes,
				TopN:   5,
			})
		},
	)
	if err := g.AddLambdaNode(nodeHeroComps, heroCompsFn,
		compose.WithOutputKey("hero_comps"), // Fan-in 时的 map key
	); err != nil {
		return nil, fmt.Errorf("add hero_comps node: %w", err)
	}

	// Tool2: 装备 → 适配阵容
	itemFitFn := compose.InvokableLambda(
		func(ctx context.Context, in *data.UserInput) (*data.ItemFitOutput, error) {
			return runtime.Tools.ItemFit.Query(ctx, &data.ItemFitInput{
				Items: in.Items,
			})
		},
	)
	if err := g.AddLambdaNode(nodeItemFit, itemFitFn,
		compose.WithOutputKey("item_fit"),
	); err != nil {
		return nil, fmt.Errorf("add item_fit node: %w", err)
	}

	// Tool3: 阵容强度（从 hero_comps 结果中提取 cluster_id 查询）
	// 注意：这里也接收 UserInput，实际查询时会先调 HeroComps 再查 Tier
	// 简化写法：直接查全部 S/A Tier 作为参考基准
	compTierFn := compose.InvokableLambda(
		func(ctx context.Context, in *data.UserInput) (*data.CompTierOutput, error) {
			return runtime.Tools.CompTier.QueryTopTier(ctx)
		},
	)
	if err := g.AddLambdaNode(nodeCompTier, compTierFn,
		compose.WithOutputKey("comp_tier"),
	); err != nil {
		return nil, fmt.Errorf("add comp_tier node: %w", err)
	}

	// ── 3. 交集计算节点 ───────────────────────────────────────────────────────
	// Fan-in：接收三个并行节点的 map 合并结果
	// 输入类型：map[string]any（Eino Fan-in 的固定格式）
	intersectFn := compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) (*data.IntersectionOutput, error) {
			// 从 map 中取出三路结果
			heroComps, _ := in["hero_comps"].(*data.HeroCompsOutput)
			itemFit, _ := in["item_fit"].(*data.ItemFitOutput)
			compTier, _ := in["comp_tier"].(*data.CompTierOutput)

			if heroComps == nil {
				heroComps = &data.HeroCompsOutput{}
			}
			if itemFit == nil {
				itemFit = &data.ItemFitOutput{}
			}
			if compTier == nil {
				compTier = &data.CompTierOutput{}
			}

			return runtime.Tools.Intersection.Compute(&data.IntersectionInput{
				HeroComps: *heroComps,
				ItemFits:  *itemFit,
				CompTiers: *compTier,
			})
		},
	)
	if err := g.AddLambdaNode(nodeIntersection, intersectFn); err != nil {
		return nil, fmt.Errorf("add intersection node: %w", err)
	}

	// ── 4. LLM 推理节点 ───────────────────────────────────────────────────────
	// 把交集结果转成 []*schema.Message 喂给 ChatModel
	llmInputFn := compose.InvokableLambda(
		func(ctx context.Context, in *data.IntersectionOutput) ([]*schema.Message, error) {
			prompt := runtime.PromptBuilder.BuildRecommendationPrompt(in)
			return []*schema.Message{
				schema.SystemMessage(systemPrompt),
				schema.UserMessage(prompt),
			}, nil
		},
	)
	if err := g.AddLambdaNode("llm_input", llmInputFn); err != nil {
		return nil, fmt.Errorf("add llm_input node: %w", err)
	}

	if err := g.AddChatModelNode(nodeLLM, chatModel); err != nil {
		return nil, fmt.Errorf("add llm node: %w", err)
	}

	// ── 5. format 节点已移除 ──────────────────────────────────────────────────
	// StreamableLambda 的 input 是普通类型 T，不是 StreamReader[T]
	// 无法在 Graph 节点内做流式透传转换
	// 解法：LLM 节点直接连 END，输出 *schema.Message
	// 在 agent.go 的 AnalyzeStream 中用 schema.StreamReaderWithConvert 做逐 chunk 类型转换

	// ── 6. 连接边（定义数据流向）─────────────────────────────────────────────

	// START -> parser
	if err := g.AddEdge(compose.START, nodeParser); err != nil {
		return nil, fmt.Errorf("edge START->parser: %w", err)
	}

	// parser -> 三个并行 Tool（Fan-out，Eino 自动并发）
	if err := g.AddEdge(nodeParser, nodeHeroComps); err != nil {
		return nil, fmt.Errorf("edge parser->hero_comps: %w", err)
	}
	if err := g.AddEdge(nodeParser, nodeItemFit); err != nil {
		return nil, fmt.Errorf("edge parser->item_fit: %w", err)
	}
	if err := g.AddEdge(nodeParser, nodeCompTier); err != nil {
		return nil, fmt.Errorf("edge parser->comp_tier: %w", err)
	}

	// 三个并行 Tool -> intersection（Fan-in，Eino 自动等待所有上游完成后合并）
	if err := g.AddEdge(nodeHeroComps, nodeIntersection); err != nil {
		return nil, fmt.Errorf("edge hero_comps->intersection: %w", err)
	}
	if err := g.AddEdge(nodeItemFit, nodeIntersection); err != nil {
		return nil, fmt.Errorf("edge item_fit->intersection: %w", err)
	}
	if err := g.AddEdge(nodeCompTier, nodeIntersection); err != nil {
		return nil, fmt.Errorf("edge comp_tier->intersection: %w", err)
	}

	// intersection -> llm_input -> llm -> format -> END
	if err := g.AddEdge(nodeIntersection, "llm_input"); err != nil {
		return nil, fmt.Errorf("edge intersection->llm_input: %w", err)
	}
	if err := g.AddEdge("llm_input", nodeLLM); err != nil {
		return nil, fmt.Errorf("edge llm_input->llm: %w", err)
	}
	// llm 直接连 END，输出 *schema.Message
	// 类型转换在 agent.go 的 AnalyzeStream 里完成
	if err := g.AddEdge(nodeLLM, compose.END); err != nil {
		return nil, fmt.Errorf("edge llm->END: %w", err)
	}

	// ── 7. 编译 Graph ─────────────────────────────────────────────────────────
	runnable, err := g.Compile(ctx,
		compose.WithGraphName("TFTCopilotGraph"),
	)
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	return runnable, nil
}

func BuildNluGraph(ctx context.Context, chatModel model.ChatModel, knowledgeAdapter *KnowledgeAdapter) (
	compose.Runnable[*NluContext, *NluEnrichedContext], error,
) {
	g := compose.NewGraph[*NluContext, *NluEnrichedContext]()
	fastNLU := NewFastNLUExtractor(knowledgeAdapter)

	if err := g.AddLambdaNode("nlu_extract", newNluExtractLambda(chatModel, fastNLU)); err != nil {
		return nil, fmt.Errorf("add nlu_extract node: %w", err)
	}

	if err := g.AddLambdaNode("data_lookup", newDataLookupLambda(knowledgeAdapter)); err != nil {
		return nil, fmt.Errorf("add data_lookup node: %w", err)
	}

	// Connect edges
	if err := g.AddEdge(compose.START, "nlu_extract"); err != nil {
		return nil, fmt.Errorf("edge START->nlu_extract: %w", err)
	}
	if err := g.AddEdge("nlu_extract", "data_lookup"); err != nil {
		return nil, fmt.Errorf("edge nlu_extract->data_lookup: %w", err)
	}
	if err := g.AddEdge("data_lookup", compose.END); err != nil {
		return nil, fmt.Errorf("edge data_lookup->END: %w", err)
	}

	runnable, err := g.Compile(ctx,
		compose.WithGraphName("NLUExtractGraph"),
	)
	if err != nil {
		return nil, fmt.Errorf("compile nlu graph: %w", err)
	}

	return runnable, nil
}

// BuildNluStreamGraph 构建支持流式输出的NLU Graph，包含LLM润色节点
func BuildNluStreamGraph(ctx context.Context, chatModel model.ChatModel, knowledgeAdapter *KnowledgeAdapter) (
	compose.Runnable[*NluContext, *schema.Message], error,
) {
	g := compose.NewGraph[*NluContext, *schema.Message]()
	fastNLU := NewFastNLUExtractor(knowledgeAdapter)

	if err := g.AddLambdaNode("nlu_extract", newNluExtractLambda(chatModel, fastNLU)); err != nil {
		return nil, fmt.Errorf("add nlu_extract node: %w", err)
	}

	if err := g.AddLambdaNode("data_lookup", newDataLookupLambda(knowledgeAdapter)); err != nil {
		return nil, fmt.Errorf("add data_lookup node: %w", err)
	}

	// Node 3: 构建LLM润色prompt
	llmInput := compose.InvokableLambda(func(ctx context.Context, input *NluEnrichedContext) ([]*schema.Message, error) {
		nluPrompt, _ := BuildNluFormatPrompt(input)
		return []*schema.Message{
			schema.SystemMessage(FormatSystemPrompt),
			schema.UserMessage(nluPrompt),
		}, nil
	})
	if err := g.AddLambdaNode("llm_input", llmInput); err != nil {
		return nil, fmt.Errorf("add llm_input node: %w", err)
	}

	// Node 4: LLM润色
	if err := g.AddChatModelNode("llm_refine", chatModel); err != nil {
		return nil, fmt.Errorf("add llm_refine node: %w", err)
	}

	// Connect edges
	if err := g.AddEdge(compose.START, "nlu_extract"); err != nil {
		return nil, fmt.Errorf("edge START->nlu_extract: %w", err)
	}
	if err := g.AddEdge("nlu_extract", "data_lookup"); err != nil {
		return nil, fmt.Errorf("edge nlu_extract->data_lookup: %w", err)
	}
	if err := g.AddEdge("data_lookup", "llm_input"); err != nil {
		return nil, fmt.Errorf("edge data_lookup->llm_input: %w", err)
	}
	if err := g.AddEdge("llm_input", "llm_refine"); err != nil {
		return nil, fmt.Errorf("edge llm_input->llm_refine: %w", err)
	}
	if err := g.AddEdge("llm_refine", compose.END); err != nil {
		return nil, fmt.Errorf("edge llm_refine->END: %w", err)
	}

	runnable, err := g.Compile(ctx,
		compose.WithGraphName("NLUStreamGraph"),
	)
	if err != nil {
		return nil, fmt.Errorf("compile nlu stream graph: %w", err)
	}

	return runnable, nil
}

func newNluExtractLambda(chatModel model.ChatModel, fastNLU *FastNLUExtractor) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *NluContext) (output *NluContext, err error) {
		start := time.Now()
		traceID, _ := trace.TraceIDFromContext(ctx)
		if c, ok := fastNLU.TryParse(input.UserInput); ok {
			input.Ctx = c
			input.FastNLUHit = true
			input.NLUProvider = "fast"
			logNluExtract(traceID, input, time.Since(start))
			return input, nil
		}
		fullPrompt, err := prompt.BuildNLUPrompt(input.UserInput)
		if err != nil {
			return nil, fmt.Errorf("build nlu prompt: %w", err)
		}
		resp, err := chatModel.Generate(ctx, []*schema.Message{
			schema.UserMessage(fullPrompt),
		})
		if err != nil {
			return nil, fmt.Errorf("generate: %w", err)
		}

		var c Context
		content := extractJSONObject(resp.Content)
		if err := json.Unmarshal([]byte(content), &c); err != nil {
			logrus.WithError(err).
				WithField("raw_content", resp.Content).
				WithField("extracted_content", content).
				Warn("JSON解析失败，使用空Context")
		}
		input.Ctx = c
		input.NLUProvider = "llm"
		input.NLUCalls = 1
		logNluExtract(traceID, input, time.Since(start))
		return input, nil
	})
}

func newDataLookupLambda(knowledgeAdapter *KnowledgeAdapter) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *NluContext) (output *NluEnrichedContext, err error) {
		start := time.Now()
		traceID, _ := trace.TraceIDFromContext(ctx)
		result, err := knowledgeAdapter.QueryNLU(input.Ctx)
		if err != nil {
			return nil, fmt.Errorf("knowledge query nlu: %w", err)
		}
		result.UserInput = input.UserInput
		result.Feedback = input.Feedback
		logKnowledgeLookup(traceID, input, result, time.Since(start))
		return result, nil
	})
}

func extractJSONObject(content string) string {
	if startIdx := strings.Index(content, "{"); startIdx != -1 {
		if endIdx := strings.LastIndex(content, "}"); endIdx != -1 && endIdx > startIdx {
			return content[startIdx : endIdx+1]
		}
	}
	return content
}

func logNluExtract(traceID string, input *NluContext, elapsed time.Duration) {
	fields := logrus.Fields{
		"trace_id":     traceID,
		"input":        input.UserInput,
		"elapsed_ms":   elapsed.Milliseconds(),
		"fast_nlu_hit": input.FastNLUHit,
		"nlu_provider": input.NLUProvider,
		"llm_calls":    input.NLUCalls,
		"intent":       input.Ctx.Intent,
		"champions":    len(input.Ctx.Champions),
		"items":        len(input.Ctx.Items),
		"traits":       len(input.Ctx.Traits),
		"role_query":   input.Ctx.RoleQuery,
		"playstyle":    input.Ctx.Playstyle,
	}
	if input.Ctx.UnitCost != nil {
		fields["unit_cost"] = *input.Ctx.UnitCost
	}
	if input.Ctx.GameStage != nil {
		fields["game_stage"] = *input.Ctx.GameStage
	}
	logrus.WithFields(fields).Info("NLU提取完成")
	if jsonBytes, err := json.Marshal(input.Ctx); err == nil {
		logrus.WithField("trace_id", traceID).WithField("ctx", string(jsonBytes)).Debug("NLU结构化结果")
	}
}

func logKnowledgeLookup(traceID string, input *NluContext, result *NluEnrichedContext, elapsed time.Duration) {
	logrus.WithFields(logrus.Fields{
		"trace_id":          traceID,
		"elapsed_ms":        elapsed.Milliseconds(),
		"fast_nlu_hit":      input.FastNLUHit,
		"nlu_provider":      input.NLUProvider,
		"llm_calls":         input.NLUCalls,
		"intent":            result.Ctx.Intent,
		"matched_comps":     len(result.MatchedComps),
		"matched_items":     len(result.MatchedItems),
		"matched_champions": len(result.MatchedChampions),
		"matched_traits":    len(result.MatchedTraits),
		"patch_notes":       len(result.PatchNotes),
	}).Info("knowledge查询完成")
}
