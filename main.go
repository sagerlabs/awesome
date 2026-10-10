//go:build !desktop

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/sagerlabs/awesome/tft"
)

// 前端资源打进二进制：页面骨架 + 拆分出来的 CSS/JS（frontend/assets/）。
//
//go:embed frontend/index.html frontend/assets
var frontendFS embed.FS

func main() {
	logger := tft.NewLogger()

	logger.WithFields(logrus.Fields{
		"version":    tft.Version,
		"git_commit": tft.GitCommit,
		"build_time": tft.BuildTime,
	}).Info("TFT Copilot 构建信息")

	// env
	if err := checkEnv(logger); err != nil {
		logger.WithError(err).Fatal("环境变量检查失败")
	}

	// init tft handler
	ctx := context.Background()
	tftHandler, err := tft.NewHandler(ctx, logger)
	if err != nil {
		logger.WithError(err).Fatal("TFT Handler 初始化失败")
	}

	if os.Getenv("LOG_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()
	e.Use(
		gin.Logger(),
		gin.Recovery(),
		tft.RateLimitMiddleware(rateLimitRPS(), rateLimitBurst()),
	)
	// FIX-04：安全响应头（含 strict CSP）。CSP_POLICY=off 可关闭
	//（自建反代已统一注入安全头时避免重复）。
	if csp := cspPolicy(); csp != "" {
		e.Use(tft.SecurityHeadersMiddleware(csp))
	}
	tftHandler.RegisterRoutes(e)

	// 内嵌前端：/ 返回页面，/assets/* 返回拆分后的 CSS/JS。
	indexHTML, err := frontendFS.ReadFile("frontend/index.html")
	if err != nil {
		logger.WithError(err).Fatal("内嵌前端缺少 index.html")
	}
	assetsFS, err := fs.Sub(frontendFS, "frontend/assets")
	if err != nil {
		logger.WithError(err).Fatal("内嵌前端缺少 assets 目录")
	}
	e.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	e.StaticFS("/assets", http.FS(assetsFS))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: e,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		logger.WithField("signal", sig.String()).Info("收到退出信号，开始优雅退出")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Server 强制退出")
		}
	}()

	logger.WithFields(logrus.Fields{
		"addr": fmt.Sprintf("http://localhost:%s", port),
		"routes": []string{
			"GET  /                    (index.html)",
			"POST /v1/tft/nlu         (primary JSON)",
			"POST /v1/tft/nlu/stream  (primary SSE)",
			"POST /v1/tft/analyze     (legacy)",
			"POST /v1/tft/analyze/stream (legacy SSE)",
			"GET  /v1/tft/health",
		},
	}).Info("服务启动")

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.WithError(err).Fatal("Server 启动失败")
	}

	logger.Info("Server 已退出")
}

// cspPolicy 读取 CSP 覆盖值：
//   - 未设置 / 空  -> 使用 tft.DefaultCSP（strict，script-src 'self'）
//   - "off" / "none" -> 关闭安全头中间件（反代已统一注入时用）
//   - 其它值        -> 原样作为 CSP 策略
func cspPolicy() string {
	v := os.Getenv("CSP_POLICY")
	switch v {
	case "", "default":
		return tft.DefaultCSP
	case "off", "none":
		return ""
	}
	return v
}

// rateLimitRPS 读取每 IP 每秒请求上限，RATE_LIMIT_RPS<=0 或未设置时返回 0（禁用限流）。
func rateLimitRPS() float64 {
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// rateLimitBurst 读取突发额度，未设置时默认 20。
func rateLimitBurst() int {
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 20
}

// checkEnv
func checkEnv(logger *logrus.Logger) error {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "openai"
	}

	logger.WithField("provider", provider).Info("LLM Provider")

	switch provider {
	case "openai", "deepseek":
		if os.Getenv("OPENAI_API_KEY") == "" {
			return fmt.Errorf("LLM_PROVIDER=%s 时需要设置 OPENAI_API_KEY", provider)
		}
	case "ark":
		if os.Getenv("ARK_API_KEY") == "" {
			return fmt.Errorf("LLM_PROVIDER=ark 时需要设置 ARK_API_KEY")
		}
		if os.Getenv("ARK_MODEL_ID") == "" {
			return fmt.Errorf("LLM_PROVIDER=ark 时需要设置 ARK_MODEL_ID")
		}
	default:
		return fmt.Errorf("不支持的 LLM_PROVIDER: %s，支持 openai / deepseek / ark", provider)
	}

	return nil
}
