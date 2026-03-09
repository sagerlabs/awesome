package agent

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/components/model"

	// OpenAI / 兼容 OpenAI 协议的接入（如 DeepSeek、通义千问）
	"github.com/cloudwego/eino-ext/components/model/openai"

	// 字节跳动火山引擎 Ark（豆包大模型）
	"github.com/cloudwego/eino-ext/components/model/ark"
)

// ── Provider 类型 ─────────────────────────────────────────────────────────────

type ModelProvider string

const (
	ProviderOpenAI   ModelProvider = "openai"
	ProviderDeepSeek ModelProvider = "deepseek" // OpenAI 兼容协议
	ProviderArk      ModelProvider = "ark"      // 火山引擎豆包
)

// ── 统一配置 ──────────────────────────────────────────────────────────────────

type ModelConfig struct {
	Provider    ModelProvider
	Temperature float32
	MaxTokens   int
}

// DefaultModelConfig 默认配置，优先用环境变量决定 Provider
//
// 环境变量：
//
//	LLM_MAX_TOKENS = 200（默认）控制输出长度，越小 TTFT 越低
//	LLM_TEMPERATURE = 0.7（默认）
func DefaultModelConfig() *ModelConfig {
	provider := ModelProvider(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = ProviderOpenAI
	}

	maxTokens := 60 // 20字建议约30 token，60为硬上限，防止 LLM 超量输出
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTokens = n
		}
	}
	// 安全兜底：无论环境变量设多少，硬限制不超过 150
	// 防止误配置导致 token 飙升（Out >> In 的根本原因）
	if maxTokens > 150 {
		maxTokens = 3333
	}

	temperature := float32(0.7)
	if v := os.Getenv("LLM_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			temperature = float32(f)
		}
	}

	return &ModelConfig{
		Provider:    provider,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
}

// ── 工厂函数 ──────────────────────────────────────────────────────────────────

// NewChatModel 根据环境变量自动选择 LLM Provider，返回统一的 model.ChatModel 接口
//
// 环境变量配置：
//
//	LLM_PROVIDER = openai | deepseek | ark
//
//	# OpenAI
//	OPENAI_API_KEY   = sk-xxx
//	OPENAI_MODEL     = gpt-4o-mini
//	OPENAI_BASE_URL  = https://api.openai.com/v1  （可选，用于代理或兼容接口）
//
//	# DeepSeek（OpenAI 兼容协议）
//	OPENAI_API_KEY   = sk-xxx
//	OPENAI_MODEL     = deepseek-chat
//	OPENAI_BASE_URL  = https://api.deepseek.com
//
//	# 火山引擎 Ark（豆包）
//	ARK_API_KEY      = xxx
//	ARK_MODEL_ID     = doubao-pro-32k-xxxxxx  （推理接入点 ID）
func NewChatModel(ctx context.Context, cfg *ModelConfig) (model.ChatModel, error) {
	if cfg == nil {
		cfg = DefaultModelConfig()
	}

	switch cfg.Provider {

	case ProviderOpenAI, ProviderDeepSeek:
		return newOpenAIModel(ctx, cfg)

	case ProviderArk:
		return newArkModel(ctx, cfg)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// ── OpenAI / DeepSeek ─────────────────────────────────────────────────────────

func newOpenAIModel(ctx context.Context, cfg *ModelConfig) (model.ChatModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}

	modelName := os.Getenv("OPENAI_MODEL")
	if modelName == "" {
		if cfg.Provider == ProviderDeepSeek {
			// deepseek-chat 延迟较高，优先用 deepseek-reasoner 或更快的 V3
			// 可通过 OPENAI_MODEL=deepseek-chat 覆盖
			modelName = "deepseek-chat"
		} else {
			// gpt-4o-mini 比 gpt-4o 快 3x，对20字建议完全够用
			modelName = "gpt-4o-mini"
		}
	}

	ocfg := &openai.ChatModelConfig{
		APIKey:      apiKey,
		Model:       modelName,
		BaseURL:     os.Getenv("OPENAI_BASE_URL"), // 空字符串时使用默认值
		Temperature: &cfg.Temperature,
		//MaxTokens:   &cfg.MaxTokens,
		ExtraFields: map[string]any{
			"enable_thinking": false,
		},
	}

	chatModel, err := openai.NewChatModel(ctx, ocfg)
	if err != nil {
		return nil, fmt.Errorf("init openai model: %w", err)
	}
	return chatModel, nil
}

// ── 火山引擎 Ark（豆包）────────────────────────────────────────────────────────

func newArkModel(ctx context.Context, cfg *ModelConfig) (model.ChatModel, error) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY is not set")
	}

	modelID := os.Getenv("ARK_MODEL_ID")
	if modelID == "" {
		return nil, fmt.Errorf("ARK_MODEL_ID is not set（推理接入点 ID，非模型名称）")
	}

	// 超时从环境变量读，默认 60s；流式接口 TTFT 敏感，不要设太短
	timeoutSec := 60
	if v := os.Getenv("ARK_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}
	timeout := time.Duration(timeoutSec) * time.Second

	acfg := &ark.ChatModelConfig{
		APIKey:      apiKey,
		Model:       modelID,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
		Timeout:     &timeout,
	}

	chatModel, err := ark.NewChatModel(ctx, acfg)
	if err != nil {
		return nil, fmt.Errorf("init ark model: %w", err)
	}
	return chatModel, nil
}
