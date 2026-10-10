package tft

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DefaultCSP 是 Web 端（非 desktop build）的默认 Content-Security-Policy。
//
// FIX-04 约束：
//   - script-src 'self'：页面已移除全部内联脚本与 on*= 事件属性，marked / sanitizer
//     均为本地 vendor 资源（随二进制 embed），不再依赖任何 CDN；
//   - style-src 放行 fonts.googleapis.com，font-src 放行 fonts.gstatic.com：
//     index.html 仍引用 Google Fonts（离线时 CSS 有系统字体兜底）；
//   - connect-src 必须放行 http:/https:：后端地址可在设置弹窗里改成任意主机端口，
//     写死 'self' 会让"改服务器地址"这个既有能力直接失效。
const DefaultCSP = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com data:; " +
	"img-src 'self' data:; " +
	"connect-src 'self' http: https: ws: wss:; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'"

// SecurityHeadersMiddleware 为所有响应补充基础安全头 + CSP。
//
// 注意 X-Content-Type-Options: nosniff 对 SSE 是必要的：SSE 走
// text/event-stream，若响应没有 Content-Type，浏览器会做 MIME 嗅探并可能把
// 流内容当 HTML 渲染，nosniff 可阻断这条路径。
func SecurityHeadersMiddleware(csp string) gin.HandlerFunc {
	if csp == "" {
		csp = DefaultCSP
	}
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

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
