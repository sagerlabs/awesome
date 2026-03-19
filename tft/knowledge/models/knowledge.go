package models

// KnowledgeDoc 知识文档模型
type KnowledgeDoc struct {
	ID          string   `json:"id" yaml:"id"`
	Title       string   `json:"title" yaml:"title"`
	Category    string   `json:"category" yaml:"category"` // "rules" | "strategy" | "faq" | "beginner"
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Content     string   `json:"content" yaml:"content"` // Markdown内容
	Summary     string   `json:"summary,omitempty" yaml:"summary,omitempty"`
	Version     string   `json:"version" yaml:"version"`
	Author      string   `json:"author,omitempty" yaml:"author,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

// SearchQuery 搜索查询
type SearchQuery struct {
	Query     string   `json:"query"`
	Category  string   `json:"category,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	TopK      int      `json:"top_k,omitempty"`
	MinScore  float64  `json:"min_score,omitempty"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Doc      KnowledgeDoc `json:"doc"`
	Score    float64      `json:"score"`
	Source   string       `json:"source"` // "keyword" | "semantic" | "hybrid"
	Highlights []string   `json:"highlights,omitempty"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Query     string        `json:"query"`
	History   []ChatMessage `json:"history,omitempty"`
	SearchCtx []SearchResult `json:"search_ctx,omitempty"`
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Answer      string         `json:"answer"`
	References  []KnowledgeDoc `json:"references,omitempty"`
	Sources     []string       `json:"sources,omitempty"`
}
