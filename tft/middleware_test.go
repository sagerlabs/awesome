package tft

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// FIX-04：安全响应头中间件契约——所有响应必须带 strict CSP 与基础安全头。
func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		csp        string
		wantCSP    string
		wantHeader bool
	}{
		{
			name:       "默认策略",
			csp:        DefaultCSP,
			wantCSP:    DefaultCSP,
			wantHeader: true,
		},
		{
			name:       "自定义策略原样透传",
			csp:        "script-src 'self' https://example.test",
			wantCSP:    "script-src 'self' https://example.test",
			wantHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := gin.New()
			e.Use(SecurityHeadersMiddleware(tt.csp))
			e.GET("/health", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			e.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			got := w.Header().Get("Content-Security-Policy")
			if !tt.wantHeader {
				if got != "" {
					t.Fatalf("CSP 应为空, got %q", got)
				}
				return
			}
			if got != tt.wantCSP {
				t.Fatalf("CSP = %q, want %q", got, tt.wantCSP)
			}

			// 基础安全头
			for header, want := range map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":       "DENY",
				"Referrer-Policy":       "no-referrer",
			} {
				if got := w.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}

// 默认 CSP 必须包含 FIX-04 的关键指令：script-src 'self'（禁内联/禁 CDN 脚本）。
func TestDefaultCSPContainsStrictScriptSrc(t *testing.T) {
	if !containsDirective(DefaultCSP, "script-src", "'self'") {
		t.Errorf("DefaultCSP 缺少 script-src 'self'：%s", DefaultCSP)
	}
	for _, banned := range []string{"unsafe-inline", "unsafe-eval", "*"} {
		if containsDirective(DefaultCSP, "script-src", banned) {
			t.Errorf("DefaultCSP 的 script-src 不应包含 %q", banned)
		}
	}
	// 后端地址可配置 -> connect-src 必须放行 http:/https:
	if !containsDirective(DefaultCSP, "connect-src", "http:") {
		t.Errorf("DefaultCSP 的 connect-src 缺少 http:（可配置后端地址会失效）")
	}
}

// containsDirective 检查策略里某个指令是否包含指定 token。
func containsDirective(policy, directive, token string) bool {
	for _, part := range splitDirectives(policy) {
		if len(part) > 0 && part[0] == directive {
			for _, tk := range part[1:] {
				if tk == token {
					return true
				}
			}
		}
	}
	return false
}

func splitDirectives(policy string) [][]string {
	var out [][]string
	current := []string{}
	for _, f := range splitFields(policy) {
		if f == ";" {
			if len(current) > 0 {
				out = append(out, current)
			}
			current = []string{}
		} else {
			current = append(current, f)
		}
	}
	if len(current) > 0 {
		out = append(out, current)
	}
	return out
}

func splitFields(s string) []string {
	var fields []string
	cur := ""
	for _, r := range s {
		switch r {
		case ' ', '\t', ';':
			if cur != "" {
				fields = append(fields, cur)
				cur = ""
			}
			if r == ';' {
				fields = append(fields, ";")
			}
		default:
			cur += string(r)
		}
	}
	if cur != "" {
		fields = append(fields, cur)
	}
	return fields
}
