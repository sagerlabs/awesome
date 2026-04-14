package knowledge

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
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

	// 1. Unmarshal请求：[]byte → shared contract
	var ctx contracts.QueryNLURequest
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

	// 4. Marshal响应：contract response → []byte
	respBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}

	return QueryResponse(respBytes), nil
}

// internalEnrichWithMetaData 用Meta数据丰富结果（内部版本）
func (s *UnifiedStore) internalEnrichWithMetaData(result *contracts.QueryNLUResponse, ctx contracts.QueryNLURequest) {
	// 补充Meta阵容数据
	for i, comp := range result.MatchedComps {
		if metaComp, ok := s.knowledgeStore.GetMetaCompByID(comp.ClusterID); ok {
			result.MatchedComps[i] = enrichCompSummaryWithMeta(comp, metaComp)
			s.logger.WithField("cluster_id", comp.ClusterID).Debug("Found meta comp")
		}
	}

	for itemIndex := range result.MatchedItems {
		for compIndex, compInfo := range result.MatchedItems[itemIndex].CompInfos {
			if metaComp, ok := s.knowledgeStore.GetMetaCompByID(compInfo.ClusterID); ok {
				if name := metaCompDisplayName(metaComp); name != "" {
					result.MatchedItems[itemIndex].CompInfos[compIndex].CompName = name
				}
			}
		}
	}

	// 补充Meta英雄数据
	for name := range ctx.Champions {
		if _, ok := s.knowledgeStore.GetMetaChampionByName(name); ok {
			s.logger.WithField("champion", name).Debug("Found meta champion")
		}
	}

	// 补充Meta装备数据
	for _, itemName := range ctx.Items {
		if _, ok := s.knowledgeStore.GetMetaItemByName(itemName); ok {
			s.logger.WithField("item", itemName).Debug("Found meta item")
		}
	}
}

func enrichCompSummaryWithMeta(summary contracts.CompSummary, metaComp *models.MetaComp) contracts.CompSummary {
	if metaComp == nil {
		return summary
	}

	if name := metaCompDisplayName(metaComp); name != "" {
		summary.Name = name
	}
	if metaComp.Tier != "" {
		summary.Tier = metaComp.Tier
	}
	if metaComp.AvgPlacement != 0 {
		summary.AvgPlacement = metaComp.AvgPlacement
	}
	if metaComp.Top4Rate != 0 {
		summary.Top4Rate = metaComp.Top4Rate
	}
	if metaComp.WinRate != 0 {
		summary.WinRate = metaComp.WinRate
	}
	if metaComp.Count != 0 {
		summary.Count = metaComp.Count
	}
	if len(metaComp.Units) > 0 {
		summary.Units = cloneStrings(metaComp.Units)
	}
	if len(metaComp.Traits) > 0 {
		summary.Traits = cloneStrings(metaComp.Traits)
	}
	if len(metaComp.Stars) > 0 {
		summary.Stars = filterStarsInUnits(metaComp.Stars, metaComp.Units, 3)
	}
	if metaComp.Levelling != "" {
		summary.Levelling = metaComp.Levelling
	}
	if metaComp.Difficulty != 0 {
		summary.Difficulty = metaComp.Difficulty
	}
	if len(metaComp.Builds) > 0 {
		summary.BestBuild = metaBuildToBuildInfo(metaComp.Builds[0])
		summary.AllBuilds = metaBuildsToBuildInfos(metaComp.Builds)
	}

	return summary
}

func metaCompDisplayName(metaComp *models.MetaComp) string {
	if metaComp == nil {
		return ""
	}

	names := make([]string, 0, len(metaComp.DisplayNames))
	for _, displayName := range metaComp.DisplayNames {
		name := strings.TrimSpace(displayName.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) > 0 {
		return strings.Join(names, "")
	}

	return strings.TrimSpace(metaComp.NameString)
}

func filterStarsInUnits(stars []string, units []string, limit int) []string {
	if len(stars) == 0 {
		return nil
	}

	unitSet := make(map[string]struct{}, len(units))
	for _, unit := range units {
		unitSet[unit] = struct{}{}
	}

	out := make([]string, 0, len(stars))
	for _, star := range stars {
		if len(unitSet) > 0 {
			if _, ok := unitSet[star]; !ok {
				continue
			}
		}
		out = append(out, star)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func metaBuildToBuildInfo(build models.CompBuild) contracts.BuildInfo {
	return contracts.BuildInfo{
		Carry:          build.Unit,
		Items:          cloneStrings(build.Items),
		PriorityScores: cloneIntMap(build.PriorityScores),
		AvgPlacement:   build.AvgPlacement,
		PlaceChange:    build.PlaceChange,
		Score:          build.Score,
	}
}

func metaBuildsToBuildInfos(builds []models.CompBuild) []contracts.BuildInfo {
	if len(builds) == 0 {
		return nil
	}

	out := make([]contracts.BuildInfo, 0, len(builds))
	for _, build := range builds {
		out = append(out, metaBuildToBuildInfo(build))
	}
	return out
}

// =============================================================================
// 阵容查询（返回JSON字节流）
// =============================================================================

// GetCompByID 通过ClusterID查询阵容（返回JSON字节流）
func (s *UnifiedStore) GetCompByID(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetCompByIDRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal get comp request: %w", err)
	}

	clusterID := request.ClusterID
	comp, ok := s.dataStore.GetCompByClusterID(clusterID)
	if !ok {
		return nil, fmt.Errorf("comp not found: %s", clusterID)
	}

	resp, err := json.Marshal(contracts.GetCompByIDResponse{Comp: ptrCompSummary(*comp, s.dataStore)})
	return Response(resp), err
}

// GetMetaCompByID 通过ClusterID查询Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetMetaCompByID(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetMetaCompByIDRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal get meta comp request: %w", err)
	}

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	clusterID := request.ClusterID
	comp, ok := s.knowledgeStore.GetMetaCompByID(clusterID)
	if !ok {
		return nil, fmt.Errorf("meta comp not found: %s", clusterID)
	}

	resp, err := json.Marshal(contracts.GetMetaCompByIDResponse{Comp: toContractMetaComp(comp)})
	return Response(resp), err
}

// GetMetaCompByName 通过名称查询Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetMetaCompByName(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetMetaCompByNameRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal get meta comp by name request: %w", err)
	}

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := request.Name
	comp, ok := s.knowledgeStore.GetMetaCompByName(name)
	if !ok {
		return nil, fmt.Errorf("meta comp not found: %s", name)
	}

	resp, err := json.Marshal(contracts.GetMetaCompByNameResponse{Comp: toContractMetaComp(comp)})
	return Response(resp), err
}

// SearchMetaComps 搜索Meta阵容（返回JSON字节流）
func (s *UnifiedStore) SearchMetaComps(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.SearchMetaCompsRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal search meta comps request: %w", err)
	}

	if s.knowledgeStore == nil {
		resp, err := json.Marshal(contracts.SearchMetaCompsResponse{Comps: []*contracts.MetaComp{}})
		return Response(resp), err
	}

	comps := toContractMetaComps(s.knowledgeStore.SearchMetaComps(request.Query))
	comps = paginateMetaComps(comps, request.Limit, request.Offset)
	resp, err := json.Marshal(contracts.SearchMetaCompsResponse{Comps: comps})
	return Response(resp), err
}

// GetAllMetaComps 获取所有Meta阵容（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaComps(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetAllMetaCompsRequest
	if len(req) > 0 {
		if err := json.Unmarshal([]byte(req), &request); err != nil {
			return nil, fmt.Errorf("unmarshal get all meta comps request: %w", err)
		}
	}

	if s.knowledgeStore == nil {
		resp, err := json.Marshal(contracts.GetAllMetaCompsResponse{Comps: []*contracts.MetaComp{}})
		return Response(resp), err
	}

	resp, err := json.Marshal(contracts.GetAllMetaCompsResponse{
		Comps: toContractMetaComps(s.knowledgeStore.GetAllMetaComps()),
	})
	return Response(resp), err
}

// =============================================================================
// 英雄查询（返回JSON字节流）
// =============================================================================

// GetMetaChampionByName 通过名称查询Meta英雄（返回JSON字节流）
func (s *UnifiedStore) GetMetaChampionByName(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetMetaChampionByNameRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal get meta champion request: %w", err)
	}

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := request.Name
	champ, ok := s.knowledgeStore.GetMetaChampionByName(name)
	if !ok {
		return nil, fmt.Errorf("meta champion not found: %s", name)
	}

	resp, err := json.Marshal(contracts.GetMetaChampionByNameResponse{Champion: toContractMetaChampion(champ)})
	return Response(resp), err
}

// GetAllMetaChampions 获取所有Meta英雄（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaChampions(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetAllMetaChampionsRequest
	if len(req) > 0 {
		if err := json.Unmarshal([]byte(req), &request); err != nil {
			return nil, fmt.Errorf("unmarshal get all meta champions request: %w", err)
		}
	}

	if s.knowledgeStore == nil {
		resp, err := json.Marshal(contracts.GetAllMetaChampionsResponse{Champions: []*contracts.MetaChampion{}})
		return Response(resp), err
	}

	resp, err := json.Marshal(contracts.GetAllMetaChampionsResponse{
		Champions: toContractMetaChampions(s.knowledgeStore.GetAllMetaChampions()),
	})
	return Response(resp), err
}

// =============================================================================
// 装备查询（返回JSON字节流）
// =============================================================================

// GetMetaItemByName 通过名称查询Meta装备（返回JSON字节流）
func (s *UnifiedStore) GetMetaItemByName(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetMetaItemByNameRequest
	if err := json.Unmarshal([]byte(req), &request); err != nil {
		return nil, fmt.Errorf("unmarshal get meta item request: %w", err)
	}

	if s.knowledgeStore == nil {
		return nil, fmt.Errorf("knowledgeStore not enabled")
	}

	name := request.Name
	item, ok := s.knowledgeStore.GetMetaItemByName(name)
	if !ok {
		return nil, fmt.Errorf("meta item not found: %s", name)
	}

	resp, err := json.Marshal(contracts.GetMetaItemByNameResponse{Item: toContractMetaItem(item)})
	return Response(resp), err
}

// GetAllMetaItems 获取所有Meta装备（返回JSON字节流）
func (s *UnifiedStore) GetAllMetaItems(req Request) (Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var request contracts.GetAllMetaItemsRequest
	if len(req) > 0 {
		if err := json.Unmarshal([]byte(req), &request); err != nil {
			return nil, fmt.Errorf("unmarshal get all meta items request: %w", err)
		}
	}

	if s.knowledgeStore == nil {
		resp, err := json.Marshal(contracts.GetAllMetaItemsResponse{Items: []*contracts.MetaItem{}})
		return Response(resp), err
	}

	resp, err := json.Marshal(contracts.GetAllMetaItemsResponse{
		Items: toContractMetaItems(s.knowledgeStore.GetAllMetaItems()),
	})
	return Response(resp), err
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
