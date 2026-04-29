package agent

import (
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge"
	"github.com/sirupsen/logrus"
)

// newKnowledgeAdapterFromStore 为 agent 构建默认的 knowledge 适配器。
// 这里优先尝试接入 knowledge data，失败时退化为只使用 dataStore 的最小实现。
func newKnowledgeAdapterFromStore(dataStore *data.Store, logger *logrus.Logger) (*KnowledgeAdapter, error) {
	var knowledgeStore *knowledge.Store
	enableMeta := false

	loader := knowledge.NewLoader("tft/knowledge/data")
	store, err := loader.LoadAll()
	if err == nil {
		knowledgeStore = store
		enableMeta = true
	} else if logger != nil {
		logger.WithError(err).Warn("load knowledge store failed, fallback to data-only knowledge adapter")
	}

	tool, err := knowledge.NewUnifiedStore(dataStore, knowledgeStore, &knowledge.ToolConfig{
		EnableMeta: enableMeta,
		EnableLog:  false,
	})
	if err != nil {
		return nil, err
	}

	return NewKnowledgeAdapter(tool), nil
}
