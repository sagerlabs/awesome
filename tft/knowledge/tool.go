package knowledge

import (
	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sagerlabs/awesome/tft/knowledge/models"
)

// TFTKnowledgeTool TFT知识库Tool接口
// 设计目标：作为独立tool/skill使用，接口清晰，可分割部署
type TFTKnowledgeTool interface {
	// =========================================================================
	// NLU查询（核心方法）
	// =========================================================================

	// QueryNLU NLU数据查询：输入Context，返回EnrichedContext
	// 这是最核心的方法，供agent层调用
	QueryNLU(ctx agent.Context) (*agent.NluEnrichedContext, error)

	// =========================================================================
	// 阵容查询
	// =========================================================================

	// GetCompByClusterID 通过ClusterID查询阵容（原有数据格式）
	GetCompByClusterID(clusterID string) (*data.Comp, bool)

	// GetMetaCompByID 通过ClusterID查询Meta阵容（新数据格式）
	GetMetaCompByID(clusterID string) (*models.MetaComp, bool)

	// GetMetaCompByName 通过名称查询Meta阵容
	GetMetaCompByName(name string) (*models.MetaComp, bool)

	// SearchMetaComps 搜索Meta阵容（关键词搜索）
	SearchMetaComps(query string) []*models.MetaComp

	// GetAllMetaComps 获取所有Meta阵容
	GetAllMetaComps() []*models.MetaComp

	// GetCompsByUnits 通过英雄列表查询阵容（原有数据格式）
	GetCompsByUnits(unitIDs []string) []data.CompMatch

	// GetCompsByTier 通过Tier查询阵容（原有数据格式）
	GetCompsByTier(tiers ...string) []*data.Comp

	// =========================================================================
	// 英雄查询
	// =========================================================================

	// GetMetaChampionByName 通过名称查询Meta英雄
	GetMetaChampionByName(name string) (*models.MetaChampion, bool)

	// GetAllMetaChampions 获取所有Meta英雄
	GetAllMetaChampions() []*models.MetaChampion

	// GetChampionByID 通过ID查询英雄（原有数据格式）
	// 注意：原有数据没有单独的英雄模型，返回最佳装备信息
	GetChampionBestBuild(clusterID string, unitName string) (*data.BuildInfo, bool)

	// =========================================================================
	// 装备查询
	// =========================================================================

	// GetMetaItemByName 通过名称查询Meta装备
	GetMetaItemByName(name string) (*models.MetaItem, bool)

	// GetAllMetaItems 获取所有Meta装备
	GetAllMetaItems() []*models.MetaItem

	// GetItemFitEntries 查询装备适配的阵容（原有数据格式）
	GetItemFitEntries(itemID string) []data.ItemFitEntry

	// GetItemFitByItems 批量装备查询（原有数据格式）
	GetItemFitByItems(itemIDs []string) []data.ItemFitResult

	// =========================================================================
	// 名称解析和转换
	// =========================================================================

	// ResolveUnitID 解析英雄输入：支持中文名、ID、模糊匹配
	ResolveUnitID(input string) string

	// ResolveItemID 解析装备输入：支持中文名、ID、模糊匹配
	ResolveItemID(input string) string

	// IDToCN 将ID转换为中文名
	IDToCN(id string) string

	// CNToID 将中文名转换为ID
	CNToID(cn string) string

	// =========================================================================
	// 数据管理
	// =========================================================================

	// Reload 重新加载数据（热更新）
	Reload() error

	// HealthCheck 健康检查
	HealthCheck() error
}

// =============================================================================
// 查询参数类型
// =============================================================================

// CompSearchQuery 阵容搜索查询
type CompSearchQuery struct {
	// 关键词（英雄、羁绊、阵容名称等）
	Keywords string `json:"keywords"`

	// 过滤条件
	Tiers     []string `json:"tiers,omitempty"`     // 只返回指定Tier的阵容
	MinTop4Rate float64  `json:"min_top4_rate,omitempty"` // 最低前4率
	MaxAvgPlace float64  `json:"max_avg_place,omitempty"` // 最高平均名次

	// 分页
	Limit  int `json:"limit,omitempty"`  // 返回数量限制
	Offset int `json:"offset,omitempty"` // 偏移量
}

// ChampionQuery 英雄查询参数
type ChampionQuery struct {
	Name     string   `json:"name"`     // 英雄名称
	Clusters []string `json:"clusters,omitempty"` // 限定在某些阵容中查询
}

// ItemQuery 装备查询参数
type ItemQuery struct {
	Name       string   `json:"name"`       // 装备名称
	MinScore   int      `json:"min_score,omitempty"`  // 最低优先级分数
	Clusters   []string `json:"clusters,omitempty"`   // 限定在某些阵容中查询
}

// =============================================================================
// Tool配置
// =============================================================================

// ToolConfig Tool配置
type ToolConfig struct {
	// 数据目录
	DataDir      string `json:"data_dir"`
	KnowledgeDir string `json:"knowledge_dir"`

	// 是否启用Meta数据
	EnableMeta bool `json:"enable_meta"`

	// 日志配置
	EnableLog  bool   `json:"enable_log"`
	LogLevel   string `json:"log_level"`
}

// DefaultToolConfig 默认配置
func DefaultToolConfig() *ToolConfig {
	return &ToolConfig{
		DataDir:      "metadata/tft-meta/data",
		KnowledgeDir: "tft/knowledge/data",
		EnableMeta:   true,
		EnableLog:    true,
		LogLevel:     "info",
	}
}
