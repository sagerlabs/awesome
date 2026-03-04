package data

// =============================================================================
// JSON 文件映射结构体
// 对应 metadata/tft-meta/data/ 下的三个数据文件
// =============================================================================

// ── comps_for_agent.json ──────────────────────────────────────────────────────

// CompsFile comps_for_agent.json 顶层结构
type CompsFile struct {
	Meta  CompsMeta `json:"meta"`
	Comps []Comp    `json:"comps"`
}

// CompsMeta 元信息
type CompsMeta struct {
	TFTSet    string `json:"tft_set"`    // "TFTSet16"
	ClusterID string `json:"cluster_id"` // "393"
}

// Comp 单个阵容的完整数据
type Comp struct {
	ClusterID    string      `json:"cluster_id"`    // "393000"，MetaTFT 唯一 ID
	Name         string      `json:"name"`          // "TFT16_Augment_RumbleCarry"
	Tier         string      `json:"tier"`          // "S" / "A" / "B" / "C"
	AvgPlacement float64     `json:"avg_placement"` // 平均名次，越低越强
	Top4Rate     float64     `json:"top4_rate"`     // 进前4率 0~1
	WinRate      float64     `json:"win_rate"`      // 第一名率 0~1
	Count        int         `json:"count"`         // 样本场次数
	Units        []string    `json:"units"`         // 核心英雄 ID 列表，如 ["TFT16_Rumble", ...]
	Traits       []string    `json:"traits"`        // 关键羁绊，如 ["TFT16_Yordle_4"]
	Stars        []string    `json:"stars"`         // 推荐升3星的英雄（优先级从高到低）
	Levelling    string      `json:"levelling"`     // 推荐升级节点，如 "lvl 8"
	Difficulty   float64     `json:"difficulty"`    // 操作难度，负数=较难，正数=较易
	BestBuild    BuildInfo   `json:"best_build"`    // 最优装备方案（score 最高的那套）
	AllBuilds    []BuildInfo `json:"all_builds"`    // 所有装备方案，按 score 降序
}

// BuildInfo 单个 carry 英雄的装备方案
type BuildInfo struct {
	Carry          string         `json:"carry"`           // carry 英雄 ID，如 "TFT16_Rumble"
	Items          []string       `json:"items"`           // 装备 ID 列表（顺序即优先级）
	PriorityScores map[string]int `json:"priority_scores"` // {"TFT_Item_Rabadons": 100, ...}
	AvgPlacement   float64        `json:"avg_placement"`   // 使用该套装备的平均名次
	PlaceChange    float64        `json:"place_change"`    // 相比不带装备的名次变化（负=更好）
	Score          float64        `json:"score,omitempty"` // MetaTFT 内部综合评分
}

// ── items_priority.json ───────────────────────────────────────────────────────

// ItemsFile items_priority.json 顶层结构
// key: 装备 ID（如 "TFT_Item_GuinsoosRageblade"）
// val: 该装备适合的阵容列表，已按 priority_score 降序排列
type ItemsFile map[string][]ItemFitEntry

// ItemFitEntry 装备在某个阵容中的适配信息
type ItemFitEntry struct {
	ClusterID     string  `json:"cluster_id"`     // "393000"
	CompName      string  `json:"comp_name"`      // "TFT16_Augment_RumbleCarry"
	CompTier      string  `json:"comp_tier"`      // "S"
	CompAvg       float64 `json:"comp_avg"`       // 阵容平均名次
	Carry         string  `json:"carry"`          // 推荐给哪个 carry 使用
	PriorityScore int     `json:"priority_score"` // 100=最优先，85=次优先，以此类推
}

// ── localization.json ─────────────────────────────────────────────────────────

// LocalizationFile localization.json 顶层结构
type LocalizationFile struct {
	Source string            `json:"source"`   // "CommunityDragon/latest"
	IDToCN map[string]string `json:"id_to_cn"` // "TFT16_LeeSin" -> "盲僧"
	CNToID map[string]string `json:"cn_to_id"` // "盲僧" -> "TFT16_LeeSin"
}

// =============================================================================
// InputParser 输入解析
// =============================================================================

// UserInput 用户输入经 InputParser 标准化后的结果
type UserInput struct {
	Raw    string   `json:"raw"`    // 原始输入字符串，供 LLM 参考
	Heroes []string `json:"heroes"` // 标准化后的英雄 ID 列表
	Items  []string `json:"items"`  // 标准化后的装备 ID 列表
	// 无法识别的 token，交给 LLM 兜底解析
	Unknown []string `json:"unknown,omitempty"`
}

// =============================================================================
// Tool 层：输入输出结构体
// 每个 Tool 对应 Eino Graph 中的一个并行节点
// =============================================================================

// ── Tool 1: QueryHeroComps ────────────────────────────────────────────────────

// HeroCompsInput Tool1 输入：英雄列表
type HeroCompsInput struct {
	Heroes []string `json:"heroes"` // 标准化英雄 ID 列表
	TopN   int      `json:"top_n"`  // 返回前 N 个推荐，默认 5
}

// HeroCompsOutput Tool1 输出：英雄匹配阵容结果
type HeroCompsOutput struct {
	Matches []CompMatch `json:"matches"` // 匹配阵容列表，按 MatchScore 降序
}

// CompMatch 英雄与阵容的匹配详情
type CompMatch struct {
	Comp         Comp     `json:"comp"`          // 阵容完整信息
	MatchedUnits []string `json:"matched_units"` // 用户持有的、属于该阵容核心的英雄
	MissingUnits []string `json:"missing_units"` // 该阵容核心英雄中用户还缺少的
	MatchScore   float64  `json:"match_score"`   // 匹配度 0~1（命中数/核心英雄总数）
}

// ── Tool 2: QueryItemFit ──────────────────────────────────────────────────────

// ItemFitInput Tool2 输入：装备列表
type ItemFitInput struct {
	Items []string `json:"items"` // 标准化装备 ID 列表
}

// ItemFitOutput Tool2 输出：装备适配阵容结果
type ItemFitOutput struct {
	Results []ItemFitResult `json:"results"` // 适配结果，按 TotalScore 降序
}

// ItemFitResult 装备组合指向的阵容推荐（多个装备合并计分）
type ItemFitResult struct {
	ClusterID    string   `json:"cluster_id"`
	CompName     string   `json:"comp_name"`
	CompTier     string   `json:"comp_tier"`
	CompAvg      float64  `json:"comp_avg"`
	Carry        string   `json:"carry"`         // 建议将这些装备给哪个英雄
	MatchedItems []string `json:"matched_items"` // 本次命中的装备 ID 列表
	TotalScore   int      `json:"total_score"`   // 所有命中装备的 priority_score 之和
}

// ── Tool 3: QueryCompTier ─────────────────────────────────────────────────────

// CompTierInput Tool3 输入：阵容 ID 列表
type CompTierInput struct {
	ClusterIDs []string `json:"cluster_ids"` // 要查询强度的阵容 ID 列表
}

// CompTierOutput Tool3 输出：阵容强度信息
type CompTierOutput struct {
	Tiers []CompTierEntry `json:"tiers"`
}

// CompTierEntry 单个阵容的强度数据
type CompTierEntry struct {
	ClusterID    string  `json:"cluster_id"`
	Tier         string  `json:"tier"`
	AvgPlacement float64 `json:"avg_placement"`
	Top4Rate     float64 `json:"top4_rate"`
	WinRate      float64 `json:"win_rate"`
}

// =============================================================================
// IntersectionNode：交集计算节点
// 接收三个并行 Tool 的输出，计算交集，生成置信度排序的推荐列表
// =============================================================================

// IntersectionInput 三路 Tool 输出汇合的输入
type IntersectionInput struct {
	HeroComps HeroCompsOutput `json:"hero_comps"` // Tool1 结果
	ItemFits  ItemFitOutput   `json:"item_fits"`  // Tool2 结果
	CompTiers CompTierOutput  `json:"comp_tiers"` // Tool3 结果
	UserInput UserInput       `json:"user_input"` // 原始输入，供后续 LLM 使用
}

// IntersectionOutput 交集计算结果，传入 LLM 推理节点
type IntersectionOutput struct {
	Recommendations []Recommendation `json:"recommendations"` // 推荐列表，按置信度降序
	UserInput       UserInput        `json:"user_input"`      // 透传给 LLM
}

// Recommendation 单条推荐结果，包含置信度和所有命中信息
type Recommendation struct {
	Comp           Comp     `json:"comp"`            // 阵容完整信息（含 Top4Rate/WinRate 等）
	Confidence     float64  `json:"confidence"`      // 置信度 0~1
	ConfidenceDesc string   `json:"confidence_desc"` // "高" / "中" / "低"
	HitSources     []string `json:"hit_sources"`     // 命中来源，如 ["hero", "item", "tier"]
	MatchedUnits   []string `json:"matched_units"`   // 已有的核心英雄（中文名）
	MissingUnits   []string `json:"missing_units"`   // 缺少的核心英雄（中文名）
	MatchedItems   []string `json:"matched_items"`   // 命中的装备（中文名）
	SuggestedCarry string   `json:"suggested_carry"` // 建议的 carry 英雄（中文名）
	Top4Rate       float64  `json:"top4_rate"`
}

// ConfidenceDesc 根据置信度数值返回描述文字
func (r *Recommendation) CalcConfidenceDesc() string {
	switch {
	case r.Confidence >= 0.7:
		return "高"
	case r.Confidence >= 0.4:
		return "中"
	default:
		return "低"
	}
}

// HitCount 命中来源数量，用于置信度计算
func (r *Recommendation) HitCount() int {
	return len(r.HitSources)
}
