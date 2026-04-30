package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

var (
	// ErrAdminLinkTokenNotFound は admin_link_tokens に該当する未使用トークンが見つからない。
	ErrAdminLinkTokenNotFound = errors.New("admin link token not found")

	// ErrAdminLinkTokenExpired は admin_link_tokens の有効期限が切れている。
	ErrAdminLinkTokenExpired = errors.New("admin link token expired")

	// ErrAdminLineLinkExists は既に同じ LINE User ID が紐付け済み。
	ErrAdminLineLinkExists = errors.New("admin line link already exists")
)

// LinkAdminLineByToken は孫の LINE User ID と admin を紐付ける usecase。
// 通知 Bot で「6 桁トークン」を受信したときに呼ばれる。
type LinkAdminLineByToken struct {
	links    AdminLineLinkRepository
	tokens   AdminLinkTokenRepository
	notifier AdminNotifier
}

// NewLinkAdminLineByToken はコンストラクタ。
func NewLinkAdminLineByToken(
	links AdminLineLinkRepository,
	tokens AdminLinkTokenRepository,
	notifier AdminNotifier,
) *LinkAdminLineByToken {
	return &LinkAdminLineByToken{
		links:    links,
		tokens:   tokens,
		notifier: notifier,
	}
}

// Execute はトークンを検証して admin_line_links を作成する。
//   - 既に紐付け済みなら ErrAdminLineLinkExists
//   - トークンが見つからない / 期限切れなら対応するエラー
//   - 成功なら nil
func (uc *LinkAdminLineByToken) Execute(ctx context.Context, lineUserID string, token string) error {
	// 既に紐付け済みかチェック
	existing, err := uc.links.FindByLineUserID(ctx, lineUserID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrAdminLineLinkExists
	}

	// トークン検証
	t, err := uc.tokens.FindUnusedByToken(ctx, token)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrAdminLinkTokenNotFound
	}
	if time.Now().After(t.ExpiresAt) {
		return ErrAdminLinkTokenExpired
	}

	// LINE プロフィール取得（ベストエフォート）
	displayName := uc.fetchDisplayName(ctx, lineUserID)

	// admin_line_links に INSERT
	link := domain.AdminLineLink{
		AdminID:     t.AdminID,
		LineUserID:  lineUserID,
		DisplayName: displayName,
	}
	linkID, err := uc.links.Create(ctx, link)
	if err != nil {
		return err
	}

	// トークンを使用済みに
	return uc.tokens.MarkUsed(ctx, token, linkID)
}

// fetchDisplayName は LINE プロフィールから DisplayName を取得する（ベストエフォート）。
func (uc *LinkAdminLineByToken) fetchDisplayName(ctx context.Context, lineUserID string) *string {
	profile, err := uc.notifier.GetProfile(ctx, lineUserID)
	if err != nil {
		slog.Warn("admin line: get profile failed", "err", err, "lineUserID", lineUserID)
		return nil
	}
	if profile == nil || profile.DisplayName == "" {
		return nil
	}
	name := profile.DisplayName
	return &name
}
