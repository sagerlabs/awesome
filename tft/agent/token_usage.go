package agent

import (
	"context"
	"sync"
	"time"
)

// TokenUsage 单次请求的token使用统计
type TokenUsage struct {
	RequestID    string        `json:"request_id"`    // 请求ID
	Model        string        `json:"model"`         // 使用的模型
	InputTokens  int           `json:"input_tokens"`  // 输入token数
	OutputTokens int           `json:"output_tokens"` // 输出token数
	TotalTokens  int           `json:"total_tokens"`  // 总token数
	CostUSD      float64       `json:"cost_usd"`      // 预估成本（美元）
	Duration     time.Duration `json:"duration"`      // 请求耗时
	NodeUsages   []NodeUsage   `json:"node_usages"`   // 各节点token使用情况
}

// NodeUsage 单个节点的token使用统计
type NodeUsage struct {
	NodeName     string `json:"node_name"`     // 节点名称
	InputTokens  int    `json:"input_tokens"`  // 节点输入token数
	OutputTokens int    `json:"output_tokens"` // 节点输出token数
}

// TokenUsageStore 并发安全的token使用存储
type TokenUsageStore struct {
	mu     sync.Mutex
	usages map[string]*TokenUsage // key: request_id
}

// NewTokenUsageStore 创建token使用存储
func NewTokenUsageStore() *TokenUsageStore {
	return &TokenUsageStore{
		usages: make(map[string]*TokenUsage),
	}
}

// StartRequest 开始一个新请求的token统计
func (s *TokenUsageStore) StartRequest(requestID string, model string) *TokenUsage {
	s.mu.Lock()
	defer s.mu.Unlock()

	usage := &TokenUsage{
		RequestID:  requestID,
		Model:      model,
		NodeUsages: make([]NodeUsage, 0),
	}
	s.usages[requestID] = usage
	return usage
}

// GetUsage 获取指定请求的token使用统计
func (s *TokenUsageStore) GetUsage(requestID string) (*TokenUsage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	usage, ok := s.usages[requestID]
	return usage, ok
}

// RecordLLMUsage 记录LLM调用的token使用
func (s *TokenUsageStore) RecordLLMUsage(requestID string, inputTokens, outputTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if usage, ok := s.usages[requestID]; ok {
		usage.InputTokens += inputTokens
		usage.OutputTokens += outputTokens
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
		usage.CostUSD = calculateCost(usage.Model, usage.InputTokens, usage.OutputTokens)
	}
}

// RecordNodeUsage 记录节点的token使用
func (s *TokenUsageStore) RecordNodeUsage(requestID, nodeName string, inputTokens, outputTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if usage, ok := s.usages[requestID]; ok {
		usage.NodeUsages = append(usage.NodeUsages, NodeUsage{
			NodeName:     nodeName,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		})
	}
}

// FinishRequest 完成请求统计，清理内存
func (s *TokenUsageStore) FinishRequest(requestID string, duration time.Duration) *TokenUsage {
	s.mu.Lock()
	defer s.mu.Unlock()

	if usage, ok := s.usages[requestID]; ok {
		usage.Duration = duration
		delete(s.usages, requestID)
		return usage
	}
	return nil
}

// calculateCost 根据模型和token数计算预估成本
func calculateCost(model string, inputTokens, outputTokens int) float64 {
	// 简化版本，实际使用时可以根据具体模型定价调整
	switch {
	case contains(model, "gpt-4o-mini"):
		return float64(inputTokens)*0.00000015 + float64(outputTokens)*0.0000006
	case contains(model, "deepseek"):
		return float64(inputTokens)*0.000000001 + float64(outputTokens)*0.000000002
	case contains(model, "doubao"):
		// Doubao价格是人民币，粗略转换为美元（1 USD ≈ 7 CNY）
		return (float64(inputTokens)*0.000008 + float64(outputTokens)*0.00002) / 7
	default:
		// 默认使用一个通用的价格
		return float64(inputTokens+outputTokens) * 0.000001
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOf(s, substr) != -1)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// contextKey 用于在context中存储token usage
type tokenUsageKey struct{}

// WithTokenUsage 将token usage存入context
func WithTokenUsage(ctx context.Context, usage *TokenUsage) context.Context {
	return context.WithValue(ctx, tokenUsageKey{}, usage)
}

// TokenUsageFromContext 从context中获取token usage
func TokenUsageFromContext(ctx context.Context) (*TokenUsage, bool) {
	usage, ok := ctx.Value(tokenUsageKey{}).(*TokenUsage)
	return usage, ok
}
