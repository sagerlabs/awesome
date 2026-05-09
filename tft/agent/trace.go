package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/tft/trace"
)

// NewTraceCallback 构建节点级链路追踪 Handler
// 使用 callbacks.NewHandlerBuilder() 注册各阶段回调，无需手动实现接口
func NewTraceCallback(logger *logrus.Logger) callbacks.Handler {
	spans := &spanStore{}

	return callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			spans.start(info)
			traceID, _ := trace.TraceIDFromContext(ctx)
			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"node":      info.Name,
				"component": info.Type,
			}).Debug("→ 节点开始")
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			elapsed, ok := spans.end(info)
			traceID, _ := trace.TraceIDFromContext(ctx)
			fields := logrus.Fields{
				"trace_id":  traceID,
				"node":      info.Name,
				"component": info.Type,
			}
			if ok {
				fields["elapsed"] = elapsed.Round(time.Millisecond).String()
			}
			// LLM 节点额外打印输出摘要
			if info.Name == nodeLLM {
				if msg, ok := output.(*schema.Message); ok && msg != nil {
					fields["preview"] = truncate(msg.Content, 60)
				}
			}
			logger.WithFields(fields).Info("✓ 节点完成")
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			elapsed, ok := spans.end(info)
			traceID, _ := trace.TraceIDFromContext(ctx)
			fields := logrus.Fields{
				"trace_id":  traceID,
				"node":      info.Name,
				"component": info.Type,
				"error":     err.Error(),
			}
			if ok {
				fields["elapsed"] = elapsed.Round(time.Millisecond).String()
			}
			logger.WithFields(fields).Error("✗ 节点失败")
			return ctx
		}).
		OnStartWithStreamInputFn(func(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
			spans.start(info)
			traceID, _ := trace.TraceIDFromContext(ctx)
			input.Close() // 不消费 stream，直接关闭避免泄露
			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"node":      info.Name,
				"component": info.Type,
			}).Debug("→ 节点开始（流式输入）")
			return ctx
		}).
		OnEndWithStreamOutputFn(func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			startTime, _ := spans.endTime(info)
			traceID, _ := trace.TraceIDFromContext(ctx)
			_ = output
			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"node":      info.Name,
				"component": info.Type,
				"elapsed":   time.Since(startTime).Round(time.Millisecond).String(),
			}).Info("✓ 节点完成（流式输出，不消费业务流）")
			return ctx
		}).
		Build()
}

// ── spanStore 并发安全的耗时记录 ──────────────────────────────────────────────

type spanStore struct {
	mu    sync.Mutex
	spans map[string]time.Time
}

func (s *spanStore) start(info *callbacks.RunInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spans == nil {
		s.spans = make(map[string]time.Time)
	}
	s.spans[spanKey(info)] = time.Now()
}

// end 返回耗时并删除记录
func (s *spanStore) end(info *callbacks.RunInfo) (time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.spans[spanKey(info)]
	if ok {
		delete(s.spans, spanKey(info))
		return time.Since(t), true
	}
	return 0, false
}

// endTime 返回开始时间（不删除，用于异步 goroutine 计时）
func (s *spanStore) endTime(info *callbacks.RunInfo) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.spans[spanKey(info)]
	if ok {
		delete(s.spans, spanKey(info))
	}
	return t, ok
}

func spanKey(info *callbacks.RunInfo) string {
	return fmt.Sprintf("%s/%s", info.Name, info.Component)
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
