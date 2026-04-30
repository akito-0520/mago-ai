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
	"github.com/akito-0520/mago-ai/backend/internal/infrastructure/claude"
	"github.com/akito-0520/mago-ai/backend/internal/infrastructure/linebot"
	"github.com/akito-0520/mago-ai/backend/internal/infrastructure/postgres"
	"github.com/akito-0520/mago-ai/backend/internal/interface/http/handler"
	"github.com/akito-0520/mago-ai/backend/internal/ratelimit"
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
	e.Use(middleware.Recover())

	// DB 接続
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("postgres close failed", "err", err)
		}
	}()

	// LINE クライアント（おばあちゃん用）
	lineClient, err := linebot.New(cfg.LineChannelAccessToken)
	if err != nil {
		slog.Error("linebot setup failed", "err", err)
		os.Exit(1)
	}

	// LINE 通知クライアント（孫用、別チャネル）
	notifierClient, err := linebot.NewNotifier(cfg.LineNotifyChannelAccessToken)
	if err != nil {
		slog.Error("line notifier setup failed", "err", err)
		os.Exit(1)
	}

	// Claude クライアント
	claudeClient := claude.New(cfg.AnthropicAPIKey, cfg.ClaudeModel)

	// Repository
	lineUsers := postgres.NewLineUserRepository(db)
	registerTokens := postgres.NewRegisterTokenRepository(db)
	conversations := postgres.NewConversationRepository(db)
	adminLinks := postgres.NewAdminLineLinkRepository(db)
	adminLinkTokens := postgres.NewAdminLinkTokenRepository(db)
	plans := postgres.NewPlanRepository(db)

	// レート制限：プロセス全体で 1 つの Limiter を共有（adminID 単位でカウント）
	limiter := ratelimit.New()
	quotaService := usecase.NewQuotaService(plans, limiter)

	// Usecase
	registerUC := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens, plans, lineClient)
	respondUC := usecase.NewRespondToIncomingMessage(
		lineUsers,
		conversations,
		adminLinks,
		lineClient,
		claudeClient,
		notifierClient,
		quotaService,
		registerUC,
		cfg.ConversationWindow,
	)
	linkAdminUC := usecase.NewLinkAdminLineByToken(adminLinks, adminLinkTokens, notifierClient)
	respondAdminUC := usecase.NewRespondToAdminLineMessage(notifierClient, linkAdminUC)

	// Handler
	webhookHandler := handler.Webhook(cfg.LineChannelSecret, respondUC)
	adminWebhookHandler := handler.AdminWebhook(cfg.LineNotifyChannelSecret, respondAdminUC)

	// ルートの登録
	e.GET("/healthz", handler.Health)
	e.POST("/webhook", webhookHandler)
	e.POST("/webhook/admin", adminWebhookHandler)

	// サーバー起動
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server start failed", "err", err)
			os.Exit(1)
		}
	}()

	// ctrl または kill をリッスン
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	// Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	slog.Info("shutdown started")
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
}
