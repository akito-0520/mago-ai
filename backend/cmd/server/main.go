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
	"github.com/akito-0520/mago-ai/backend/internal/infrastructure/linebot"
	"github.com/akito-0520/mago-ai/backend/internal/infrastructure/postgres"
	"github.com/akito-0520/mago-ai/backend/internal/interface/http/handler"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"
)

func main() {
	// slog の設定
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 環境変数の読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
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

	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer func() { // graceful shutdown 時に DB 接続も閉じる
		err := db.Close()
		if err != nil {
			slog.Error("postgres close failed", "err", err)
		}
	}()

	// lineClient の生成
	lineClient, err := linebot.New(cfg.LineChannelAccessToken)
	if err != nil {
		slog.Error("linebot setup failed", "err", err)
		os.Exit(1)
	}

	// Repository 生成
	lineUsers := postgres.NewLineUserRepository(db)
	registerTokens := postgres.NewRegisterTokenRepository(db)

	// Usecase 生成
	registerUC := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens)
	respondUC := usecase.NewRespondToIncomingMessage(lineUsers, lineClient, registerUC)

	// webhook ハンドラーのセットアップ
	webhookHandler := handler.Webhook(cfg.LineChannelSecret, respondUC)

	// ルートの登録
	e.GET("/healthz", handler.Health)
	e.POST("/webhook", webhookHandler)

	// サーバー起動
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server start failed", "err", err)
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
