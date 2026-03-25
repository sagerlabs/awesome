package knowledge

import (
	"fmt"
	"sync"

	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
	"github.com/sirupsen/logrus"
)

// UnifiedStore 统一知识库实现
// 同时持有data.Store和knowledge.Store，实现TFTKnowledgeTool接口
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

// QueryNLU NLU数据查询
func (s *UnifiedStore) QueryNLU(ctx agent.Context) (*agent.NluEnrichedContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.WithField("intent", ctx.Intent).Debug("QueryNLU called")

	// 1. 先用原有dataStore查询（保持兼容性）
	result := agent.QueryNLUData(ctx, s.dataStore)

	// 2. 如果启用Meta数据，补充Meta数据
	if s.config.EnableMeta && s.knowledgeStore != nil {
		s.enrichWithMetaData(result, ctx)
	}

	return result, nil
}

// enrichWithMetaData 用Meta数据丰富结果
func (s *UnifiedStore) enrichWithMetaData(result *agent.NluEnrichedContext, ctx agent.Context) {
	// 补充Meta阵容数据
	for _, comp := range result.MatchedComps {
		if metaComp, ok := s.knowledgeStore.GetMetaCompByID(comp.ClusterID); ok {
			// 这里可以扩展NluEnrichedContext来包含Meta数据
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
// 阵容查询
// =============================================================================

// GetCompByClusterID 通过ClusterID查询阵容（原有数据格式）
func (s *UnifiedStore) GetCompByClusterID(clusterID string) (*data.Comp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.GetCompByClusterID(clusterID)
}

// GetMetaCompByID 通过ClusterID查询Meta阵容（新数据格式）
func (s *UnifiedStore) GetMetaCompByID(clusterID string) (*models.MetaComp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil, false
	}
	return s.knowledgeStore.GetMetaCompByID(clusterID)
}

// GetMetaCompByName 通过名称查询Meta阵容
func (s *UnifiedStore) GetMetaCompByName(name string) (*models.MetaComp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil, false
	}
	return s.knowledgeStore.GetMetaCompByName(name)
}

// SearchMetaComps 搜索Meta阵容（关键词搜索）
func (s *UnifiedStore) SearchMetaComps(query string) []*models.MetaComp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil
	}
	return s.knowledgeStore.SearchMetaComps(query)
}

// GetAllMetaComps 获取所有Meta阵容
func (s *UnifiedStore) GetAllMetaComps() []*models.MetaComp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil
	}
	return s.knowledgeStore.GetAllMetaComps()
}

// GetCompsByUnits 通过英雄列表查询阵容（原有数据格式）
func (s *UnifiedStore) GetCompsByUnits(unitIDs []string) []data.CompMatch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.GetCompsByUnits(unitIDs)
}

// GetCompsByTier 通过Tier查询阵容（原有数据格式）
func (s *UnifiedStore) GetCompsByTier(tiers ...string) []*data.Comp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.GetCompsByTier(tiers...)
}

// =============================================================================
// 英雄查询
// =============================================================================

// GetMetaChampionByName 通过名称查询Meta英雄
func (s *UnifiedStore) GetMetaChampionByName(name string) (*models.MetaChampion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil, false
	}
	return s.knowledgeStore.GetMetaChampionByName(name)
}

// GetAllMetaChampions 获取所有Meta英雄
func (s *UnifiedStore) GetAllMetaChampions() []*models.MetaChampion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil
	}
	return s.knowledgeStore.GetAllMetaChampions()
}

// GetChampionBestBuild 获取英雄最佳装备（原有数据格式）
func (s *UnifiedStore) GetChampionBestBuild(clusterID string, unitName string) (*data.BuildInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	build := s.dataStore.GetBestItemsForUnit(clusterID, unitName)
	return build, build != nil
}

// =============================================================================
// 装备查询
// =============================================================================

// GetMetaItemByName 通过名称查询Meta装备
func (s *UnifiedStore) GetMetaItemByName(name string) (*models.MetaItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil, false
	}
	return s.knowledgeStore.GetMetaItemByName(name)
}

// GetAllMetaItems 获取所有Meta装备
func (s *UnifiedStore) GetAllMetaItems() []*models.MetaItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.knowledgeStore == nil {
		return nil
	}
	return s.knowledgeStore.GetAllMetaItems()
}

// GetItemFitEntries 查询装备适配的阵容（原有数据格式）
func (s *UnifiedStore) GetItemFitEntries(itemID string) []data.ItemFitEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.GetItemFitEntries(itemID)
}

// GetItemFitByItems 批量装备查询（原有数据格式）
func (s *UnifiedStore) GetItemFitByItems(itemIDs []string) []data.ItemFitResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataStore.GetItemFitByItems(itemIDs)
}

// =============================================================================
// 名称解析和转换
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

	// TODO: 实现重新加载逻辑
	// 需要重新创建dataStore和knowledgeStore

	return fmt.Errorf("reload not implemented yet")
}

// HealthCheck 健康检查
func (s *UnifiedStore) HealthCheck() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 检查dataStore
	if s.dataStore == nil {
		return fmt.Errorf("dataStore is nil")
	}

	// 检查knowledgeStore（如果启用了）
	if s.config.EnableMeta && s.knowledgeStore == nil {
		return fmt.Errorf("knowledgeStore is nil but EnableMeta is true")
	}

	return nil
}

// =============================================================================
// 辅助方法
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
