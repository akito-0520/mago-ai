package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

var (
	// ErrTokenNotFound は register_tokens に該当する未使用トークンが見つからないことを示す。
	ErrTokenNotFound = errors.New("register token not found")

	// ErrTokenExpired は指定された token の有効期限が切れていることを示す。
	ErrTokenExpired = errors.New("register token expired")

	// ErrLineUserExists は既に現役ユーザーが登録済みであることを示す。
	// UPSERT で取り消し済みでない行と衝突した場合に返される。
	ErrLineUserExists = errors.New("line user already registered")
)

// RegisterLineUserByToken は依存関係を管理する struct
type RegisterLineUserByToken struct {
	lineUsers      LineUserRepository
	registerTokens RegisterTokenRepository
	line           LineGateway
}

// NewRegisterLineUserByToken はコンストラクタ
func NewRegisterLineUserByToken(
	lineUsers LineUserRepository,
	registerTokens RegisterTokenRepository,
	line LineGateway,
) *RegisterLineUserByToken {
	return &RegisterLineUserByToken{
		lineUsers:      lineUsers,
		registerTokens: registerTokens,
		line:           line,
	}
}

// Execute はトークンを検証し、LINE プロフィールを取得して line_users に Upsert する。
// 取り消し済みのユーザーの場合は復活し、新規の場合は INSERT する。
// 既に現役のユーザーが存在する場合は ErrLineUserExists を返す。
func (uc *RegisterLineUserByToken) Execute(ctx context.Context, lineUserID string, token string) error {
	// トークン検証
	rt, err := uc.registerTokens.FindUnusedByToken(ctx, token)
	if err != nil {
		return err
	}
	if rt == nil {
		return ErrTokenNotFound
	}
	if time.Now().After(rt.ExpiresAt) {
		return ErrTokenExpired
	}

	// LINE プロフィール取得（ベストエフォート）
	displayName := uc.fetchDisplayName(ctx, lineUserID)

	// Upsert（新規 or 復活）
	user := domain.LineUser{
		AdminID:     rt.AdminID,
		LineUserID:  lineUserID,
		DisplayName: displayName,
	}
	if err := uc.lineUsers.Upsert(ctx, user); err != nil {
		return err
	}

	// トークン使用済みに
	return uc.registerTokens.MarkUsed(ctx, token, lineUserID)
}

// fetchDisplayName は LINE プロフィールを取得して DisplayName を返す。
// 取得失敗・空文字の場合は nil を返す（display_name は NULL にする想定）。
func (uc *RegisterLineUserByToken) fetchDisplayName(ctx context.Context, lineUserID string) *string {
	profile, err := uc.line.GetProfile(ctx, lineUserID)
	if err != nil {
		slog.Warn("get profile failed", "err", err, "lineUserID", lineUserID)
		return nil
	}
	if profile == nil || profile.DisplayName == "" {
		return nil
	}
	name := profile.DisplayName
	return &name
}
