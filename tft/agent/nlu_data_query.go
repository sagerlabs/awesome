package agent

import (
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
)

type NluContext struct {
	UserInput  string
	Ctx        Context
	FinalReply string
}

// NluEnrichedContext 是 agent 侧对共享 QueryNLUResponse 的别名。
type NluEnrichedContext = contracts.QueryNLUResponse

// MatchedItemInfo 是 agent 侧对共享 MatchedItemInfo 的别名。
type MatchedItemInfo = contracts.MatchedItemInfo

// ItemFitCompInfo 是 agent 侧对共享 ItemFitCompInfo 的别名。
type ItemFitCompInfo = contracts.ItemFitCompInfo

// QueryNLUData 为保留当前对外 API，内部转调 knowledge 的共享 contract 路径。
func QueryNLUData(ctx Context, store *data.Store) *NluEnrichedContext {
	tool, err := knowledge.NewUnifiedStore(store, nil, &knowledge.ToolConfig{
		DataDir:    "",
		EnableMeta: false,
		EnableLog:  false,
	})
	if err != nil {
		return &NluEnrichedContext{Ctx: ctx}
	}

	result, err := NewKnowledgeAdapter(tool).QueryNLU(ctx)
	if err != nil {
		return &NluEnrichedContext{Ctx: ctx}
	}

	return result
}
