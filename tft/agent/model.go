package agent

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// ── Provider 类型 ─────────────────────────────────────────────────────────────

type ModelProvider string

const (
	ProviderOpenAI   ModelProvider = "openai"
	ProviderDeepSeek ModelProvider = "deepseek"
	ProviderArk      ModelProvider = "ark"
)

// ── 统一配置 ──────────────────────────────────────────────────────────────────

type ModelConfig struct {
	Provider    ModelProvider
	Temperature float32
	MaxTokens   int
}

// DefaultModelConfig 默认配置，优先用环境变量决定 Provider
func DefaultModelConfig() *ModelConfig {
	provider := ModelProvider(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = ProviderOpenAI // 默认 OpenAI
	}
	return &ModelConfig{
		Provider:    provider,
		Temperature: 0.7,
		MaxTokens:   50000,
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
		// DeepSeek 默认模型；OpenAI 默认模型
		if cfg.Provider == ProviderDeepSeek {
			modelName = "deepseek-chat"
		} else {
			modelName = "gpt-4o-mini"
		}
	}

	ocfg := &openai.ChatModelConfig{
		APIKey:      apiKey,
		Model:       modelName,
		BaseURL:     os.Getenv("OPENAI_BASE_URL"), // 空字符串时使用默认值
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
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

	timeout := 300 * time.Second
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
