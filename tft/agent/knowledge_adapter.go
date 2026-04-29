package agent

import (
	"encoding/json"
	"fmt"

	"github.com/sagerlabs/awesome/tft/knowledge"
	"github.com/sagerlabs/awesome/tft/knowledge/contracts"
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
	reqBytes, err := marshalKnowledgeRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	// 2. 调用knowledge（字节流接口）
	respBytes, err := a.tool.QueryNLU(knowledge.QueryRequest(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("query knowledge: %w", err)
	}

	// 3. Unmarshal: []byte → shared contract response
	var result contracts.QueryNLUResponse
	if err := json.Unmarshal([]byte(respBytes), &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// =============================================================================
// 其他类型安全的查询方法
// =============================================================================

// GetCompByID 通过ClusterID查询阵容（类型安全）
func (a *KnowledgeAdapter) GetCompByID(clusterID string) (*contracts.CompSummary, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetCompByIDRequest{ClusterID: clusterID})
	if err != nil {
		return nil, fmt.Errorf("marshal get comp request: %w", err)
	}

	respBytes, err := a.tool.GetCompByID(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetCompByIDResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get comp response: %w", err)
	}

	return resp.Comp, nil
}

// GetMetaCompByID 通过ClusterID查询Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetMetaCompByID(clusterID string) (*MetaComp, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetMetaCompByIDRequest{ClusterID: clusterID})
	if err != nil {
		return nil, fmt.Errorf("marshal get meta comp request: %w", err)
	}

	respBytes, err := a.tool.GetMetaCompByID(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetMetaCompByIDResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get meta comp response: %w", err)
	}

	return resp.Comp, nil
}

// GetMetaCompByName 通过名称查询Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetMetaCompByName(name string) (*MetaComp, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetMetaCompByNameRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("marshal get meta comp by name request: %w", err)
	}

	respBytes, err := a.tool.GetMetaCompByName(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetMetaCompByNameResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get meta comp by name response: %w", err)
	}

	return resp.Comp, nil
}

// SearchMetaComps 搜索Meta阵容（类型安全）
func (a *KnowledgeAdapter) SearchMetaComps(query string) ([]*MetaComp, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.SearchMetaCompsRequest{Query: query})
	if err != nil {
		return nil, fmt.Errorf("marshal search meta comps request: %w", err)
	}

	respBytes, err := a.tool.SearchMetaComps(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.SearchMetaCompsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal search meta comps response: %w", err)
	}

	return resp.Comps, nil
}

// GetAllMetaComps 获取所有Meta阵容（类型安全）
func (a *KnowledgeAdapter) GetAllMetaComps() ([]*MetaComp, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetAllMetaCompsRequest{})
	if err != nil {
		return nil, fmt.Errorf("marshal get all meta comps request: %w", err)
	}

	respBytes, err := a.tool.GetAllMetaComps(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetAllMetaCompsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get all meta comps response: %w", err)
	}

	return resp.Comps, nil
}

// GetMetaChampionByName 通过名称查询Meta英雄（类型安全）
func (a *KnowledgeAdapter) GetMetaChampionByName(name string) (*MetaChampion, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetMetaChampionByNameRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("marshal get meta champion request: %w", err)
	}

	respBytes, err := a.tool.GetMetaChampionByName(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetMetaChampionByNameResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get meta champion response: %w", err)
	}

	return resp.Champion, nil
}

// GetAllMetaChampions 获取所有Meta英雄（类型安全）
func (a *KnowledgeAdapter) GetAllMetaChampions() ([]*MetaChampion, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetAllMetaChampionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("marshal get all meta champions request: %w", err)
	}

	respBytes, err := a.tool.GetAllMetaChampions(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetAllMetaChampionsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get all meta champions response: %w", err)
	}

	return resp.Champions, nil
}

// GetMetaItemByName 通过名称查询Meta装备（类型安全）
func (a *KnowledgeAdapter) GetMetaItemByName(name string) (*MetaItem, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetMetaItemByNameRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("marshal get meta item request: %w", err)
	}

	respBytes, err := a.tool.GetMetaItemByName(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetMetaItemByNameResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get meta item response: %w", err)
	}

	return resp.Item, nil
}

// GetAllMetaItems 获取所有Meta装备（类型安全）
func (a *KnowledgeAdapter) GetAllMetaItems() ([]*MetaItem, error) {
	reqBytes, err := marshalKnowledgeRequest(contracts.GetAllMetaItemsRequest{})
	if err != nil {
		return nil, fmt.Errorf("marshal get all meta items request: %w", err)
	}

	respBytes, err := a.tool.GetAllMetaItems(reqBytes)
	if err != nil {
		return nil, err
	}

	var resp contracts.GetAllMetaItemsResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal get all meta items response: %w", err)
	}

	return resp.Items, nil
}

func marshalKnowledgeRequest(req any) (knowledge.Request, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	return knowledge.Request(reqBytes), nil
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
