package linebot

import (
	"context"

	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Notifier は usecase.AdminNotifier の LINE SDK 実装（通知専用チャネル用）。
//
// おばあちゃん用の Bot とは別の LINE 公式アカウント（Channel）を使う前提。
// Reply ではなく Push API でメッセージを送る。
type Notifier struct {
	api *messaging_api.MessagingApiAPI
}

var _ usecase.AdminNotifier = (*Notifier)(nil)

// NewNotifier は通知用チャネルの ChannelAccessToken から Notifier を生成する。
func NewNotifier(channelAccessToken string) (*Notifier, error) {
	api, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	if err != nil {
		return nil, err
	}
	return &Notifier{api: api}, nil
}

// Push は指定の LINE User ID にテキストメッセージを送る（Push API）。
// 友だち追加されていない / ブロックされている場合はエラーになる。
func (n *Notifier) Push(_ context.Context, lineUserID string, text string) error {
	_, err := n.api.PushMessage(
		&messaging_api.PushMessageRequest{
			To: lineUserID,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{Text: text},
			},
		},
		"",
	)
	return err
}

// GetProfile は通知用チャネル経由で LINE プロフィールを取得する。
func (n *Notifier) GetProfile(_ context.Context, lineUserID string) (*usecase.LineProfile, error) {
	profile, err := n.api.GetProfile(lineUserID)
	if err != nil {
		return nil, err
	}
	return &usecase.LineProfile{
		UserID:      profile.UserId,
		DisplayName: profile.DisplayName,
	}, nil
}
