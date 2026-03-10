package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/sagerlabs/awesome/tft/parser"
	"github.com/sagerlabs/awesome/tft/prompt"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/tool"
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
	// NewGraph[输入类型, 输出类型]
	g := compose.NewGraph[*GraphInput, *schema.Message]()

	// ── 1. InputParser 节点 ───────────────────────────────────────────────────
	// 把用户原始输入标准化为 UserInput{Heroes, Items}
	parserFn := compose.InvokableLambda(
		func(ctx context.Context, in *GraphInput) (*data.UserInput, error) {
			inputParser := parser.NewInputParser(store)
			return inputParser.Parse(in.RawInput)
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
			t := tool.NewHeroCompsTool(store)
			return t.Query(ctx, &data.HeroCompsInput{
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
			t := tool.NewItemFitTool(store)
			return t.Query(ctx, &data.ItemFitInput{
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
			t := tool.NewCompTierTool(store)
			return t.QueryTopTier(ctx)
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

			calc := tool.NewIntersectionCalc(store)
			return calc.Compute(&data.IntersectionInput{
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
			prompt := BuildPrompt(in)
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

type NluContext struct {
	UserInput  string
	Ctx        Context
	FinalReply string
}

// NluEnrichedContext NLU enriched context with data lookup results
type NluEnrichedContext struct {
	UserInput    string                 `json:"user_input"`
	Ctx          Context                `json:"ctx"`
	MatchedComps []data.Comp           `json:"matched_comps"` // 匹配的阵容
	MatchedItems []MatchedItemInfo      `json:"matched_items"` // 匹配的装备信息
}

// MatchedItemInfo 匹配的装备信息
type MatchedItemInfo struct {
	ItemID    string              `json:"item_id"`
	ItemName  string              `json:"item_name"`
	CompInfos []ItemFitCompInfo   `json:"comp_infos"` // 适配该装备的阵容
}

// ItemFitCompInfo 装备适配的阵容信息
type ItemFitCompInfo struct {
	ClusterID   string  `json:"cluster_id"`
	CompName    string  `json:"comp_name"`
	CompTier    string  `json:"comp_tier"`
	CompAvg     float64 `json:"comp_avg"`
	Carry       string  `json:"carry"`
	CarryName   string  `json:"carry_name"`
	PriorityScore int   `json:"priority_score"`
}

func BuildNluGraph(ctx context.Context, chatModel model.ChatModel, store *data.Store) (
	compose.Runnable[*NluContext, *NluEnrichedContext], error,
) {
	g := compose.NewGraph[*NluContext, *NluEnrichedContext]()

	// Node 1: NLU提取
	nluExtract := compose.InvokableLambda(func(ctx context.Context, input *NluContext) (output *NluContext, err error) {
		logrus.Println("用户输入:", input.UserInput)
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
		// 提取JSON内容：去除think标签和其他前缀，找到第一个{和最后一个}之间的内容
		content := resp.Content
		if startIdx := strings.Index(content, "{"); startIdx != -1 {
			if endIdx := strings.LastIndex(content, "}"); endIdx != -1 && endIdx > startIdx {
				content = content[startIdx : endIdx+1]
			}
		}
		if err := json.Unmarshal([]byte(content), &c); err != nil {
			logrus.WithError(err).WithField("raw_content", resp.Content).WithField("extracted_content", content).Warn("JSON解析失败，使用空Context")
		}
		input.Ctx = c
		logrus.Printf("llm 提取的内容为: %+v\n", input.Ctx)
		return input, nil
	})
	if err := g.AddLambdaNode("nlu_extract", nluExtract); err != nil {
		return nil, fmt.Errorf("add nlu_extract node: %w", err)
	}

	// Node 2: 数据查询和中文转换
	dataLookup := compose.InvokableLambda(func(ctx context.Context, input *NluContext) (output *NluEnrichedContext, err error) {
		result := &NluEnrichedContext{
			UserInput: input.UserInput,
			Ctx:       input.Ctx,
		}

		// 1. 将英雄/装备/羁绊转换为标准ID
		normalizedCtx := normalizeContext(input.Ctx, store)

		// 2. 查询匹配的阵容（根据英雄）
		if len(normalizedCtx.Champions) > 0 {
			heroIDs := make([]string, 0, len(normalizedCtx.Champions))
			for name := range normalizedCtx.Champions {
				if id := store.ResolveUnitID(name); id != "" {
					heroIDs = append(heroIDs, id)
				}
			}
			if len(heroIDs) > 0 {
				matches := store.GetCompsByUnits(heroIDs)
				for _, match := range matches {
					result.MatchedComps = append(result.MatchedComps, match.Comp)
				}
			}
		}

		// 3. 查询匹配的装备
		if len(normalizedCtx.Items) > 0 {
			for _, itemName := range normalizedCtx.Items {
				if itemID := store.ResolveItemID(itemName); itemID != "" {
					itemInfo := MatchedItemInfo{
						ItemID:   itemID,
						ItemName: store.IDToCN(itemID),
					}
					entries := store.GetItemFitEntries(itemID)
					for _, entry := range entries {
						compInfo := ItemFitCompInfo{
							ClusterID:    entry.ClusterID,
							CompName:     entry.CompName,
							CompTier:     entry.CompTier,
							CompAvg:      entry.CompAvg,
							Carry:        entry.Carry,
							CarryName:    store.IDToCN(entry.Carry),
							PriorityScore: entry.PriorityScore,
						}
						itemInfo.CompInfos = append(itemInfo.CompInfos, compInfo)
					}
					result.MatchedItems = append(result.MatchedItems, itemInfo)
				}
			}
		}

		return result, nil
	})
	if err := g.AddLambdaNode("data_lookup", dataLookup); err != nil {
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

// normalizeContext 尝试将Context中的黑话/昵称转换为标准名称
func normalizeContext(ctx Context, store *data.Store) Context {
	result := ctx

	// 规范化英雄名称（尝试用store.ResolveUnitID解析）
	if len(ctx.Champions) > 0 {
		normalizedChampions := make(map[string]int8)
		for name, star := range ctx.Champions {
			if id := store.ResolveUnitID(name); id != "" {
				normalizedChampions[store.IDToCN(id)] = star
			} else {
				normalizedChampions[name] = star
			}
		}
		result.Champions = normalizedChampions
	}

	// 规范化装备名称
	if len(ctx.Items) > 0 {
		normalizedItems := make([]string, 0, len(ctx.Items))
		for _, item := range ctx.Items {
			if id := store.ResolveItemID(item); id != "" {
				normalizedItems = append(normalizedItems, store.IDToCN(id))
			} else {
				normalizedItems = append(normalizedItems, item)
			}
		}
		result.Items = normalizedItems
	}

	// 规范化羁绊名称
	if len(ctx.Traits) > 0 {
		normalizedTraits := make([]string, 0, len(ctx.Traits))
		for _, trait := range ctx.Traits {
			normalizedTraits = append(normalizedTraits, store.IDToCN(trait))
		}
		result.Traits = normalizedTraits
	}

	return result
}
