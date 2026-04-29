package usecase

import (
	"context"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

// LineProfile は LINE Platform から取得したユーザープロフィール情報。
type LineProfile struct {
	UserID      string
	DisplayName string
}

// LineGateway は LINE Messaging API への送信操作を抽象化する。
type LineGateway interface {
	Reply(ctx context.Context, replyToken string, text string) error
	GetProfile(ctx context.Context, lineUserID string) (*LineProfile, error)
}

// LineUserRepository は line_users テーブルの操作を抽象化する。
type LineUserRepository interface {
	FindActiveByLineUserID(ctx context.Context, lineUserID string) (*domain.LineUser, error)
	ExistsRevokedByLineUserID(ctx context.Context, lineUserID string) (bool, error)
	Upsert(ctx context.Context, user domain.LineUser) error
	UpdateSessionResetAt(ctx context.Context, lineUserID string) error
}

// RegisterTokenRepository は register_tokens テーブルの操作を抽象化する。
type RegisterTokenRepository interface {
	FindUnusedByToken(ctx context.Context, token string) (*domain.RegisterToken, error)
	MarkUsed(ctx context.Context, token string, lineUserID string) error
}

// ConversationRepository は conversations テーブルの操作を抽象化する。
type ConversationRepository interface {
	// Recent は cutoff より新しい行を最大 limit 件、新しい順に並べた状態で返す。
	// （呼び出し側で時系列順に並べ替える必要あり）
	Recent(ctx context.Context, lineUserID string, since time.Time, limit int) ([]domain.Conversation, error)

	// CreateUser は user role の発言を保存する。
	CreateUser(ctx context.Context, lineUserID string, content string) error

	// CreateAssistant は assistant role の発言をメタ情報とともに保存する。
	CreateAssistant(ctx context.Context, lineUserID string, content string, meta domain.AssistantMeta) error
}

// ClaudeRequest は ClaudeGateway.Complete に渡す入力。
type ClaudeRequest struct {
	SystemPrompt string
	History      []ClaudeMessage // 過去ターン（時系列順）
	UserMessage  string          // 今回のユーザー発言
}

// ClaudeMessage は Claude API に送る 1 ターン分の発言。
type ClaudeMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ClaudeResponse は ClaudeGateway.Complete の戻り値。
type ClaudeResponse struct {
	Content                  string
	Model                    string
	LatencyMs                int
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
}

// ClaudeGateway は Claude API への送信操作を抽象化する。
type ClaudeGateway interface {
	Complete(ctx context.Context, req ClaudeRequest) (*ClaudeResponse, error)
}
