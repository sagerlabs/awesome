# 可观测性架构设计方案

## 现状分析

### 现有能力
1. **基础链路追踪**（trace.go）
   - 节点级耗时统计
   - 开始/结束/错误事件日志
   - 流式输出token chunk统计

2. **日志系统**（logrus）
   - 结构化日志
   - 多级别日志（Debug/Info/Warn/Error）

### 缺失能力
1. Token消耗统计（输入/输出/总成本）
2. 请求级全链路追踪
3. Metrics指标暴露
4. 分布式追踪（OpenTelemetry）
5. 性能分析（pprof）

---

## 方案设计

### 1. Token消耗统计

#### 1.1 数据结构
```go
type TokenUsage struct {
    RequestID    string        // 请求ID
    Model        string        // 使用的模型
    InputTokens  int           // 输入token数
    OutputTokens int           // 输出token数
    TotalTokens  int           // 总token数
    CostUSD      float64       // 预估成本（美元）
    Duration     time.Duration // 请求耗时
    NodeUsages   []NodeUsage   // 各节点token使用情况
}

type NodeUsage struct {
    NodeName     string // 节点名称
    InputTokens  int    // 节点输入token数
    OutputTokens int    // 节点输出token数
}
```

#### 1.2 实现位置
- **agent/model.go** - 在ChatModel调用时统计token
- **agent/agent.go** - 在Invoke/Stream调用时收集token数据
- **tft/handler.go** - 在HTTP handler层汇总并记录

#### 1.3 Token计算方式
- OpenAI/DeepSeek: 从API响应的`usage`字段获取
- Ark: 从API响应的`usage`字段获取
- 兜底方案：使用`tiktoken`或近似计算（中文1字符≈1token，英文4字符≈1token）

### 2. 全链路追踪

#### 2.1 Request ID注入
在HTTP handler层生成Request ID，通过context传递：
```go
func (h *Handler) withRequestID(c *gin.Context) context.Context {
    requestID := c.GetHeader("X-Request-ID")
    if requestID == "" {
        requestID = uuid.NewString()
    }
    return context.WithValue(c.Request.Context(), requestIDKey{}, requestID)
}
```

#### 2.2 结构化日志增强
所有日志必须包含：
- `request_id` - 请求ID
- `user_id` - 用户ID（如果有）
- `node` - 节点名称（如果在节点内）
- `elapsed` - 耗时
- `token_usage` - token使用情况

### 3. Metrics指标

#### 3.1 Prometheus指标
使用`prometheus/client_golang`暴露以下指标：

```go
// 请求指标
tft_requests_total{method, endpoint, status} // 总请求数
tft_request_duration_seconds{endpoint}        // 请求耗时分布
tft_request_tokens_total{endpoint, model}     // token消耗总数

// LLM指标
tft_llm_calls_total{model, node}              // LLM调用次数
tft_llm_tokens_total{model, type}              // type: input/output
tft_llm_duration_seconds{model, node}         // LLM调用耗时

// 系统指标
tft_active_requests                            // 活跃请求数
tft_queue_length                               // 队列长度
```

#### 3.2 实现位置
- 新建 `tft/metrics/metrics.go` - 指标定义和注册
- 在 `handler.go` 中注入metrics middleware
- 在 `agent.go` 和 `model.go` 中记录LLM指标

### 4. 分布式追踪（OpenTelemetry）

#### 4.1 集成方案
- 使用OpenTelemetry Go SDK
- 支持导出到Jaeger/Zipkin/Otlp
- Trace ID通过HTTP header传递

#### 4.2 Span设计
```
HTTP Request (span)
├── NLU Extract (span)
│   └── LLM Call (span, with token attributes)
├── Data Lookup (span)
└── LLM Refine (span, with token attributes)
    └── LLM Stream (span)
```

### 5. 健康检查与Profiling

#### 5.1 健康检查端点
- `GET /health` - 基础健康检查
- `GET /health/ready` - 就绪检查（数据加载完成）
- `GET /health/live` - 存活检查

#### 5.2 pprof集成
- `GET /debug/pprof/` - Go标准pprof端点
- 可按需开启，生产环境默认关闭

---

## 实施计划

### Phase 1: Token统计（优先级：高）
1. 实现TokenUsage数据结构
2. 在model.go中集成token统计
3. 在handler.go中记录token使用日志
4. 添加环境变量控制是否开启详细token日志

### Phase 2: Metrics（优先级：高）
1. 集成Prometheus client
2. 定义核心metrics
3. 添加`/metrics`端点
4. 编写Grafana dashboard示例

### Phase 3: 链路追踪增强（优先级：中）
1. 实现Request ID传递
2. 增强结构化日志
3. 实现节点级token统计
4. 添加trace ID到所有日志

### Phase 4: OpenTelemetry（优先级：低）
1. 集成OpenTelemetry SDK
2. 实现Span创建和传播
3. 添加token usage到span attributes
4. 支持导出到Jaeger

---

## 成本计算参考

| 模型 | 输入价格 | 输出价格 |
|------|---------|---------|
| GPT-4o-mini | $0.15/1M tokens | $0.60/1M tokens |
| DeepSeek-Chat | $0.001/1M tokens | $0.002/1M tokens |
| Doubao-Pro | ¥0.008/1K tokens | ¥0.02/1K tokens |

---

## 附录：快速实现建议

对于MVP版本，可以先实现：
1. ✅ Token统计（从API response获取usage）
2. ✅ Request ID + 结构化日志
3. ✅ 基础Prometheus metrics（请求数、耗时、token总数）

更高级的功能可以后续迭代。
