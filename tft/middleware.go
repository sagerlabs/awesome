package tft

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogMiddleware logs method, path, status, latency, client IP, and errors for each request.
func RequestLogMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		fields := logrus.Fields{
			"method":  method,
			"path":    path,
			"status":  status,
			"latency": duration,
			"ip":      c.ClientIP(),
		}
		if errs := c.Errors.Errors(); len(errs) > 0 {
			fields["errors"] = errs
		}

		entry := logger.WithFields(fields)
		switch {
		case status >= 500:
			entry.Error("请求处理失败")
		case status >= 400:
			entry.Warn("请求参数错误")
		default:
			entry.Info("请求完成")
		}
	}
}
