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

// AdminNotifier は管理者（孫）への LINE Push 通知を抽象化する。
// 通知用 LINE 公式アカウント経由で Push API を呼び出す。
type AdminNotifier interface {
	// Push は指定の LINE User ID に通知メッセージを送る（Reply ではなく Push）。
	Push(ctx context.Context, lineUserID string, text string) error
	// GetProfile は通知用チャネル経由で LINE プロフィールを取得する。
	GetProfile(ctx context.Context, lineUserID string) (*LineProfile, error)
}

// AdminLineLinkRepository は admin_line_links テーブルの操作を抽象化する。
type AdminLineLinkRepository interface {
	// FindByLineUserID は通知 Bot 友だちの孫を探す。見つからなければ (nil, nil)。
	FindByLineUserID(ctx context.Context, lineUserID string) (*domain.AdminLineLink, error)
	// FindByAdminID は admin に紐付いた連携を全て返す（複数 LINE 端末対応）。
	FindByAdminID(ctx context.Context, adminID string) ([]domain.AdminLineLink, error)
	// Create は新規連携を INSERT する。
	Create(ctx context.Context, link domain.AdminLineLink) (string, error)
}

// AdminLinkTokenRepository は admin_link_tokens テーブルの操作を抽象化する。
type AdminLinkTokenRepository interface {
	// FindUnusedByToken は未使用のトークンを返す。見つからなければ (nil, nil)。
	FindUnusedByToken(ctx context.Context, token string) (*domain.AdminLinkToken, error)
	// MarkUsed は token を使用済みにマークする（used_at = now(), used_by = linkID）。
	MarkUsed(ctx context.Context, token string, linkID string) error
}

// LineUserRepository は line_users テーブルの操作を抽象化する。
type LineUserRepository interface {
	FindActiveByLineUserID(ctx context.Context, lineUserID string) (*domain.LineUser, error)
	ExistsRevokedByLineUserID(ctx context.Context, lineUserID string) (bool, error)
	// CountActiveByAdminID は admin に紐付く現役（取り消されていない）ユーザー数を返す。
	// プラン上限チェック用。
	CountActiveByAdminID(ctx context.Context, adminID string) (int, error)
	Upsert(ctx context.Context, user domain.LineUser) error
	UpdateSessionResetAt(ctx context.Context, lineUserID string) error
}

// RegisterTokenRepository は register_tokens テーブルの操作を抽象化する。
type RegisterTokenRepository interface {
	FindUnusedByToken(ctx context.Context, token string) (*domain.RegisterToken, error)
	MarkUsed(ctx context.Context, token string, lineUserID string) error
}

// PlanRepository はプラン情報を取得する Repository。
//
// 内部実装はキャッシュを持つ前提（プラン情報は滅多に変わらない）。
// FindByAdminID は admin_plans.plan_code → plans を辿って Plan を解決する。
// admin_plans に行が無い場合は domain.DefaultPlanCode を使う。
type PlanRepository interface {
	// FindByAdminID は admin_id に紐付くプランを返す。
	// 行が無ければ DefaultPlanCode (free) のプランを返す（エラーにはしない）。
	FindByAdminID(ctx context.Context, adminID string) (*domain.Plan, error)
}

// QuotaResult は QuotaService.Allow の戻り値。
type QuotaResult struct {
	Allowed bool        // 許可されたか
	Plan    domain.Plan // 判定に使ったプラン（拒否時のメッセージ生成等に使う）
}

// QuotaService はプランに基づいたレート制限の判定サービス。
//
// 内部実装は PlanRepository + RateLimiter のコンポジション。
// usecase 層からはこの interface 越しに「叩いていいか」だけ問い合わせる。
type QuotaService interface {
	// Allow は admin の現在の使用状況とプランから、新たな Claude 呼び出しを許可するか判定する。
	// 許可なら内部のカウンタに now を記録する。
	Allow(ctx context.Context, adminID string, now time.Time) (*QuotaResult, error)
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
