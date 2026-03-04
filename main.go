package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagerlabs/awesome/tft"
	"github.com/sirupsen/logrus"
)

func main() {
	// ── 1. 初始化 logrus ──────────────────────────────────────────────────────
	logger := tft.NewLogger()

	// ── 2. 检查环境变量 ───────────────────────────────────────────────────────
	if err := checkEnv(logger); err != nil {
		logger.WithError(err).Fatal("环境变量检查失败")
	}

	// ── 3. 初始化 TFT Handler ─────────────────────────────────────────────────
	ctx := context.Background()
	tftHandler, err := tft.NewHandler(ctx, logger)
	if err != nil {
		logger.WithError(err).Fatal("TFT Handler 初始化失败")
	}

	// ── 4. 配置 Gin ───────────────────────────────────────────────────────────
	if os.Getenv("LOG_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()

	e.Use(
		cors.Default(),
		gin.Recovery(), // panic 恢复
	)
	e.StaticFile("/", "./fronted/index.html")
	tftHandler.RegisterRoutes(e)

	// ── 5. 启动 HTTP Server ───────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      e,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── 6. 优雅退出 ───────────────────────────────────────────────────────────
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
			"POST /v1/tft/analyze",
			"POST /v1/tft/analyze/stream",
			"GET  /v1/tft/health",
		},
	}).Info("服务启动")

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.WithError(err).Fatal("Server 启动失败")
	}

	logger.Info("Server 已退出")
}

// checkEnv 检查必要的环境变量
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
