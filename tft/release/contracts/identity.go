package contracts

import "time"

// SessionRecord 描述一个会话的所有权与版本绑定（ADR013）。
//
// 对话每轮使用 CAS 或串行化（state_revision 单调递增），
// 防止并发回答乱序覆盖；assigned_release 固定后可因撤回/补丁
// 变化失效，由路由层显式迁移。
type SessionRecord struct {
	OwnerID          string            `json:"owner_id" yaml:"owner_id"`
	SessionID        string            `json:"session_id" yaml:"session_id"`
	StateRevision    int64             `json:"state_revision" yaml:"state_revision"`
	Scope            DataScope         `json:"scope" yaml:"scope"`
	AssignedRelease  string            `json:"assigned_release" yaml:"assigned_release"`
	ExperimentID     string            `json:"experiment_id,omitempty" yaml:"experiment_id,omitempty"`
	Context          map[string]string `json:"context,omitempty" yaml:"context,omitempty"`
	ExpiresAt        time.Time         `json:"expires_at" yaml:"expires_at"`
}

// Validate 校验会话记录。
func (s *SessionRecord) Validate() error {
	if s == nil {
		return newErr(ErrCategoryFormat, "session", "", "session 为 nil")
	}
	if s.OwnerID == "" {
		return newErr(ErrCategoryIdentity, "owner_id", "", "owner_id 不能为空（会话必须绑定所有权）")
	}
	if s.SessionID == "" {
		return newErr(ErrCategoryIdentity, "session_id", "", "session_id 不能为空")
	}
	if s.StateRevision < 0 {
		return newErr(ErrCategoryFormat, "state_revision", "", "state_revision 不能为负")
	}
	if s.AssignedRelease == "" {
		return newErr(ErrCategoryIdentity, "assigned_release", "", "assigned_release 不能为空")
	}
	if err := s.Scope.Validate(); err != nil {
		return err
	}
	if s.ExpiresAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "expires_at", "", "expires_at 为零值")
	}
	return nil
}

// RouteDecision 是服务端签发的路由决策（ADR013）。
//
// 由服务端决定；客户端只能回传签发结果，不能决定目标实例或
// 任意数据包。expires_at 过期后必须重新请求，不得缓存复用。
type RouteDecision struct {
	RequestID      string    `json:"request_id" yaml:"request_id"`
	SubjectID      string    `json:"subject_id" yaml:"subject_id"`
	SessionID      string    `json:"session_id" yaml:"session_id"`
	ReleaseID      string    `json:"release_id" yaml:"release_id"`
	PolicyRevision int64     `json:"policy_revision" yaml:"policy_revision"`
	BackendID      string    `json:"backend_id" yaml:"backend_id"`
	ExpiresAt      time.Time `json:"expires_at" yaml:"expires_at"`
}

// Validate 校验路由决策。
func (r *RouteDecision) Validate() error {
	if r == nil {
		return newErr(ErrCategoryFormat, "route_decision", "", "route_decision 为 nil")
	}
	for field, v := range map[string]string{
		"request_id": r.RequestID,
		"subject_id": r.SubjectID,
		"session_id": r.SessionID,
		"release_id": r.ReleaseID,
		"backend_id": r.BackendID,
	} {
		if v == "" {
			return newErr(ErrCategoryIdentity, field, "", "%s 不能为空", field)
		}
	}
	if r.PolicyRevision <= 0 {
		return newErr(ErrCategoryFormat, "policy_revision", "", "policy_revision 必须为正")
	}
	if r.ExpiresAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "expires_at", "", "expires_at 为零值")
	}
	return nil
}
