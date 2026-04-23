package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Webhook は LINE Platform からの Webhook リクエストを処理する。
// 署名検証 + イベントパースを SDK の webhook.ParseRequest で行う。
func Webhook(channelSecret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		cb, err := webhook.ParseRequest(channelSecret, c.Request())
		if err != nil {
			slog.Warn("webhook parse failed", "err", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
		}

		for _, event := range cb.Events {
			slog.Info("webhook event received", "event", event)
		}

		return c.NoContent(http.StatusOK)
	}
}
