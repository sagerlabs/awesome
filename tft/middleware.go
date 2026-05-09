package tft

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultMaxRequestBytes int64 = 1 << 20 // 1 MiB

// RequestSizeLimit caps JSON request bodies before Gin attempts to bind them.
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = defaultMaxRequestBytes
	}
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// OptionalAPIKey protects API routes only when TFT_API_KEY is configured.
func OptionalAPIKey() gin.HandlerFunc {
	expected := strings.TrimSpace(os.Getenv("TFT_API_KEY"))
	return func(c *gin.Context) {
		if expected == "" || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if c.GetHeader("X-API-Key") != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid api key",
			})
			return
		}
		c.Next()
	}
}

// CORSMiddleware applies a small allowlist. Empty TFT_CORS_ORIGINS means local/dev only.
func CORSMiddleware() gin.HandlerFunc {
	allowlist := parseCORSAllowlist(os.Getenv("TFT_CORS_ORIGINS"))
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isOriginAllowed(origin, allowlist) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-Trace-ID, X-API-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func MaxRequestBytesFromEnv() int64 {
	raw := strings.TrimSpace(os.Getenv("TFT_MAX_REQUEST_BYTES"))
	if raw == "" {
		return defaultMaxRequestBytes
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultMaxRequestBytes
	}
	return n
}

func parseCORSAllowlist(raw string) map[string]struct{} {
	allowlist := map[string]struct{}{
		"http://localhost:8080": {},
		"http://127.0.0.1:8080": {},
	}
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowlist[origin] = struct{}{}
		}
	}
	return allowlist
}

func isOriginAllowed(origin string, allowlist map[string]struct{}) bool {
	if _, ok := allowlist["*"]; ok {
		return true
	}
	_, ok := allowlist[origin]
	return ok
}
