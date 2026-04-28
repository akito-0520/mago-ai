package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// MessageResponder は受信メッセージへの応答ロジックを抽象化する。
// 具象実装は usecase.RespondToIncomingMessage が担う。
type MessageResponder interface {
	Execute(ctx context.Context, lineUserID, replyToken, text string) error
}

// Webhook は LINE Platform からの Webhook リクエストを処理する。
// 署名検証 + イベントパースを SDK の webhook.ParseRequest で行う。
func Webhook(
	channelSecret string,
	respond MessageResponder,
) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.Background()

		cb, err := webhook.ParseRequest(channelSecret, c.Request())
		if err != nil {
			slog.Warn("webhook parse failed", "err", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
		}

		for _, event := range cb.Events {
			go handleEvent(ctx, event, respond)
		}

		return c.NoContent(http.StatusOK)
	}
}

func handleEvent(
	ctx context.Context,
	event webhook.EventInterface,
	respond MessageResponder,
) {
	switch e := event.(type) {
	case webhook.MessageEvent:
		handleMessageEvent(ctx, e, respond)
	case webhook.FollowEvent:
		slog.Info("friend added")
	case webhook.UnfollowEvent:
		slog.Info("unfollowed")
	default:
		slog.Warn("unknown event type", "type", fmt.Sprintf("%T", event))
	}
}

func handleMessageEvent(
	ctx context.Context,
	e webhook.MessageEvent,
	respond MessageResponder,
) {
	switch msg := e.Message.(type) {
	case webhook.TextMessageContent:
		senderID := getSenderUserID(e)
		if senderID == "" {
			slog.Warn("sender user id missing")
			return
		}

		// メッセージのレスポンスを組み立てる
		err := respond.Execute(ctx, senderID, e.ReplyToken, msg.Text)
		if err != nil {
			slog.Error("reply failed", "err", err)
			return
		}
		slog.Info("replied", "text", msg.Text)

	case webhook.StickerMessageContent:
		senderID := getSenderUserID(e)
		if senderID == "" {
			slog.Warn("sender user id missing")
			return
		}

		// メッセージのレスポンスを組み立てる
		err := respond.Execute(ctx, senderID, e.ReplyToken, "こんにちは。私に聞きたいことを文字で送ってもらえますか？")
		if err != nil {
			slog.Error("reply failed", "err", err)
			return
		}
		slog.Info("sticker message received")

	default:
		slog.Warn("unknown message type", "type", fmt.Sprintf("%T", msg))
	}
}

// getSenderUserID は webhook イベントから送信者の LINE User ID を取り出す。
// e.Source が UserSource のときだけ取れる。
func getSenderUserID(e webhook.MessageEvent) string {
	if src, ok := e.Source.(webhook.UserSource); ok {
		return src.UserId
	}
	return ""
}
