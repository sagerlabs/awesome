# 可观测性功能实施计划

## Phase 1: Token统计（MVP）

### 任务列表
- [ ] 创建 `tft/agent/token_usage.go` - TokenUsage数据结构
- [ ] 修改 `tft/agent/model.go` - 从LLM响应中提取token usage
- [ ] 修改 `tft/agent/agent.go` - 收集并传递token usage
- [ ] 修改 `tft/handler.go` - 记录token usage日志
- [ ] 添加环境变量 `ENABLE_TOKEN_LOGGING` 控制开关

### 预计工作量
- 2-3小时

---

## Phase 2: Metrics

### 任务列表
- [ ] 创建 `tft/metrics/metrics.go` - Prometheus metrics定义
- [ ] 修改 `tft/handler.go` - 添加metrics middleware
- [ ] 修改 `tft/agent/model.go` - 记录LLM metrics
- [ ] 添加 `/metrics` 端点
- [ ] 创建 `grafana/dashboard.json` - Grafana仪表盘示例

### 预计工作量
- 3-4小时

---

## Phase 3: 链路追踪增强

### 任务列表
- [ ] 创建 `tft/trace/request_id.go` - Request ID管理
- [ ] 修改 `tft/handler.go` - 注入Request ID
- [ ] 修改所有日志 - 添加request_id字段
- [ ] 修改 `tft/agent/trace.go` - 增强节点追踪

### 预计工作量
- 2-3小时
