package tft

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/sse"
	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/data"
)

// Handler TFT Copilot 的 HTTP 处理器
type Handler struct {
	ag     *agent.Agent
	logger *logrus.Logger
}

// NewHandler 初始化 Handler，加载数据 + 编译 Graph
func NewHandler(ctx context.Context, logger *logrus.Logger) (*Handler, error) {
	logger.Info("开始初始化 TFT Handler")

	// 1. 加载数据
	logger.WithField("data_dir", data.GetDataDir()).Info("加载 TFT 数据文件")
	store, err := data.NewStore(data.GetDataDir())
	if err != nil {
		logger.WithError(err).Error("数据加载失败")
		return nil, fmt.Errorf("load tft data: %w", err)
	}
	logger.WithField("comp_count", len(store.AllComps())).Info("数据加载完成")

	// 2. 初始化 Agent
	logger.Info("初始化 Eino Agent（编译 Graph + 加载 LLM）")
	a, err := agent.NewAgent(ctx, store)
	if err != nil {
		logger.WithError(err).Error("Agent 初始化失败")
		return nil, fmt.Errorf("init tft agent: %w", err)
	}
	logger.Info("TFT Handler 初始化完成")

	return &Handler{ag: a, logger: logger}, nil
}

// ── 请求/响应结构体 ────────────────────────────────────────────────────────────

type AnalyzeRequest struct {
	Input string `json:"input"`
	Plain bool   `json:"plain"` // true = data 字段直接输出纯文本，方便 curl 测试
}

type AnalyzeResponse struct {
	Success bool               `json:"success"`
	Data    *agent.GraphOutput `json:"data,omitempty"`
	Error   string             `json:"error,omitempty"`
}

type StreamChunk struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ── 路由注册 ───────────────────────────────────────────────────────────────────

func (h *Handler) RegisterRoutes(e *gin.Engine) {
	group := e.Group("/v1")
	group.POST("/tft/analyze", h.Analyze)
	group.POST("/tft/analyze/stream", h.AnalyzeStream) // ← 注意：路由是 /stream 不是 /analyzeStream
	group.GET("/tft/health", h.Health)
}

// ── POST /v1/tft/analyze ──────────────────────────────────────────────────────

func (h *Handler) Analyze(c *gin.Context) {
	log := h.logger

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Warn("请求体解析失败")
		c.JSON(http.StatusBadRequest, AnalyzeResponse{Success: false, Error: "invalid request body"})
		return
	}
	if req.Input == "" {
		log.Warn("input 字段为空")
		c.JSON(http.StatusBadRequest, AnalyzeResponse{Success: false, Error: "input is required"})
		return
	}

	log.WithField("input", req.Input).Info("开始分析")
	start := time.Now()

	// 用独立 context，不受客户端断开影响
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output, err := h.ag.Analyze(ctx, req.Input)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"input":   req.Input,
			"elapsed": time.Since(start).String(),
		}).Error("Agent 推理失败")
		c.JSON(http.StatusInternalServerError, AnalyzeResponse{Success: false, Error: err.Error()})
		return
	}

	log.WithFields(logrus.Fields{
		"input":     req.Input,
		"elapsed":   time.Since(start).String(),
		"rec_count": len(output.Recommendations),
	}).Info("分析完成")

	c.JSON(http.StatusOK, AnalyzeResponse{Success: true, Data: output})
}

// ── POST /v1/tft/analyze/stream ───────────────────────────────────────────────

func (h *Handler) AnalyzeStream(c *gin.Context) {
	log := h.logger

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Warn("流式请求体解析失败")
		c.JSON(http.StatusBadRequest, AnalyzeResponse{Success: false, Error: "invalid request body"})
		return
	}
	if req.Input == "" {
		log.Warn("流式接口 input 字段为空")
		c.JSON(http.StatusBadRequest, AnalyzeResponse{Success: false, Error: "input is required"})
		return
	}

	// ── SSE panic 修复 ────────────────────────────────────────────────────────
	// gin 的 c.Writer 实现了 http.Flusher，但 sse.Server 内部直接断言
	// 提前检查，避免 panic，给客户端一个明确的错误
	if _, ok := c.Writer.(http.Flusher); !ok {
		log.Error("当前 ResponseWriter 不支持流式推送（不实现 http.Flusher）")
		c.JSON(http.StatusInternalServerError, AnalyzeResponse{
			Success: false, Error: "streaming not supported",
		})
		return
	}

	log.WithField("input", req.Input).Info("开始流式分析")

	var srv *sse.Server
	srv = sse.NewServer(
		sse.WithBufferSize(100),
		sse.WithOnConnect(func(_ chan *sse.Event) {
			// 流式接口保留 c.Request.Context()：客户端断开时级联取消推理
			go h.runStream(c.Request.Context(), req.Input, req.Plain, srv)
		}),
	)

	srv.ServeHTTP(c.Writer, c.Request)
}

// ── GET /v1/tft/health ────────────────────────────────────────────────────────

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ── 流式推理 goroutine ────────────────────────────────────────────────────────

// flushThreshold 触发推送的条件：
// 缓冲累积超过此字数，或收到标点/换行时立即推送
const flushThreshold = 5

func (h *Handler) runStream(ctx context.Context, input string, plain bool, srv *sse.Server) {
	log := h.logger.WithField("input", input)
	start := time.Now()
	tokenCount := 0

	sr, err := h.ag.AnalyzeStream(ctx, input)
	if err != nil {
		log.WithError(err).Error("流式推理启动失败")
		srv.Publish(buildEvent("error", StreamChunk{Type: "error", Error: err.Error()}, plain))
		return
	}
	defer sr.Close()

	var buf strings.Builder // token 缓冲区

	// flush 把缓冲区内容推送出去，然后清空
	flush := func() {
		s := buf.String()
		if s == "" {
			return
		}
		tokenCount++
		srv.Publish(buildEvent("message", StreamChunk{Type: "token", Content: s}, plain))
		buf.Reset()
	}

	for {
		output, err := sr.Recv()
		if err != nil {
			// 流结束前把缓冲区剩余内容推出去
			flush()
			if err == io.EOF {
				log.WithFields(logrus.Fields{
					"token_count": tokenCount,
					"elapsed":     time.Since(start).String(),
				}).Info("流式推理完成")
				srv.Publish(buildEvent("done", StreamChunk{Type: "done"}, plain))
			} else {
				log.WithError(err).WithFields(logrus.Fields{
					"token_count": tokenCount,
					"elapsed":     time.Since(start).String(),
				}).Error("流式推理中断")
				srv.Publish(buildEvent("error", StreamChunk{Type: "error", Error: err.Error()}, plain))
			}
			return
		}
		if output == nil || output.LLMAdvice == "" {
			continue
		}

		buf.WriteString(output.LLMAdvice)

		// 遇到标点或换行立即推送（自然断句，阅读体验好）
		// 或缓冲超过阈值也推送（防止长句一直不断句）
		last := output.LLMAdvice[len(output.LLMAdvice)-1]
		shouldFlush := buf.Len() >= flushThreshold ||
			last == ':' || last == ' ' ||
			last == '.' || last == ',' || last == '!' || last == '?'

		if shouldFlush {
			flush()
		}
	}
}

// ── SSE 事件构造 ──────────────────────────────────────────────────────────────

// buildEvent 构造 SSE 事件
// plain=false（默认）: data 字段是 JSON，前端用 JSON.parse(e.data).content 解析
// plain=true         : data 字段直接是纯文本，方便 curl / 终端测试
func buildEvent(eventType string, chunk StreamChunk, plain bool) *sse.Event {
	e := &sse.Event{Event: eventType}
	if plain {
		// 纯文本模式：done/error 事件输出对应提示，message 直接输出内容
		switch chunk.Type {
		case "token":
			e.Data = chunk.Content
		case "done":
			e.Data = ""
		case "error":
			e.Data = "[错误] " + chunk.Error
		}
	} else {
		_ = e.MarshalData(chunk)
	}
	return e
}
