package postgres

import (
	"context"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// LineUserRepository は usecase.LineUserRepository の Postgres 実装。
type LineUserRepository struct {
	db *sqlx.DB
}

// この struct がその interface を満たしているかコンパイラに確認させる
var _ usecase.LineUserRepository = (*LineUserRepository)(nil)

// NewLineUserRepository は LineUserRepository を生成する。
func NewLineUserRepository(db *sqlx.DB) *LineUserRepository {
	return &LineUserRepository{db: db}
}

// ExistsByLineUserID は line_user_id が登録済みかを返す。
func (r *LineUserRepository) ExistsByLineUserID(ctx context.Context, lineUserID string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM line_users WHERE line_user_id = $1)`,
		lineUserID,
	)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Create は line_users に新規行を INSERT する。
func (r *LineUserRepository) Create(ctx context.Context, user domain.LineUser) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO line_users (admin_id, line_user_id, display_name) VALUES ($1, $2, $3)`,
		user.AdminID, user.LineUserID, user.DisplayName,
	)
	return err
}
