package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// RegisterTokenRepository は usecase.RegisterTokenRepository の Postgres 実装。
type RegisterTokenRepository struct {
	db *sqlx.DB
}

// この struct がその interface を満たしているかコンパイラに確認させる
var _ usecase.RegisterTokenRepository = (*RegisterTokenRepository)(nil)

// NewRegisterTokenRepository は RegisterTokenRepository を生成する。
func NewRegisterTokenRepository(db *sqlx.DB) *RegisterTokenRepository {
	return &RegisterTokenRepository{db: db}
}

// FindUnusedByToken は未使用の register_token を返す。
// 見つからなければ (nil, nil)。期限切れチェックは usecase 層で行う。
func (r *RegisterTokenRepository) FindUnusedByToken(ctx context.Context, token string) (*domain.RegisterToken, error) {
	var rt domain.RegisterToken
	err := r.db.GetContext(ctx, &rt,
		`SELECT token, admin_id, expires_at, used_at, used_by, created_at
           FROM register_tokens
          WHERE token = $1 AND used_at IS NULL`,
		token,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// MarkUsed は token を使用済みにマークする。
// lineUserID（LINE の生 ID）から line_users.id を引いて used_by に入れる。
func (r *RegisterTokenRepository) MarkUsed(ctx context.Context, token string, lineUserID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE register_tokens
            SET used_at = now(),
                used_by = (SELECT id FROM line_users WHERE line_user_id = $2)
          WHERE token = $1`,
		token, lineUserID,
	)
	return err
}
