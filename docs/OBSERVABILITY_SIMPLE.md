# 可观测性简易实施方案（学习版）

## 核心理念

对于学习AI Agent的人来说，可观测性的核心价值是：
1. **能追踪**：从HTTP请求进来，到内部LLM调用，能一路追踪到底
2. **好调试**：出问题时，通过trace_id能快速定位到是哪个环节出了问题
3. **易扩展**：为以后接入Prometheus、OpenTelemetry等高级工具打好基础

---

## 三层架构设计

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: HTTP Handler层                                      │
│  - 生成trace_id                                              │
│  - 注入到context                                             │
│  - 记录请求开始/结束日志                                      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Agent层                                             │
│  - 从context获取trace_id                                      │
│  - 传递给各节点                                               │
│  - 记录各节点执行情况                                          │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Model层                                             │
│  - 记录LLM调用详情                                            │
│  - 统计token使用情况                                          │
│  - 记录耗时和成本                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 具体实现步骤

### Step 1: Trace ID管理

**文件**: `tft/trace/trace.go`

```go
package trace

import (
	"context"
	"github.com/google/uuid"
)

type traceIDKey struct{}

// NewTraceID 生成新的trace ID
func NewTraceID() string {
	return uuid.NewString()
}

// WithTraceID 将trace ID存入context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDFromContext 从context获取trace ID
func TraceIDFromContext(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey{}).(string)
	return traceID, ok
}

// TraceIDOrNew 从context获取trace ID，如果没有则生成新的
func TraceIDOrNew(ctx context.Context) (context.Context, string) {
	if traceID, ok := TraceIDFromContext(ctx); ok {
		return ctx, traceID
	}
	traceID := NewTraceID()
	return WithTraceID(ctx, traceID), traceID
}
```

### Step 2: 日志增强

**修改所有日志**，确保每条日志都包含`trace_id`字段：

```go
// 之前
logrus.WithField("input", input).Info("开始分析")

// 之后
logrus.WithFields(logrus.Fields{
    "trace_id": traceID,
    "input":    input,
}).Info("开始分析")
```

### Step 3: Handler层集成

**修改** `tft/handler.go`:

```go
// 在每个handler开头添加
func (h *Handler) Analyze(c *gin.Context) {
    // 1. 生成或获取trace_id
    traceID := c.GetHeader("X-Trace-ID")
    if traceID == "" {
        traceID = trace.NewTraceID()
    }
    
    // 2. 注入到context
    ctx := trace.WithTraceID(c.Request.Context(), traceID)
    
    // 3. 记录请求开始
    h.logger.WithFields(logrus.Fields{
        "trace_id": traceID,
        "endpoint": "/v1/tft/analyze",
        "input":    req.Input,
    }).Info("请求开始")
    
    // 4. 后续处理都使用这个ctx
    output, err := h.ag.Analyze(ctx, req.Input)
    
    // 5. 记录请求结束
    h.logger.WithFields(logrus.Fields{
        "trace_id": traceID,
        "elapsed":  time.Since(start).String(),
        "success":  err == nil,
    }).Info("请求结束")
}
```

### Step 4: Agent层传递

**修改** `tft/agent/agent.go`:

```go
func (a *Agent) Analyze(ctx context.Context, rawInput string) (*GraphOutput, error) {
    // 从context获取trace_id
    traceID, _ := trace.TraceIDFromContext(ctx)
    
    a.logger.WithFields(logrus.Fields{
        "trace_id": traceID,
        "input":    rawInput,
    }).Debug("开始推理")
    
    // 传递opts时包含trace相关的callback
    opts := append(a.traceOpts, ...)
    
    // ... 其余代码
}
```

### Step 5: Trace Callback增强

**修改** `tft/agent/trace.go`:

```go
func NewTraceCallback(logger *logrus.Logger) callbacks.Handler {
    return callbacks.NewHandlerBuilder().
        OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            traceID, _ := trace.TraceIDFromContext(ctx)
            logger.WithFields(logrus.Fields{
                "trace_id":  traceID,
                "node":      info.Name,
                "component": info.Type,
            }).Debug("→ 节点开始")
            return ctx
        }).
        OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            traceID, _ := trace.TraceIDFromContext(ctx)
            logger.WithFields(logrus.Fields{
                "trace_id":  traceID,
                "node":      info.Name,
                "component": info.Type,
                "elapsed":   elapsed.Round(time.Millisecond).String(),
            }).Info("✓ 节点完成")
            return ctx
        }).
        Build()
}
```

---

## 使用效果

### 日志示例

```
INFO[0001] 请求开始     trace_id=abc-123 endpoint=/v1/tft/nlu/stream input="我有日炎斗篷给谁呢"
DEBUG[0002] → 节点开始  trace_id=abc-123 node=nlu_extract component=lambda
INFO[0005] ✓ 节点完成  trace_id=abc-123 node=nlu_extract component=lambda elapsed=321ms
DEBUG[0006] → 节点开始  trace_id=abc-123 node=data_lookup component=lambda
INFO[0008] ✓ 节点完成  trace_id=abc-123 node=data_lookup component=lambda elapsed=123ms
DEBUG[0009] → 节点开始  trace_id=abc-123 node=llm_refine component=chat_model
INFO[0055] ✓ 节点完成  trace_id=abc-123 node=llm_refine component=chat_model elapsed=46.123s
INFO[0056] 请求结束   trace_id=abc-123 elapsed=55.123s success=true
```

### 调试流程

1. 发现问题：某条请求响应很慢
2. 查找日志：用`grep "请求结束" | grep "success=false"`找到失败请求
3. 获取trace_id：从日志中拿到`trace_id=abc-123`
4. 追踪全链路：`grep "trace_id=abc-123"`看到完整的执行流程
5. 定位问题：发现`llm_refine`节点耗时46秒

---

## 扩展路线图

### 第一阶段（当前）
✅ Trace ID生成和传递  
✅ 所有日志包含trace_id  
✅ 节点级链路追踪
✅ NLU日志记录 `fast_nlu_hit`、`nlu_provider`、`llm_calls`、intent 和命中数量
✅ 流式 Trace Callback 不再消费业务流，避免 TRACE=true 时影响 SSE 输出

### 第二阶段
📊 Token使用统计  
⏱️ 更多性能指标  
📈 Prometheus metrics暴露

### 第三阶段
🔗 OpenTelemetry集成  
📊 Jaeger分布式追踪  
💰 成本分析和告警

---

## 总结

这个方案的优势：
1. **简单**：核心就是一个trace_id在各层传递
2. **实用**：解决了最痛点的调试问题
3. **易扩展**：以后加metrics、tracing都很方便
4. **适合学习**：从简单开始，逐步深入

对于学习AI Agent的人来说，先把这个基础打牢，比一开始就上复杂的OpenTelemetry更有意义！
