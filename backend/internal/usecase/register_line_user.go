package usecase

import (
	"context"
	"errors"
	"fmt"
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

	// ErrPlanMaxLineUsersReached はプランの最大登録人数に達していることを示す。
	// 通常は generate_register_token RPC 側で先に弾かれるが、token 発行後〜登録までの間に
	// 別ユーザー登録 / 取り消し済みユーザーの復活で枠が埋まるケースに備えた防御的チェック。
	ErrPlanMaxLineUsersReached = errors.New("plan max line users reached")
)

// RegisterLineUserByToken は依存関係を管理する struct
type RegisterLineUserByToken struct {
	lineUsers      LineUserRepository
	registerTokens RegisterTokenRepository
	plans          PlanRepository
	line           LineGateway
}

// NewRegisterLineUserByToken はコンストラクタ
func NewRegisterLineUserByToken(
	lineUsers LineUserRepository,
	registerTokens RegisterTokenRepository,
	plans PlanRepository,
	line LineGateway,
) *RegisterLineUserByToken {
	return &RegisterLineUserByToken{
		lineUsers:      lineUsers,
		registerTokens: registerTokens,
		plans:          plans,
		line:           line,
	}
}

// Execute はトークンを検証し、LINE プロフィールを取得して line_users に Upsert する。
//
// 流れ:
//  1. token 検証（未使用 / 期限内）
//  2. プラン解決 + 現役 line_users 数チェック（上限到達なら ErrPlanMaxLineUsersReached）
//  3. LINE プロフィール取得（ベストエフォート）
//  4. Upsert（新規 INSERT or 取り消し済みからの復活）
//  5. token を used に更新
//
// なお (2) は generate_register_token RPC でも一次チェックされるが、
// token 発行後〜登録までの間にプラン枠が埋まる可能性があるため、ここでも防御的に検証する。
func (uc *RegisterLineUserByToken) Execute(ctx context.Context, lineUserID string, token string) error {
	// (1) トークン検証
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

	// (2) プラン枠チェック
	if err := uc.checkPlanCapacity(ctx, rt.AdminID); err != nil {
		return err
	}

	// (3) LINE プロフィール取得（ベストエフォート）
	displayName := uc.fetchDisplayName(ctx, lineUserID)

	// (4) Upsert（新規 or 復活）
	user := domain.LineUser{
		AdminID:     rt.AdminID,
		LineUserID:  lineUserID,
		DisplayName: displayName,
	}
	if err := uc.lineUsers.Upsert(ctx, user); err != nil {
		return err
	}

	// (5) トークン使用済みに
	return uc.registerTokens.MarkUsed(ctx, token, lineUserID)
}

// checkPlanCapacity は admin の現役 line_users 数がプラン上限に達していないかを検証する。
// 上限到達時は ErrPlanMaxLineUsersReached を返す。
func (uc *RegisterLineUserByToken) checkPlanCapacity(ctx context.Context, adminID string) error {
	plan, err := uc.plans.FindByAdminID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("resolve plan for admin %s: %w", adminID, err)
	}
	count, err := uc.lineUsers.CountActiveByAdminID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("count active line users for admin %s: %w", adminID, err)
	}
	if count >= plan.MaxLineUsers {
		slog.Info("register blocked by plan limit",
			"adminID", adminID,
			"plan", plan.Code,
			"max", plan.MaxLineUsers,
			"current", count,
		)
		return ErrPlanMaxLineUsersReached
	}
	return nil
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
