package tft

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapMiddleware 请求日志中间件
// 记录：method、path、status、耗时、客户端 IP、错误信息
func ZapMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 把 logger 注入 gin.Context，handler 里通过 ctxLogger(c) 取出
		c.Set("logger", logger)

		c.Next()

		// 请求结束后记录日志
		duration := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", duration),
			zap.String("ip", c.ClientIP()),
		}

		// 有错误时附加错误信息
		if errs := c.Errors.Errors(); len(errs) > 0 {
			fields = append(fields, zap.Strings("errors", errs))
		}

		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			logger.Error("请求处理失败", fields...)
		case status >= 400:
			logger.Warn("请求参数错误", fields...)
		default:
			logger.Info("请求完成", fields...)
		}
	}
}

// ctxLogger 从 gin.Context 中取出 logger
// 如果没注入（比如单测），降级为 zap.NewNop()
func ctxLogger(c *gin.Context) *zap.Logger {
	if v, exists := c.Get("logger"); exists {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return zap.NewNop()
}
