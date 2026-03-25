package knowledge

// =============================================================================
// 接口层（零依赖！）
// =============================================================================

// QueryRequest 查询请求（字节流）
// 用于：agent.Context → JSON → QueryRequest
type QueryRequest []byte

// QueryResponse 查询响应（字节流）
// 用于：QueryResponse → JSON → agent.NluEnrichedContext
type QueryResponse []byte

// TFTKnowledgeTool TFT知识库Tool接口
// 设计目标：作为独立tool/skill使用，接口清晰，可分割部署
// 注意：接口层零依赖，不引用agent、data等包
type TFTKnowledgeTool interface {
	// =========================================================================
	// NLU查询（核心方法）
	// =========================================================================

	// QueryNLU NLU数据查询：输入字节流，返回字节流
	// 输入格式：JSON序列化的 agent.Context
	// 输出格式：JSON序列化的 agent.NluEnrichedContext
	QueryNLU(req QueryRequest) (QueryResponse, error)

	// =========================================================================
	// 阵容查询
	// =========================================================================

	// GetCompByID 通过ClusterID查询阵容（返回JSON字节流）
	GetCompByID(clusterID string) ([]byte, error)

	// GetMetaCompByID 通过ClusterID查询Meta阵容（返回JSON字节流）
	GetMetaCompByID(clusterID string) ([]byte, error)

	// GetMetaCompByName 通过名称查询Meta阵容（返回JSON字节流）
	GetMetaCompByName(name string) ([]byte, error)

	// SearchMetaComps 搜索Meta阵容（返回JSON字节流）
	SearchMetaComps(query string) ([]byte, error)

	// GetAllMetaComps 获取所有Meta阵容（返回JSON字节流）
	GetAllMetaComps() ([]byte, error)

	// =========================================================================
	// 英雄查询
	// =========================================================================

	// GetMetaChampionByName 通过名称查询Meta英雄（返回JSON字节流）
	GetMetaChampionByName(name string) ([]byte, error)

	// GetAllMetaChampions 获取所有Meta英雄（返回JSON字节流）
	GetAllMetaChampions() ([]byte, error)

	// =========================================================================
	// 装备查询
	// =========================================================================

	// GetMetaItemByName 通过名称查询Meta装备（返回JSON字节流）
	GetMetaItemByName(name string) ([]byte, error)

	// GetAllMetaItems 获取所有Meta装备（返回JSON字节流）
	GetAllMetaItems() ([]byte, error)

	// =========================================================================
	// 名称解析和转换
	// =========================================================================

	// ResolveUnitID 解析英雄输入
	ResolveUnitID(input string) string

	// ResolveItemID 解析装备输入
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
