package usecase

import (
	"context"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

// LineGateway は LINE Messaging API への送信操作を抽象化する。
type LineGateway interface {
	Reply(ctx context.Context, replyToken string, text string) error
}

// LineUserRepository は line_users テーブルの操作を抽象化する。
type LineUserRepository interface {
	ExistsByLineUserID(ctx context.Context, lineUserID string) (bool, error)
	Create(ctx context.Context, user domain.LineUser) error
}

// RegisterTokenRepository は register_tokens テーブルの操作を抽象化する。
type RegisterTokenRepository interface {
	FindUnusedByToken(ctx context.Context, token string) (*domain.RegisterToken, error)
	MarkUsed(ctx context.Context, token string, lineUserID string) error
}
