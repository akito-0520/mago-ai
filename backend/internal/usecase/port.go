package usecase

import "context"

// LineGateway は LINE Messaging API への送信操作を抽象化する。
type LineGateway interface {
	Reply(ctx context.Context, replyToken string, text string) error
}
