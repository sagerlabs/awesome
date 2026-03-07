package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/sagerlabs/awesome/tft/agent"
	"github.com/sagerlabs/awesome/tft/data"
	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// index.html embed 进二进制（路径相对于 desktop/ 目录，需上一级）
//
//go:embed ../frontend/index.html
var assets embed.FS

type App struct {
	ctx    context.Context
	agent  *agent.Agent
	logger *logrus.Logger
}

// NewApp Wails 要求的构造函数
func NewApp(logger *logrus.Logger) *App {
	return &App{logger: logger}
}

// OnStartup Wails 启动时调用，初始化 Store + Agent
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// 加载数据
	a.logger.WithField("data_dir", data.GetDataDir()).Info("加载 TFT 数据")
	store, err := data.NewStore(data.GetDataDir())
	if err != nil {
		a.logger.WithError(err).Fatal("数据加载失败")
	}

	// 初始化 Agent
	a.agent, err = agent.NewAgentWithConfig(ctx, store, nil)
	if err != nil {
		a.logger.WithError(err).Fatal("Agent 初始化失败")
	}
	a.logger.Info("TFT Copilot 启动完成")

	// 全屏覆盖，置顶
	screens, err := wailsRuntime.ScreenGetAll(ctx)
	if err == nil && len(screens) > 0 {
		w := screens[0].Size.Width
		h := screens[0].Size.Height
		wailsRuntime.WindowSetSize(ctx, w, h)
		wailsRuntime.WindowSetPosition(ctx, 0, 0)
	}
	wailsRuntime.WindowSetAlwaysOnTop(ctx, true)
}

// Analyze 暴露给前端 JS 调用：window.go.main.App.Analyze(input)
func (a *App) Analyze(input string) (string, error) {
	out, err := a.agent.Analyze(a.ctx, input)
	if err != nil {
		return "", fmt.Errorf("推理失败: %w", err)
	}
	return out.LLMAdvice, nil
}

// ── 主入口 ────────────────────────────────────────────────────────
func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	app := NewApp(logger)

	err := wails.Run(&options.App{
		Title:  "TFT Copilot",
		Width:  1920,
		Height: 1080,

		Frameless:        true,
		AlwaysOnTop:      true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		OnStartup: app.OnStartup,
		Bind:      []interface{}{app},

		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			TitleBar:             mac.TitleBarHiddenInset(),
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableFramelessWindowDecorations: true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		logger.WithError(err).Fatal("Wails 启动失败")
	}
}
