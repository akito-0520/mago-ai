package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

var (
	// ErrTokenNotFound は register_tokens に該当する未使用トークンが見つからないことを示す。
	ErrTokenNotFound = errors.New("register token not found")

	// ErrTokenExpired は指定された token の有効期限が切れていることを示す。
	ErrTokenExpired = errors.New("register token expired")

	// ErrLineUserExists は既にユーザーが登録済みであることを示す。
	ErrLineUserExists = errors.New("line user already registered")
)

// RegisterLineUserByToken は依存関係を管理する struct
type RegisterLineUserByToken struct {
	lineUsers      LineUserRepository
	registerTokens RegisterTokenRepository
}

// NewRegisterLineUserByToken はコンストラクタを作成する関数
func NewRegisterLineUserByToken(lineUsers LineUserRepository, registerTokens RegisterTokenRepository) *RegisterLineUserByToken {
	return &RegisterLineUserByToken{lineUsers: lineUsers, registerTokens: registerTokens}
}

// Execute はユーザーの未登録を確認した後にトークンの有効性を確認し、新規ユーザーを作成するメソッド。
func (uc *RegisterLineUserByToken) Execute(ctx context.Context, lineUserID string, token string) error {
	// 登録済みかを確認
	exists, err := uc.lineUsers.ExistsByLineUserID(ctx, lineUserID)
	if err != nil {
		return err
	}
	if exists {
		return ErrLineUserExists
	}

	// トークンの有効期限の確認
	rt, err := uc.registerTokens.FindUnusedByToken(ctx, token)
	if err != nil {
		return err
	}
	if rt == nil {
		return ErrTokenNotFound
	}

	// 期限切れかをチェック
	if time.Now().After(rt.ExpiresAt) {
		return ErrTokenExpired
	}

	// 新規ユーザーを作成
	user := domain.LineUser{
		AdminID:    rt.AdminID,
		LineUserID: lineUserID,
	}
	if err = uc.lineUsers.Create(ctx, user); err != nil {
		return err
	}

	// トークンを使用済みに変更
	err = uc.registerTokens.MarkUsed(ctx, token, lineUserID)
	if err != nil {
		return err
	}

	return nil
}
