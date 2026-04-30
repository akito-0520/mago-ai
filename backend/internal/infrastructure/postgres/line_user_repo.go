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

// FindActiveByLineUserID は現役（取り消されていない）の line_user 行を返す。
// 行が見つからない / 取り消し済みの場合は (nil, nil) を返す。
func (r *LineUserRepository) FindActiveByLineUserID(ctx context.Context, lineUserID string) (*domain.LineUser, error) {
	var u domain.LineUser
	err := r.db.GetContext(ctx, &u,
		`SELECT id, admin_id, line_user_id, display_name, session_reset_at, revoked_at, created_at
           FROM line_users
          WHERE line_user_id = $1
            AND revoked_at IS NULL`,
		lineUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
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

// CountActiveByAdminID は admin に紐付く現役ユーザー数を返す。
func (r *LineUserRepository) CountActiveByAdminID(ctx context.Context, adminID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*)::int
		   FROM line_users
		  WHERE admin_id   = $1
		    AND revoked_at IS NULL`,
		adminID,
	)
	if err != nil {
		return 0, err
	}
	return count, nil
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
		return usecase.ErrLineUserExists
	}
	return err
}

// UpdateSessionResetAt は line_user_id に該当する現役ユーザーの session_reset_at を NOW() に更新する。
func (r *LineUserRepository) UpdateSessionResetAt(ctx context.Context, lineUserID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE line_users
            SET session_reset_at = NOW()
          WHERE line_user_id = $1
            AND revoked_at IS NULL`,
		lineUserID,
	)
	return err
}
