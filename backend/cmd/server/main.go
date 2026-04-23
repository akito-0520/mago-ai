package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/akito-0520/mago-ai/backend/internal/config"
	"github.com/akito-0520/mago-ai/backend/internal/interface/http/handler"
)

func main() {
	// slog の設定
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 環境変数の読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load faild", "err", err)
		os.Exit(1)
	}

	// Echoインスタンスの作成
	e := echo.New()

	// middleware の登録
	// リクエストをログで出力
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:  true,
		LogURI:     true,
		LogMethod:  true,
		LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			slog.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency_ms", v.Latency.Milliseconds(),
			)
			return nil
		},
	}))
	e.Use(middleware.Recover()) // panicが起きた時に500エラーに変換

	// ルートの登録
	e.GET("/healthz", handler.Health)
	e.POST("/webhook", handler.Webhook(cfg.LineChannelSecret))

	// サーバー起動
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("sever start failed", "err", err)
			os.Exit(1)
		}
	}()

	// ctrlまたはkillをリッスンする
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// contextを作成してGraceful Shutdown を行う
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	slog.Info("shutdown started")
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
}
