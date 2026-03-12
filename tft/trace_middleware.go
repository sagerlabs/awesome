package tft

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// ContextKeyTraceID 在 gin.Context 中存放 trace_id 的 key
	ContextKeyTraceID = "trace_id"
	// ContextKeyLogger 在 gin.Context 中存放带字段 logger 的 key
	ContextKeyLogger = "logger"
	// HeaderTraceID HTTP 头里的 trace id
	HeaderTraceID = "X-Request-Id"
)

// TraceMiddleware 为每个请求生成/透传 trace_id，并注入带 trace_id 的 logger
func TraceMiddleware(baseLogger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.Request.Header.Get(HeaderTraceID)
		if traceID == "" {
			traceID = uuid.NewString()
		}

		// 写回响应头，方便客户端和日志平台关联
		c.Writer.Header().Set(HeaderTraceID, traceID)

		// 带 trace_id 的 logger
		entry := baseLogger.WithField("trace_id", traceID)

		// 注入到 gin.Context，后续 handler 可以取出
		c.Set(ContextKeyTraceID, traceID)
		c.Set(ContextKeyLogger, entry)

		c.Next()
	}
}

