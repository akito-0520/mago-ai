package linebot

import (
	"context"

	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// Client は LineGateway の LINE SDK ベース実装。
type Client struct {
	api *messaging_api.MessagingApiAPI
}

// New は ChannelAccessToken から Client を生成する。
func New(channelAccessToken string) (*Client, error) {
	api, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	if err != nil {
		return nil, err
	}
	return &Client{api: api}, nil
}

// Reply は指定の reply token に対してテキストメッセージを返信する。
func (c *Client) Reply(ctx context.Context, replyToken string, text string) error {
	_, err := c.api.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{Text: text},
			},
		},
	)
	return err
}

var _ usecase.LineGateway = (*Client)(nil)
