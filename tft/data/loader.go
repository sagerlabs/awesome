package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Store 全局数据仓库，持有所有 JSON 数据并提供查询方法
// 启动时加载一次，之后只读，天然线程安全
type Store struct {
	// 原始数据
	comps    []Comp                    // 所有阵容列表
	items    map[string][]ItemFitEntry // 装备 -> 阵容适配列表
	localize LocalizationFile          // 汉化映射

	// 查询索引（加载时构建，避免每次查询遍历）
	compByClusterID map[string]*Comp   // cluster_id -> Comp
	compsByUnit     map[string][]*Comp // unit_id    -> 包含该英雄的阵容列表
	compsByTier     map[string][]*Comp // tier       -> 该 Tier 的阵容列表

	once sync.Once
}

// GetDataDir 数据目录路径，可通过环境变量 TFT_DATA_DIR 覆盖
func GetDataDir() string {
	if dir := os.Getenv("TFT_DATA_DIR"); dir != "" {
		return dir
	}
	// 默认：项目根目录下的 metadata/tft-meta/data/
	return filepath.Join("metadata", "tft-meta", "data")
}

// NewStore 加载所有数据文件，构建查询索引
func NewStore(dataDir string) (*Store, error) {
	s := &Store{}
	if err := s.load(dataDir); err != nil {
		return nil, err
	}
	return s, nil
}

// ── 加载 ──────────────────────────────────────────────────────────────────────

func (s *Store) load(dataDir string) error {
	// 1. 加载阵容数据
	if err := s.loadComps(filepath.Join(dataDir, "comps_for_agent.json")); err != nil {
		return fmt.Errorf("load comps: %w", err)
	}

	// 2. 加载装备优先级数据
	if err := s.loadItems(filepath.Join(dataDir, "items_priority.json")); err != nil {
		return fmt.Errorf("load items: %w", err)
	}

	// 3. 加载汉化表（失败不阻塞，降级为 ID 原文）
	if err := s.loadLocalization(filepath.Join(dataDir, "localization.json")); err != nil {
		s.localize = LocalizationFile{
			IDToCN: make(map[string]string),
			CNToID: make(map[string]string),
		}
	}

	// 4. 构建查询索引
	s.buildIndexes()

	return nil
}

func (s *Store) loadComps(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f CompsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	s.comps = f.Comps
	return nil
}

func (s *Store) loadItems(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f ItemsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	s.items = f
	return nil
}

func (s *Store) loadLocalization(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.localize)
}

func (s *Store) buildIndexes() {
	s.compByClusterID = make(map[string]*Comp, len(s.comps))
	s.compsByUnit = make(map[string][]*Comp)
	s.compsByTier = make(map[string][]*Comp)

	for i := range s.comps {
		c := &s.comps[i]

		// cluster_id 索引
		s.compByClusterID[c.ClusterID] = c

		// tier 索引
		s.compsByTier[c.Tier] = append(s.compsByTier[c.Tier], c)

		// unit 索引：每个核心英雄都指向这个阵容
		for _, unit := range c.Units {
			s.compsByUnit[unit] = append(s.compsByUnit[unit], c)
		}
	}
}

// ── 阵容查询 ──────────────────────────────────────────────────────────────────

// GetCompByClusterID 通过 cluster_id 精确查询阵容
func (s *Store) GetCompByClusterID(clusterID string) (*Comp, bool) {
	c, ok := s.compByClusterID[clusterID]
	return c, ok
}

// GetCompsByUnits 查询包含指定英雄的阵容
// 返回每个阵容命中了几个英雄，按命中数降序
func (s *Store) GetCompsByUnits(unitIDs []string) []CompMatch {
	// 统计每个阵容命中的英雄数
	hitCount := make(map[string]int)      // clusterID -> 命中数
	hitUnits := make(map[string][]string) // clusterID -> 命中的英雄列表

	for _, uid := range unitIDs {
		for _, comp := range s.compsByUnit[uid] {
			hitCount[comp.ClusterID]++
			hitUnits[comp.ClusterID] = append(hitUnits[comp.ClusterID], uid)
		}
	}

	// 构建 CompMatch 列表
	var matches []CompMatch
	for clusterID, count := range hitCount {
		comp := s.compByClusterID[clusterID]
		if comp == nil {
			continue
		}

		matched := hitUnits[clusterID]
		missing := missingUnits(comp.Units, matched)

		// 匹配度 = 命中数 / 核心英雄总数
		matchScore := float64(count) / float64(len(comp.Units))

		matches = append(matches, CompMatch{
			Comp:         *comp,
			MatchedUnits: matched,
			MissingUnits: missing,
			MatchScore:   matchScore,
		})
	}

	// 按匹配度降序排序
	sortCompMatches(matches)
	return matches
}

// GetCompsByTier 查询指定 Tier 的所有阵容
func (s *Store) GetCompsByTier(tiers ...string) []*Comp {
	var result []*Comp
	for _, tier := range tiers {
		result = append(result, s.compsByTier[tier]...)
	}
	return result
}

// AllComps 返回所有阵容（按 avg_placement 升序，即越强越靠前）
func (s *Store) AllComps() []Comp {
	return s.comps
}

// GetTopCompsByUnit 查询包含指定英雄的顶级阵容
// 返回该英雄的 S/A 级阵容，按 avg_placement 升序（越强越靠前）
func (s *Store) GetTopCompsByUnit(unitName string) []*Comp {
	unitID := s.ResolveUnitID(unitName)
	if unitID == "" {
		return nil
	}

	var result []*Comp
	for _, comp := range s.compsByUnit[unitID] {
		if comp.Tier == "S" || comp.Tier == "A" {
			result = append(result, comp)
		}
	}

	// 按 avg_placement 升序（越低越强）
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].AvgPlacement < result[j-1].AvgPlacement; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result
}

// GetBestItemsForUnit 查询特定阵容下某个英雄的最佳装备
// 返回该英雄的最优装备方案（BuildInfo）
func (s *Store) GetBestItemsForUnit(clusterID string, unitName string) *BuildInfo {
	comp, ok := s.GetCompByClusterID(clusterID)
	if !ok {
		return nil
	}

	unitID := s.ResolveUnitID(unitName)
	if unitID == "" {
		// 如果找不到unitID，尝试直接匹配中文名
		for _, build := range comp.AllBuilds {
			if s.IDToCN(build.Carry) == unitName || build.Carry == unitName {
				return &build
			}
		}
		return nil
	}

	// 优先找 BestBuild
	if comp.BestBuild.Carry == unitID || s.IDToCN(comp.BestBuild.Carry) == unitName {
		return &comp.BestBuild
	}

	// 再找 AllBuilds
	for _, build := range comp.AllBuilds {
		if build.Carry == unitID || s.IDToCN(build.Carry) == unitName {
			return &build
		}
	}

	return nil
}

// GetCompStrategy 查询阵容的运营策略
// 返回该阵容的升级节奏（Levelling）、推荐升3星的英雄（Stars）等信息
type CompStrategy struct {
	Levelling string   `json:"levelling"` // 推荐升级节点，如 "Fast 8", "lvl 7"
	Stars     []string `json:"stars"`     // 推荐升3星的英雄（中文名）
	Difficulty float64  `json:"difficulty"` // 操作难度
}

func (s *Store) GetCompStrategy(clusterID string) *CompStrategy {
	comp, ok := s.GetCompByClusterID(clusterID)
	if !ok {
		return nil
	}

	// 把 Stars 里的英雄ID转成中文名
	starsCN := make([]string, len(comp.Stars))
	for i, unitID := range comp.Stars {
		starsCN[i] = s.IDToCN(unitID)
	}

	return &CompStrategy{
		Levelling: comp.Levelling,
		Stars:     starsCN,
		Difficulty: comp.Difficulty,
	}
}

// ── 装备查询 ──────────────────────────────────────────────────────────────────

// GetItemFitEntries 查询装备适配的阵容列表
func (s *Store) GetItemFitEntries(itemID string) []ItemFitEntry {
	return s.items[itemID]
}

// GetItemFitByItems 批量装备查询，聚合多个装备指向的阵容
// 同一阵容被多个装备命中时，合并为一条结果，分数累加
func (s *Store) GetItemFitByItems(itemIDs []string) []ItemFitResult {
	// clusterID -> 聚合结果
	merged := make(map[string]*ItemFitResult)

	for _, itemID := range itemIDs {
		for _, entry := range s.items[itemID] {
			key := entry.ClusterID + "|" + entry.Carry // 同阵容同 carry 合并
			if r, exists := merged[key]; exists {
				r.MatchedItems = append(r.MatchedItems, itemID)
				r.TotalScore += entry.PriorityScore
			} else {
				merged[key] = &ItemFitResult{
					ClusterID:    entry.ClusterID,
					CompName:     entry.CompName,
					CompTier:     entry.CompTier,
					CompAvg:      entry.CompAvg,
					Carry:        entry.Carry,
					MatchedItems: []string{itemID},
					TotalScore:   entry.PriorityScore,
				}
			}
		}
	}

	// 转成列表，按 TotalScore 降序
	var results []ItemFitResult
	for _, r := range merged {
		results = append(results, *r)
	}
	sortItemFitResults(results)
	return results
}

// ── 汉化查询 ──────────────────────────────────────────────────────────────────

// IDToCN 将 TFT ID 转换为中文名，找不到时返回原 ID
func (s *Store) IDToCN(id string) string {
	if cn, ok := s.localize.IDToCN[id]; ok {
		return cn
	}
	return id
}

// CNToID 将中文名转换为 TFT ID，找不到时返回原字符串
func (s *Store) CNToID(cn string) string {
	if id, ok := s.localize.CNToID[cn]; ok {
		return id
	}
	return cn
}

// ResolveUnitID 解析英雄输入：支持中文名、ID、模糊匹配
// 返回标准 TFT ID，找不到返回空字符串
func (s *Store) ResolveUnitID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 1. 直接是 TFT ID
	if strings.HasPrefix(input, "TFT") {
		if _, ok := s.compsByUnit[input]; ok {
			return input
		}
	}

	// 2. 中文名精确匹配
	if id := s.CNToID(input); id != input {
		return id
	}

	// 3. 中文名模糊匹配（包含关系）
	for cn, id := range s.localize.CNToID {
		if strings.Contains(cn, input) || strings.Contains(input, cn) {
			return id
		}
	}

	return ""
}

// ResolveItemID 解析装备输入：支持中文名、ID、模糊匹配
func (s *Store) ResolveItemID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// 1. 直接是 TFT ID
	if strings.HasPrefix(input, "TFT_Item_") {
		if _, ok := s.items[input]; ok {
			return input
		}
	}

	// 2. 中文名精确匹配
	if id := s.CNToID(input); id != input {
		if _, ok := s.items[id]; ok {
			return id
		}
	}

	// 3. 中文名模糊匹配
	for cn, id := range s.localize.CNToID {
		if !strings.HasPrefix(id, "TFT_Item_") {
			continue
		}
		if strings.Contains(cn, input) || strings.Contains(input, cn) {
			return id
		}
	}

	return ""
}

// ── 工具函数 ──────────────────────────────────────────────────────────────────

// missingUnits 计算 required 中不在 owned 里的元素
func missingUnits(required, owned []string) []string {
	ownedSet := make(map[string]struct{}, len(owned))
	for _, u := range owned {
		ownedSet[u] = struct{}{}
	}
	var missing []string
	for _, u := range required {
		if _, ok := ownedSet[u]; !ok {
			missing = append(missing, u)
		}
	}
	return missing
}

func sortCompMatches(matches []CompMatch) {
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0 && matches[j].MatchScore > matches[j-1].MatchScore; j-- {
			matches[j], matches[j-1] = matches[j-1], matches[j]
		}
	}
}

func sortItemFitResults(results []ItemFitResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].TotalScore > results[j-1].TotalScore; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

func NewStoreFromRaw(comps []Comp, items ItemsFile, loc LocalizationFile) *Store {
	s := &Store{
		comps:    comps,
		items:    items,
		localize: loc,
	}
	s.buildIndexes()
	return s
}
