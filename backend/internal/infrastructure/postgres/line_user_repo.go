package postgres

import (
	"context"
	"database/sql"
	"errors"

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

// ExistsActiveByLineUserID は line_user_id が現役（取り消されていない）として登録済みかを返す。
func (r *LineUserRepository) ExistsActiveByLineUserID(ctx context.Context, lineUserID string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(
            SELECT 1 FROM line_users
             WHERE line_user_id = $1
               AND revoked_at IS NULL
        )`,
		lineUserID,
	)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ExistsRevokedByLineUserID は line_user_id が取り消し済み状態で存在するかを返す。
func (r *LineUserRepository) ExistsRevokedByLineUserID(ctx context.Context, lineUserID string) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(
            SELECT 1 FROM line_users
             WHERE line_user_id = $1
               AND revoked_at IS NOT NULL
        )`,
		lineUserID,
	)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Upsert は line_users への INSERT、または取り消し済みユーザーの復活を行う。
//
// 動作:
//   - 行が存在しない場合 → INSERT
//   - 行が存在し revoked_at IS NOT NULL → UPDATE（admin_id 上書き、revoked_at = NULL、display_name 更新、created_at 更新）
//   - 行が存在し revoked_at IS NULL（現役）→ UPDATE 条件にマッチせず RETURNING が空 → ErrLineUserExists を返す
func (r *LineUserRepository) Upsert(ctx context.Context, user domain.LineUser) error {
	var id string
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO line_users (admin_id, line_user_id, display_name)
         VALUES ($1, $2, $3)
         ON CONFLICT (line_user_id) DO UPDATE
            SET admin_id     = EXCLUDED.admin_id,
                display_name = EXCLUDED.display_name,
                revoked_at   = NULL,
                created_at   = NOW()
         WHERE line_users.revoked_at IS NOT NULL
         RETURNING id`,
		user.AdminID, user.LineUserID, user.DisplayName,
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		// RETURNING が空 → 現役ユーザーで衝突
		return usecase.ErrLineUserExists
	}
	return err
}
