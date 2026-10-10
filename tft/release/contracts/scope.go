package contracts

import (
	"regexp"
	"time"
)

// Game 固定值：当前仓库只服务云顶之弈，禁止发明其他游戏域。
const Game = "tft"

// Region/Queue 的已知值。unknown 是显式声明"不知道"，
// 不是编造（ADR013：未知统计地区保留 unknown，不能编造）。
var knownRegions = map[string]bool{
	"cn": true, "na": true, "eu": true, "kr": true,
	"unknown": true,
}

var knownQueues = map[string]bool{
	"ranked": true, "normal": true, "unknown": true,
}

// GamePatchPattern 补丁号格式：MAJOR.MINOR，可选单字母热修后缀（如 18.1a）。
// 这是补丁真相源的格式约束；热修是否生效由 HotfixRevision 表达。
var GamePatchPattern = regexp.MustCompile(`^\d+\.\d+[a-z]?$`)

// HotfixRevision 允许为空（表示无热修），但非空时必须形如 "a"、"b"。
var hotfixPattern = regexp.MustCompile(`^[a-z]$`)

// LocalePattern BCP-47 简化形式（zh-CN / en-US）。
var LocalePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z]{2,8})*$`)

// DataScope 描述一份数据的游戏适用范围。
//
// 语义要点（ADR013）：
//   - locale 是语言（zh-CN），不代表统计地区；统计地区在 SourceRecord.StatsRegion；
//   - region/queue 允许 "unknown"，表示显式未知，不能编造；
//   - hotfix_revision 为空 = 无热修；非空 = 该热修已包含在数据中；
//   - effective_at 是补丁在 region 生效时间，与统计窗口无关。
type DataScope struct {
	Game           string    `json:"game" yaml:"game"`
	SetID          int       `json:"set_id" yaml:"set_id"`
	GamePatch      string    `json:"game_patch" yaml:"game_patch"`
	HotfixRevision string    `json:"hotfix_revision,omitempty" yaml:"hotfix_revision,omitempty"`
	Region         string    `json:"region" yaml:"region"`
	Queue          string    `json:"queue" yaml:"queue"`
	Locale         string    `json:"locale" yaml:"locale"`
	EffectiveAt    time.Time `json:"effective_at" yaml:"effective_at"`
}

// Validate 校验 scope。所有非法情况返回 ErrCategoryScope。
//
// 注意：game_patch 必须显式给出。禁止从 set_id 推导、禁止从公告标题
// 盲猜（调用方责任）；本函数至少保证它非空且格式合法。
func (s *DataScope) Validate() error {
	if s == nil {
		return newErr(ErrCategoryScope, "scope", "", "scope 为 nil")
	}
	if s.Game != Game {
		return newErr(ErrCategoryScope, "game", "", "game 必须为 %q，got %q", Game, s.Game)
	}
	if s.SetID <= 0 {
		return newErr(ErrCategoryScope, "set_id", "", "set_id 必须为正整数，got %d", s.SetID)
	}
	if s.GamePatch == "" {
		return newErr(ErrCategoryScope, "game_patch", "", "game_patch 不能为空（禁止从赛季或公告标题推导）")
	}
	if !GamePatchPattern.MatchString(s.GamePatch) {
		return newErr(ErrCategoryScope, "game_patch", "", "game_patch 格式非法（要求 MAJOR.MINOR[letter]）：%q", s.GamePatch)
	}
	if s.HotfixRevision != "" && !hotfixPattern.MatchString(s.HotfixRevision) {
		return newErr(ErrCategoryScope, "hotfix_revision", "", "hotfix_revision 非空时必须为单字母：%q", s.HotfixRevision)
	}
	if !knownRegions[s.Region] {
		return newErr(ErrCategoryScope, "region", "", "region 未知（允许 %q / 具体地区代码）：%q", "unknown", s.Region)
	}
	if !knownQueues[s.Queue] {
		return newErr(ErrCategoryScope, "queue", "", "queue 未知（允许 ranked/normal/unknown）：%q", s.Queue)
	}
	if s.Locale == "" || !LocalePattern.MatchString(s.Locale) {
		return newErr(ErrCategoryScope, "locale", "", "locale 必须为 BCP-47 形式（如 zh-CN）：%q", s.Locale)
	}
	if s.EffectiveAt.IsZero() {
		return newErr(ErrCategoryScope, "effective_at", "", "effective_at 不能为零值（补丁生效时间必须显式给出）")
	}
	return nil
}

// SamePatchInRegion 判断两个 scope 是否对同一 region 的同一补丁有效
//（热修不同视为不同补丁）。用于发布/路由层的"同补丁灰度"判断。
func (s *DataScope) SamePatchInRegion(other *DataScope) bool {
	if s == nil || other == nil {
		return false
	}
	return s.Game == other.Game &&
		s.SetID == other.SetID &&
		s.GamePatch == other.GamePatch &&
		s.HotfixRevision == other.HotfixRevision &&
		s.Region == other.Region
}
