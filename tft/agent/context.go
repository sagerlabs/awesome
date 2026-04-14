package agent

import "github.com/sagerlabs/awesome/tft/knowledge/contracts"

// Context 是 agent 侧对共享 QueryNLURequest 的别名，保留现有调用习惯。
type Context = contracts.QueryNLURequest
