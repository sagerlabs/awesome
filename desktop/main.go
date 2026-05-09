//go:build desktop
// +build desktop

package main

import (
	"context"
	"fmt"
	"os"

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

// App 暴露给前端的 Go 方法
type App struct {
	ctx    context.Context
	agent  *agent.Agent
	logger *logrus.Logger
}

func NewApp(logger *logrus.Logger) *App {
	return &App{logger: logger}
}

// OnStartup Wails 启动时调用，初始化 Store + Agent
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	a.logger.WithField("data_dir", data.GetDataDir()).Info("加载 TFT 数据")
	store, err := data.NewStore(data.GetDataDir())
	if err != nil {
		a.logger.WithError(err).Fatal("数据加载失败")
	}

	a.agent, err = agent.NewAgentWithConfig(ctx, store, &agent.AgentConfig{
		Logger:      a.logger,
		EnableTrace: os.Getenv("TRACE") == "true",
	})
	if err != nil {
		a.logger.WithError(err).Fatal("Agent 初始化失败")
	}
	a.logger.Info("TFT Copilot 启动完成")

	// 桌面端使用真实小浮窗：右下角置顶，不再创建全屏透明覆盖层。
	screens, err := wailsRuntime.ScreenGetAll(ctx)
	if err == nil && len(screens) > 0 {
		const width = 430
		const height = 660
		margin := 28
		screen := screens[0].Size
		wailsRuntime.WindowSetSize(ctx, width, height)
		wailsRuntime.WindowSetPosition(ctx, screen.Width-width-margin, screen.Height-height-margin)
	}
	wailsRuntime.WindowSetAlwaysOnTop(ctx, true)
}

// Analyze 暴露给前端 JS：window.go.main.App.Analyze(input)
func (a *App) Analyze(input string) (string, error) {
	out, err := a.agent.NluAdvice(a.ctx, input)
	if err != nil {
		return "", fmt.Errorf("推理失败: %w", err)
	}
	return out, nil
}

// Minimize 让无边框浮窗可从前端最小化。
func (a *App) Minimize() {
	if a.ctx != nil {
		wailsRuntime.WindowMinimise(a.ctx)
	}
}

func frontendAssetsDir() string {
	for _, dir := range []string{"frontend", "../frontend"} {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return "frontend"
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	app := NewApp(logger)

	err := wails.Run(&options.App{
		Title:            "TFT Copilot",
		Width:            430,
		Height:           660,
		Frameless:        true,
		AlwaysOnTop:      true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},

		AssetServer: &assetserver.Options{
			Assets: os.DirFS(frontendAssetsDir()),
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
