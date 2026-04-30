package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// AdminLineLinkRepository は usecase.AdminLineLinkRepository の Postgres 実装。
type AdminLineLinkRepository struct {
	db *sqlx.DB
}

var _ usecase.AdminLineLinkRepository = (*AdminLineLinkRepository)(nil)

// NewAdminLineLinkRepository は AdminLineLinkRepository を生成する。
func NewAdminLineLinkRepository(db *sqlx.DB) *AdminLineLinkRepository {
	return &AdminLineLinkRepository{db: db}
}

// FindByLineUserID は通知 Bot 友だちの孫を探す。見つからなければ (nil, nil)。
func (r *AdminLineLinkRepository) FindByLineUserID(ctx context.Context, lineUserID string) (*domain.AdminLineLink, error) {
	var link domain.AdminLineLink
	err := r.db.GetContext(ctx, &link,
		`SELECT id, admin_id, line_user_id, display_name, created_at
           FROM admin_line_links
          WHERE line_user_id = $1`,
		lineUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// FindByAdminID は admin に紐付いた連携を全て返す。
func (r *AdminLineLinkRepository) FindByAdminID(ctx context.Context, adminID string) ([]domain.AdminLineLink, error) {
	rows := []domain.AdminLineLink{}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, admin_id, line_user_id, display_name, created_at
           FROM admin_line_links
          WHERE admin_id = $1
          ORDER BY created_at DESC`,
		adminID,
	)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Create は新規連携を INSERT し、生成された id を返す。
func (r *AdminLineLinkRepository) Create(ctx context.Context, link domain.AdminLineLink) (string, error) {
	var id string
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO admin_line_links (admin_id, line_user_id, display_name)
         VALUES ($1, $2, $3)
         RETURNING id`,
		link.AdminID, link.LineUserID, link.DisplayName,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}
