package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// AdminLinkTokenRepository は usecase.AdminLinkTokenRepository の Postgres 実装。
type AdminLinkTokenRepository struct {
	db *sqlx.DB
}

var _ usecase.AdminLinkTokenRepository = (*AdminLinkTokenRepository)(nil)

// NewAdminLinkTokenRepository は AdminLinkTokenRepository を生成する。
func NewAdminLinkTokenRepository(db *sqlx.DB) *AdminLinkTokenRepository {
	return &AdminLinkTokenRepository{db: db}
}

// FindUnusedByToken は未使用の admin_link_token を返す。
// 見つからなければ (nil, nil)。期限切れチェックは usecase 層で行う。
func (r *AdminLinkTokenRepository) FindUnusedByToken(ctx context.Context, token string) (*domain.AdminLinkToken, error) {
	var t domain.AdminLinkToken
	err := r.db.GetContext(ctx, &t,
		`SELECT token, admin_id, expires_at, used_at, used_by, created_at
           FROM admin_link_tokens
          WHERE token = $1 AND used_at IS NULL`,
		token,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkUsed は token を使用済みにマークする。
// linkID には作成された admin_line_links.id (UUID) を渡す。
func (r *AdminLinkTokenRepository) MarkUsed(ctx context.Context, token string, linkID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE admin_link_tokens
            SET used_at = now(),
                used_by = $2
          WHERE token = $1`,
		token, linkID,
	)
	return err
}
