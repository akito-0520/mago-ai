package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/labstack/echo/v4"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Webhook は LINE Platform からの Webhook リクエストを処理する。
// 署名検証 + イベントパースを SDK の webhook.ParseRequest で行う。
func Webhook(line usecase.LineGateway, channelSecret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		cb, err := webhook.ParseRequest(channelSecret, c.Request())
		if err != nil {
			slog.Warn("webhook parse failed", "err", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
		}

		for _, event := range cb.Events {
			go handleEvent(line, event)
		}

		return c.NoContent(http.StatusOK)
	}
}

func handleEvent(line usecase.LineGateway, event webhook.EventInterface) {
	switch e := event.(type) {
	case webhook.MessageEvent:
		handleMessageEvent(line, e)
	case webhook.FollowEvent:
		slog.Info("friend added")
	case webhook.UnfollowEvent:
		slog.Info("unfollowed")
	default:
		slog.Warn("unknown event type", "type", fmt.Sprintf("%T", event))
	}
}

func handleMessageEvent(line usecase.LineGateway, e webhook.MessageEvent) {
	switch msg := e.Message.(type) {
	case webhook.TextMessageContent:
		// メッセージのレスポンスを組み立てる
		err := line.Reply(context.Background(), e.ReplyToken, msg.Text)

		if err != nil {
			slog.Error("reply failed", "err", err)
			return
		}
		slog.Info("replied", "text", msg.Text)
	case webhook.StickerMessageContent:
		// メッセージのレスポンスを組み立てる
		err := line.Reply(context.Background(), e.ReplyToken, "こんにちは。お孫さんに聞きたいことを文字で送ってもらえますか？")
		if err != nil {
			slog.Error("reply failed", "err", err)
			return
		}
		slog.Info("sticker message received")
	default:
		slog.Warn("unknown message type", "type", fmt.Sprintf("%T", msg))
	}
}
