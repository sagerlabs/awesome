package contracts

import "time"

// SamplePolicy 统一统计资格（ADR013）：主推荐与 Meta 补充不能执行
// 互相矛盾的样本过滤；低样本事实可保留展示，是否可进入排名由此声明。
type SamplePolicy struct {
	// MinSampleFloor 进入排名/推荐的最小样本门槛（对局数）。
	MinSampleFloor int `json:"min_sample_floor" yaml:"min_sample_floor"`
	// BelowFloorAction 低于门槛时的行为：degrade_basic（降级为基础事实，
	// 标注积累中）/ reject（不返回该统计）。
	BelowFloorAction string `json:"below_floor_action" yaml:"below_floor_action"`
	// Note 面向用户的不确定性说明（如"新补丁统计积累中"）。
	Note string `json:"note,omitempty" yaml:"note,omitempty"`
}

// Freshness 描述证据的新鲜度，四个时间不得互换语义（见包文档）。
type Freshness struct {
	FetchedAt       time.Time `json:"fetched_at" yaml:"fetched_at"`
	StatsWindowEnd  time.Time `json:"stats_window_end" yaml:"stats_window_end"`
	EffectiveAt     time.Time `json:"effective_at" yaml:"effective_at"`
	BuiltAt         time.Time `json:"built_at" yaml:"built_at"`
}

// EvidenceEnvelope 是回答、日志、反馈与评测共享的证据信封（ADR013）。
//
// 每个回答携带版本与来源身份；样本按事实记录，不把不同阵容的
// Count 相加当作去重对局数。LLM 只整理证据与解释建议，
// 不补齐缺失事实（缺失时用 Warnings 表达）。
type EvidenceEnvelope struct {
	TraceID       string        `json:"trace_id" yaml:"trace_id"`
	ReleaseID     string        `json:"release_id" yaml:"release_id"`
	Scope         DataScope     `json:"scope" yaml:"scope"`
	SourceIDs     []string      `json:"source_ids" yaml:"source_ids"`
	EvidenceIDs   []string      `json:"evidence_ids,omitempty" yaml:"evidence_ids,omitempty"`
	Freshness     Freshness     `json:"freshness" yaml:"freshness"`
	SamplePolicy  SamplePolicy  `json:"sample_policy" yaml:"sample_policy"`
	Warnings      []string      `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	Capabilities  []string      `json:"capabilities" yaml:"capabilities"`
}

// Validate 校验证据信封（作为契约样例与测试使用；
// CORE-02 接管线上组装）。
func (e *EvidenceEnvelope) Validate() error {
	if e == nil {
		return newErr(ErrCategoryFormat, "evidence", "", "evidence 为 nil")
	}
	if e.TraceID == "" {
		return newErr(ErrCategoryIdentity, "trace_id", "", "trace_id 不能为空")
	}
	if e.ReleaseID == "" {
		return newErr(ErrCategoryIdentity, "release_id", "", "release_id 不能为空")
	}
	if err := e.Scope.Validate(); err != nil {
		return err
	}
	if len(e.SourceIDs) == 0 {
		return newErr(ErrCategoryIdentity, "source_ids", "", "source_ids 不能为空（证据必须可溯源）")
	}
	if len(e.Capabilities) == 0 {
		return newErr(ErrCategoryCapability, "capabilities", "", "capabilities 不能为空")
	}
	for _, c := range e.Capabilities {
		if !KnownCapabilities[c] {
			return newErr(ErrCategoryCapability, "capabilities", "", "未知能力：%q", c)
		}
	}
	switch e.SamplePolicy.BelowFloorAction {
	case "degrade_basic", "reject":
	default:
		return newErr(ErrCategoryFormat, "below_floor_action", "",
			"必须为 degrade_basic 或 reject：%q", e.SamplePolicy.BelowFloorAction)
	}
	// 时间语义：fetched <= built；window_end <= fetched；effective_at 独立但必须显式
	if e.Freshness.EffectiveAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "effective_at", "",
			"freshness.effective_at 为零值（补丁生效时间必须显式给出，与 scope 同规则）")
	}
	if e.Freshness.FetchedAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "fetched_at", "", "freshness.fetched_at 为零值")
	}
	if e.Freshness.BuiltAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "built_at", "", "freshness.built_at 为零值")
	}
	if e.Freshness.FetchedAt.After(e.Freshness.BuiltAt) {
		return newErr(ErrCategoryTimeOrder, "freshness", "", "fetched_at 晚于 built_at")
	}
	if e.Freshness.StatsWindowEnd.IsZero() {
		return newErr(ErrCategoryTimeOrder, "stats_window_end", "", "freshness.stats_window_end 为零值")
	}
	if e.Freshness.StatsWindowEnd.After(e.Freshness.FetchedAt) {
		return newErr(ErrCategoryTimeOrder, "freshness", "", "stats_window_end 晚于 fetched_at")
	}
	return nil
}
