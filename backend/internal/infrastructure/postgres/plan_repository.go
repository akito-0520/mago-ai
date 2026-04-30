package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// defaultPlanCacheTTL は PlanRepository のキャッシュ有効期間。
//
// プラン情報は migration 経由でしか変わらないため長めで OK。
// 将来 Stripe webhook を導入したら、特定 admin のキャッシュを明示削除する API を
// 追加して即時反映する想定（その場合も DB 変更後の最大遅延はこの TTL）。
const defaultPlanCacheTTL = 5 * time.Minute

// PlanRepository は usecase.PlanRepository の Postgres 実装。
//
// admin_plans → plans を 1 クエリで JOIN し Plan を返す。
// admin_plans に行が無い場合は DefaultPlanCode の plan を返す。
//
// adminID をキーにしたインメモリキャッシュ (sync.Map) を持ち、TTL 経過後に再取得する。
type PlanRepository struct {
	db    *sqlx.DB
	cache sync.Map // map[string]cachedPlan : adminID → cachedPlan
	ttl   time.Duration
}

// cachedPlan はキャッシュ 1 エントリの内容。
type cachedPlan struct {
	plan      domain.Plan
	expiresAt time.Time
}

var _ usecase.PlanRepository = (*PlanRepository)(nil)

// NewPlanRepository は PlanRepository を生成する。
func NewPlanRepository(db *sqlx.DB) *PlanRepository {
	return &PlanRepository{db: db, ttl: defaultPlanCacheTTL}
}

// FindByAdminID は admin_id に紐付くプランを返す。
//
// 解決ロジック (1 クエリ):
//
//	plans p
//	WHERE p.code = COALESCE(
//	  (SELECT plan_code FROM admin_plans WHERE admin_id = $1),
//	  $2  -- DefaultPlanCode
//	)
//
// admin_plans に行が無くても DefaultPlanCode に解決されるので、
// プランが見つからない (= 'free' すら未投入) ケース以外はエラーにならない。
func (r *PlanRepository) FindByAdminID(ctx context.Context, adminID string) (*domain.Plan, error) {
	// 1. キャッシュ参照（期限切れなら削除して取り直す）
	if v, ok := r.cache.Load(adminID); ok {
		cp := v.(cachedPlan)
		if time.Now().Before(cp.expiresAt) {
			plan := cp.plan
			return &plan, nil
		}
		r.cache.Delete(adminID)
	}

	// 2. DB から取得
	var p domain.Plan
	err := r.db.GetContext(ctx, &p,
		`SELECT
		   p.code,
		   p.display_name,
		   p.max_line_users,
		   p.hourly_limit,
		   p.daily_limit
		 FROM plans p
		 WHERE p.code = COALESCE(
		   (SELECT plan_code FROM admin_plans WHERE admin_id = $1),
		   $2
		 )`,
		adminID, domain.DefaultPlanCode,
	)
	if err != nil {
		return nil, err
	}

	// 3. キャッシュ保存
	r.cache.Store(adminID, cachedPlan{plan: p, expiresAt: time.Now().Add(r.ttl)})
	return &p, nil
}

// Invalidate は指定 adminID のキャッシュを破棄する。
//
// 将来 Stripe webhook 等でプラン変更があった時に呼び出す前提。
func (r *PlanRepository) Invalidate(adminID string) {
	r.cache.Delete(adminID)
}

// InvalidateAll は全キャッシュを破棄する。プラン定義変更時の手動利用想定。
func (r *PlanRepository) InvalidateAll() {
	r.cache.Range(func(key, _ any) bool {
		r.cache.Delete(key)
		return true
	})
}
