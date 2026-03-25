package agent

import (
	"encoding/json"
	"fmt"

	"github.com/sagerlabs/awesome/tft/knowledge"
)

// KnowledgeAdapter knowledge的适配器
// 提供类型安全的接口，内部做字节流转换
// 这样agent包可以继续使用类型安全的Context和NluEnrichedContext
// 而knowledge包保持零依赖
type KnowledgeAdapter struct {
	tool knowledge.TFTKnowledgeTool
}

// NewKnowledgeAdapter 创建适配器
func NewKnowledgeAdapter(tool knowledge.TFTKnowledgeTool) *KnowledgeAdapter {
	return &KnowledgeAdapter{tool: tool}
}

// =============================================================================
// 类型安全的查询方法（agent包使用）
// =============================================================================

// QueryNLU 类型安全的NLU查询
// 输入：agent.Context（类型安全）
// 输出：*agent.NluEnrichedContext（类型安全）
func (a *KnowledgeAdapter) QueryNLU(ctx Context) (*NluEnrichedContext, error) {
	// 1. Marshal: agent.Context → []byte
	reqBytes, err := json.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	// 2. 调用knowledge（字节流接口）
	respBytes, err := a.tool.QueryNLU(knowledge.QueryRequest(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("query knowledge: %w", err)
	}

	// 3. Unmarshal: []byte → agent.NluEnrichedContext
	var result NluEnrichedContext
	if err := json.Unmarshal([]byte(respBytes), &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// =============================================================================
// 其他类型安全的查询方法
// =============================================================================

// GetCompByID 通过ClusterID查询阵容（类型安全）
func (a *KnowledgeAdapter) GetCompByID(clusterID string) (*Comp, error) {
	respBytes, err := a.tool.GetCompByID(clusterID)
	if err != nil {
		return nil, err
	}

	var comp Comp
	if err := json.Unmarshal(respBytes, &comp); err != nil {
		return nil, fmt.Errorf("unmarshal comp: %w", err)
	}

	return &comp, nil
}

// GetMetaCompByID 通过ClusterID查询Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetMetaCompByID(clusterID string) (*MetaComp, error) {
	respBytes, err := a.tool.GetMetaCompByID(clusterID)
	if err != nil {
		return nil, err
	}

	var comp MetaComp
	if err := json.Unmarshal(respBytes, &comp); err != nil {
		return nil, fmt.Errorf("unmarshal meta comp: %w", err)
	}

	return &comp, nil
}

// GetMetaCompByName 通过名称查询Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetMetaCompByName(name string) (*MetaComp, error) {
	respBytes, err := a.tool.GetMetaCompByName(name)
	if err != nil {
		return nil, err
	}

	var comp MetaComp
	if err := json.Unmarshal(respBytes, &comp); err != nil {
		return nil, fmt.Errorf("unmarshal meta comp: %w", err)
	}

	return &comp, nil
}

// SearchMetaComps 搜索Meta阵容（类型安全）
func (a *KnowledgeAdapter) SearchMetaComps(query string) ([]*MetaComp, error) {
	respBytes, err := a.tool.SearchMetaComps(query)
	if err != nil {
		return nil, err
	}

	var comps []*MetaComp
	if err := json.Unmarshal(respBytes, &comps); err != nil {
		return nil, fmt.Errorf("unmarshal meta comps: %w", err)
	}

	return comps, nil
}

// GetAllMetaComps 获取所有Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetAllMetaComps() ([]*MetaComp, error) {
	respBytes, err := a.tool.GetAllMetaComps()
	if err != nil {
		return nil, err
	}

	var comps []*MetaComp
	if err := json.Unmarshal(respBytes, &comps); err != nil {
		return nil, fmt.Errorf("unmarshal meta comps: %w", err)
	}

	return comps, nil
}

// GetMetaChampionByName 通过名称查询Meta英雄（类型安全）
func (a *KnowledgeAdapter) GetMetaChampionByName(name string) (*MetaChampion, error) {
	respBytes, err := a.tool.GetMetaChampionByName(name)
	if err != nil {
		return nil, err
	}

	var champ MetaChampion
	if err := json.Unmarshal(respBytes, &champ); err != nil {
		return nil, fmt.Errorf("unmarshal meta champion: %w", err)
	}

	return &champ, nil
}

// GetAllMetaChampions 获取所有Meta英雄（类型安全）
func (a *KnowledgeAdapter) GetAllMetaChampions() ([]*MetaChampion, error) {
	respBytes, err := a.tool.GetAllMetaChampions()
	if err != nil {
		return nil, err
	}

	var champs []*MetaChampion
	if err := json.Unmarshal(respBytes, &champs); err != nil {
		return nil, fmt.Errorf("unmarshal meta champions: %w", err)
	}

	return champs, nil
}

// GetMetaItemByName 通过名称查询Meta装备（类型安全）
func (a *KnowledgeAdapter) GetMetaItemByName(name string) (*MetaItem, error) {
	respBytes, err := a.tool.GetMetaItemByName(name)
	if err != nil {
		return nil, err
	}

	var item MetaItem
	if err := json.Unmarshal(respBytes, &item); err != nil {
		return nil, fmt.Errorf("unmarshal meta item: %w", err)
	}

	return &item, nil
}

// GetAllMetaItems 获取所有Meta装备（类型安全）
func (a *KnowledgeAdapter) GetAllMetaItems() ([]*MetaItem, error) {
	respBytes, err := a.tool.GetAllMetaItems()
	if err != nil {
		return nil, err
	}

	var items []*MetaItem
	if err := json.Unmarshal(respBytes, &items); err != nil {
		return nil, fmt.Errorf("unmarshal meta items: %w", err)
	}

	return items, nil
}

// =============================================================================
// 名称解析和转换（直接代理，不需要JSON）
// =============================================================================

// ResolveUnitID 解析英雄输入
func (a *KnowledgeAdapter) ResolveUnitID(input string) string {
	return a.tool.ResolveUnitID(input)
}

// ResolveItemID 解析装备输入
func (a *KnowledgeAdapter) ResolveItemID(input string) string {
	return a.tool.ResolveItemID(input)
}

// IDToCN 将ID转换为中文名
func (a *KnowledgeAdapter) IDToCN(id string) string {
	return a.tool.IDToCN(id)
}

// CNToID 将中文名转换为ID
func (a *KnowledgeAdapter) CNToID(cn string) string {
	return a.tool.CNToID(cn)
}

// =============================================================================
// 数据管理（直接代理）
// =============================================================================

// Reload 重新加载数据
func (a *KnowledgeAdapter) Reload() error {
	return a.tool.Reload()
}

// HealthCheck 健康检查
func (a *KnowledgeAdapter) HealthCheck() error {
	return a.tool.HealthCheck()
}

// =============================================================================
// 辅助方法
// =============================================================================

// GetTool 获取底层tool（用于兼容性）
func (a *KnowledgeAdapter) GetTool() knowledge.TFTKnowledgeTool {
	return a.tool
}
