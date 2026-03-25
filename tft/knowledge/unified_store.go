package knowledge

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sirupsen/logrus"
)

// UnifiedStore 统一知识库实现
// 同时持有data.Store和knowledge.Store，实现TFTKnowledgeTool接口
// 注意：实现层可以依赖data包，接口层是零依赖的
type UnifiedStore struct {
	dataStore      *data.Store
	knowledgeStore *Store
	config         *ToolConfig
	logger         *logrus.Logger
	mu             sync.RWMutex
}

// NewUnifiedStore 创建统一知识库
func NewUnifiedStore(dataStore *data.Store, knowledgeStore *Store, config *ToolConfig) (*UnifiedStore, error) {
	if config == nil {
		config = DefaultToolConfig()
	}

	logger := logrus.StandardLogger()
	if !config.EnableLog {
		logger.SetLevel(logrus.PanicLevel)
	}

	return &UnifiedStore{
		dataStore:      dataStore,
		knowledgeStore: knowledgeStore,
		config:         config,
		logger:         logger,
	}, nil
}

// =============================================================================
// NLU查询（核心方法）
// =============================================================================

// QueryNLU NLU数据查询：输入字节流，返回字节流
func (s *UnifiedStore) QueryNLU(req QueryRequest) (QueryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Debug("QueryNLU called (byte stream)")

	// 1. Unmarshal请求：[]byte → internalContext
	var ctx internalContext
	if err := json.Unmarshal([]byte(req), &ctx); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	s.logger.WithField("intent", ctx.Intent).Debug("Parsed context")

	// 2. 内部调用（使用内部逻辑，避免引用agent包）
	result := internalQueryNLUData(ctx, s.dataStore)

	// 3. 如果启用Meta数据，补充Meta数据
	if s.config.EnableMeta && s.knowledgeStore != nil {
		s.internalEnrichWithMetaData(result, ctx)
	}

	// 4. Marshal响应：internalNluEnrichedContext → []byte
	respBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	return QueryResponse(respBytes), nil
}

// internalEnrichWithMetaData 用Meta数据丰富结果（内部版本）
func (s *UnifiedStore) internalEnrichWithMetaData(result *internalNluEnrichedContext, ctx internalContext) {
	// 补充Meta阵容数据
	for _, comp := range result.MatchedComps {
		if metaComp, ok := s.knowledgeStore.GetMetaCompByID(comp.ClusterID); ok {
			s.logger.WithField("cluster_id", comp.ClusterID).Debug("Found meta comp")
		}
	}

	// 补充Meta英雄数据
	for name := range ctx.Champions {
		if metaChamp, ok := s.knowledgeStore.GetMetaChampionByName(name); ok {
			s.logger.WithField("champion", name).Debug("Found meta champion")
		}
	}

	// 补充Meta装备数据
	for _, itemName := range ctx.Items {
		if metaItem, ok := s.knowledgeStore.GetMetaItemByName(itemName); ok {
			s.logger.WithField("item", itemName).Debug("Found meta item")
		}
	}
}

// =============================================================================
// 阵容查询（返回JSON字节流）
// =============================================================================

// GetCompByID 通过ClusterID查询阵容（返回JSON字节流）
func (s *UnifiedStore) GetCompByID(clusterID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	comp, ok := s.dataStore.GetCompByClusterID(clusterID)
	if !ok {
		return nil, fmt.Errorf("comp not found: %s", clusterID)
	}

	return json.Marshal(comp)
}

// GetMetaCompByID 通过ClusterID查询Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetMetaCompByID(clusterID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	comp, ok := s.knowledgeStore.GetMetaCompByID(clusterID)
	if !ok {
		return nil, fmt.Errorf("meta comp not found: %s", clusterID)
	}

	return json.Marshal(comp)
}

// GetMetaCompByName 通过名称查询Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetMetaCompByName(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	comp, ok := s.knowledgeStore.GetMetaCompByName(name)
	if !ok {
		return nil, fmt.Errorf("meta comp not found: %s", name)
	}

	return json.Marshal(comp)
}

// SearchMetaComps 搜索Meta阵容（返回JSON字节流）
func (s *UnifiedStore) SearchMetaComps(query string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return json.Marshal([]interface{}{})
	}

	comps := s.knowledgeStore.SearchMetaComps(query)
	return json.Marshal(comps)
}

// GetAllMetaComps 获取所有Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaComps() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return json.Marshal([]interface{}{})
	}

	comps := s.knowledgeStore.GetAllMetaComps()
	return json.Marshal(comps)
}

// =============================================================================
// 英雄查询（返回JSON字节流）
// =============================================================================

// GetMetaChampionByName 通过名称查询Meta英雄（返回JSON字节流）
func (s *UnifiedStore) GetMetaChampionByName(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	champ, ok := s.knowledgeStore.GetMetaChampionByName(name)
	if !ok {
		return nil, fmt.Errorf("meta champion not found: %s", name)
	}

	return json.Marshal(champ)
}

// GetAllMetaChampions 获取所有Meta英雄（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaChampions() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return json.Marshal([]interface{}{})
	}

	champs := s.knowledgeStore.GetAllMetaChampions()
	return json.Marshal(champs)
}

// =============================================================================
// 装备查询（返回JSON字节流）
// =============================================================================

// GetMetaItemByName 通过名称查询Meta装备（返回JSON字节流）
func (s *UnifiedStore) GetMetaItemByName(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	item, ok := s.knowledgeStore.GetMetaItemByName(name)
	if !ok {
		return nil, fmt.Errorf("meta item not found: %s", name)
	}

	return json.Marshal(item)
}

// GetAllMetaItems 获取所有Meta装备（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaItems() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.knowledgeStore == nil {
		return json.Marshal([]interface{}{})
	}

	items := s.knowledgeStore.GetAllMetaItems()
	return json.Marshal(items)
}

// =============================================================================
// 名称解析和转换（直接返回值，不需要JSON）
// =============================================================================

// ResolveUnitID 解析英雄输入
func (s *UnifiedStore) ResolveUnitID(input string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.ResolveUnitID(input)
}

// ResolveItemID 解析装备输入
func (s *UnifiedStore) ResolveItemID(input string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.ResolveItemID(input)
}

// IDToCN 将ID转换为中文名
func (s *UnifiedStore) IDToCN(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.IDToCN(id)
}

// CNToID 将中文名转换为ID
func (s *UnifiedStore) CNToID(cn string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.CNToID(cn)
}

// =============================================================================
// 数据管理
// =============================================================================

// Reload 重新加载数据（热更新）
func (s *UnifiedStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Reloading data...")

	// 1. 重新加载dataStore
	dataDir := s.config.DataDir
	if dataDir == "" {
		dataDir = data.GetDataDir()
	}

	newDataStore, err := data.NewStore(dataDir)
	if err != nil {
		return fmt.Errorf("reload data store: %w", err)
	}

	// 2. 重新加载knowledgeStore（如果启用了）
	var newKnowledgeStore *Store
	if s.config.EnableMeta {
		knowledgeDir := s.config.KnowledgeDir
		if knowledgeDir == "" {
			knowledgeDir = "tft/knowledge/data"
		}

		loader := NewLoader(knowledgeDir)
		var loadErr error
		newKnowledgeStore, loadErr = loader.LoadAll()
		if loadErr != nil {
			s.logger.WithError(loadErr).Warn("Reload knowledge store failed, keeping old one")
			// 不返回错误，继续使用旧的knowledgeStore
			newKnowledgeStore = s.knowledgeStore
		}
	}

	// 3. 原子替换
	s.dataStore = newDataStore
	s.knowledgeStore = newKnowledgeStore

	s.logger.Info("Data reloaded successfully")
	return nil
}

// HealthCheck 健康检查
func (s *UnifiedStore) HealthCheck() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dataStore == nil {
		return fmt.Errorf("dataStore is nil")
	}

	if s.config.EnableMeta && s.knowledgeStore == nil {
		return fmt.Errorf("knowledgeStore is nil but EnableMeta is true")
	}

	return nil
}

// =============================================================================
// 辅助方法（用于兼容性，不通过接口暴露）
// =============================================================================

// GetDataStore 获取底层dataStore（用于兼容性）
func (s *UnifiedStore) GetDataStore() *data.Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore
}

// GetKnowledgeStore 获取底层knowledgeStore（用于兼容性）
func (s *UnifiedStore) GetKnowledgeStore() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.knowledgeStore
}
