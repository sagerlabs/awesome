package tft

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/session"
)

// stubAgent is a hand-rolled agentService stub. Each field overrides one
// method; nil fields fall back to a benign default so tests only specify
// what they care about.
type stubAgent struct {
	analyzeFn    func(ctx context.Context, in string) (*agent.GraphOutput, error)
	nluAnalyzeFn func(ctx context.Context, in string) (*agent.NluEnrichedContext, error)

	// lastSessionID records the session seen by the most recent call, so tests
	// can assert the handler threaded it through context.
	lastSessionID string
}

func (s *stubAgent) Analyze(ctx context.Context, in string) (*agent.GraphOutput, error) {
	s.lastSessionID = session.IDFromContext(ctx)
	if s.analyzeFn != nil {
		return s.analyzeFn(ctx, in)
	}
	return &agent.GraphOutput{LLMAdvice: "ok"}, nil
}

func (s *stubAgent) AnalyzeStream(ctx context.Context, in string) (*schema.StreamReader[*agent.GraphOutput], error) {
	s.lastSessionID = session.IDFromContext(ctx)
	return schema.StreamReaderFromArray([]*agent.GraphOutput{{LLMAdvice: "chunk"}}), nil
}

func (s *stubAgent) NluAnalyze(ctx context.Context, in string) (*agent.NluEnrichedContext, error) {
	s.lastSessionID = session.IDFromContext(ctx)
	if s.nluAnalyzeFn != nil {
		return s.nluAnalyzeFn(ctx, in)
	}
	return &agent.NluEnrichedContext{UserInput: in}, nil
}

func (s *stubAgent) NluAnalyzeStream(ctx context.Context, in string) (*schema.StreamReader[*agent.GraphOutput], error) {
	s.lastSessionID = session.IDFromContext(ctx)
	return schema.StreamReaderFromArray([]*agent.GraphOutput{{LLMAdvice: "chunk"}}), nil
}

func (s *stubAgent) RecordAdvice(ctx context.Context, userInput, advice, intent string) {}

// newTestRouter wires a Handler with the stub agent into a fresh gin engine.
func newTestRouter(ag agentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	h := &Handler{ag: ag, logger: logger}
	e := gin.New()
	h.RegisterRoutes(e)
	return e
}

func postJSON(e *gin.Engine, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w
}

// ── /v1/tft/nlu ───────────────────────────────────────────────────────────────

func TestNluAnalyzeRejectsBadRequests(t *testing.T) {
	e := newTestRouter(&stubAgent{})

	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{not-json`},
		{"empty input", `{"input":""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postJSON(e, "/v1/tft/nlu", tc.body, nil)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestNluAnalyzeSuccess(t *testing.T) {
	e := newTestRouter(&stubAgent{})

	w := postJSON(e, "/v1/tft/nlu", `{"input":"剑魔打工强吗"}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("response should report success, got: %s", w.Body.String())
	}
}

func TestNluAnalyzeThreadsSessionID(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		headers map[string]string
		want    string
	}{
		{"from body", `{"input":"q","session_id":"sess-body"}`, nil, "sess-body"},
		{"from header", `{"input":"q"}`, map[string]string{"X-Session-ID": "sess-header"}, "sess-header"},
		{"body wins over header", `{"input":"q","session_id":"sess-body"}`,
			map[string]string{"X-Session-ID": "sess-header"}, "sess-body"},
		{"absent means stateless", `{"input":"q"}`, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubAgent{}
			e := newTestRouter(stub)
			w := postJSON(e, "/v1/tft/nlu", tc.body, tc.headers)
			if w.Code != http.StatusOK {
				t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
			}
			if stub.lastSessionID != tc.want {
				t.Fatalf("agent saw session %q, want %q", stub.lastSessionID, tc.want)
			}
		})
	}
}

// ── legacy /v1/tft/analyze ───────────────────────────────────────────────────

func TestLegacyAnalyzeSetsDeprecationHeaders(t *testing.T) {
	e := newTestRouter(&stubAgent{})

	w := postJSON(e, "/v1/tft/analyze", `{"input":"q"}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Deprecation") != "true" {
		t.Error("legacy route should set Deprecation header")
	}
	if w.Header().Get("Sunset") == "" {
		t.Error("legacy route should set Sunset header")
	}
	if !strings.Contains(w.Header().Get("Link"), "successor-version") {
		t.Errorf("legacy route should link the successor, got %q", w.Header().Get("Link"))
	}
}

func TestLegacyAnalyzeDisabledReturns410(t *testing.T) {
	stub := &stubAgent{
		analyzeFn: func(context.Context, string) (*agent.GraphOutput, error) {
			return nil, agent.ErrLegacyGraphDisabled
		},
	}
	e := newTestRouter(stub)

	w := postJSON(e, "/v1/tft/analyze", `{"input":"q"}`, nil)
	if w.Code != http.StatusGone {
		t.Fatalf("disabled legacy graph should map to 410, got %d: %s", w.Code, w.Body.String())
	}
}

// ── /v1/tft/health ───────────────────────────────────────────────────────────

func TestHealthDegradedWithoutData(t *testing.T) {
	// store is nil → comp_count 0 → 503 degraded.
	e := newTestRouter(&stubAgent{})

	req := httptest.NewRequest(http.MethodGet, "/v1/tft/health", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("health without data should be 503, got %d", w.Code)
	}
	body := w.Body.String()
	for _, field := range []string{`"status":"degraded"`, `"version"`, `"git_commit"`} {
		if !strings.Contains(body, field) {
			t.Errorf("health body missing %s: %s", field, body)
		}
	}
}

// ── 限流中间件经 gin 全链路 ───────────────────────────────────────────────────

func TestRateLimitMiddlewareReturns429ThroughRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(RateLimitMiddleware(0.001 /* 几乎不回填 */, 1 /* burst */))
	e.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "9.9.9.9:1234"
		w := httptest.NewRecorder()
		e.ServeHTTP(w, req)
		return w.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("second request should be limited to 429, got %d", code)
	}
}
