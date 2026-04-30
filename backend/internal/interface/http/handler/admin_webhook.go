package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// AdminMessageResponder は通知 Bot 経由の管理者メッセージへの応答ロジックを抽象化する。
// 具象実装は usecase.RespondToAdminLineMessage が担う。
type AdminMessageResponder interface {
	Execute(ctx context.Context, lineUserID, text string) error
}

// AdminWebhook は通知 LINE 公式アカウントからの Webhook を受け付けるハンドラー。
// 主な役割は連携用の 6 桁トークンの受信。
//
// 通常の Bot（おばあちゃん用）とは別の Channel Secret を使う。
func AdminWebhook(channelSecret string, respond AdminMessageResponder) echo.HandlerFunc {
	return func(c echo.Context) error {
		cb, err := webhook.ParseRequest(channelSecret, c.Request())
		if err != nil {
			slog.Warn("admin webhook parse failed", "err", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
		}

		for _, event := range cb.Events {
			ctx := context.Background()
			go handleAdminEvent(ctx, respond, event)
		}

		return c.NoContent(http.StatusOK)
	}
}

// handleAdminEvent は通知 Bot 用のイベント振り分け。
func handleAdminEvent(
	ctx context.Context,
	respond AdminMessageResponder,
	event webhook.EventInterface,
) {
	switch e := event.(type) {
	case webhook.MessageEvent:
		handleAdminMessageEvent(ctx, respond, e)
	case webhook.FollowEvent:
		slog.Info("admin: friend added")
	case webhook.UnfollowEvent:
		slog.Info("admin: unfollowed")
	default:
		slog.Warn("admin: unknown event type", "type", fmt.Sprintf("%T", event))
	}
}

// handleAdminMessageEvent は管理者から受信したメッセージを usecase に流す。
func handleAdminMessageEvent(
	ctx context.Context,
	respond AdminMessageResponder,
	e webhook.MessageEvent,
) {
	switch msg := e.Message.(type) {
	case webhook.TextMessageContent:
		senderID := getSenderUserID(e)
		if senderID == "" {
			slog.Warn("admin: sender user id missing")
			return
		}
		if err := respond.Execute(ctx, senderID, msg.Text); err != nil {
			slog.Error("admin respond failed", "err", err)
		}
	default:
		slog.Warn("admin: unknown message type", "type", fmt.Sprintf("%T", msg))
	}
}
