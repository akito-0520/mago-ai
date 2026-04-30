package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/ratelimit"
)

// quotaService は QuotaService の具象実装。
//
// PlanRepository でプランを解決し、ratelimit.Limiter にプランの制限値を渡して判定する。
// プラン情報のキャッシュは PlanRepository 側、ウィンドウのカウントは Limiter 側が責任を持つ。
type quotaService struct {
	plans   PlanRepository
	limiter *ratelimit.Limiter
}

var _ QuotaService = (*quotaService)(nil)

// NewQuotaService は quotaService を生成する。
//
// limiter は外部から渡す（main.go で 1 つだけ作って共有する想定）。
// 1 プロセス内で複数の Limiter を作るとカウントが分散して意味がない。
func NewQuotaService(plans PlanRepository, limiter *ratelimit.Limiter) QuotaService {
	return &quotaService{
		plans:   plans,
		limiter: limiter,
	}
}

// Allow は admin のプランを解決して制限値を取り、Limiter に判定を委譲する。
//
// プラン解決でエラー（DB ダウン等）が起きた場合はそのまま返す。
// 呼び出し側はエラー時の挙動を選べる（フェイルクローズで拒否扱いにする等）。
func (s *quotaService) Allow(ctx context.Context, adminID string, now time.Time) (*QuotaResult, error) {
	plan, err := s.plans.FindByAdminID(ctx, adminID)
	if err != nil {
		return nil, fmt.Errorf("resolve plan: %w", err)
	}

	limits := ratelimit.Limits{
		Hourly: plan.HourlyLimit,
		Daily:  plan.DailyLimit,
	}
	allowed := s.limiter.Allow(adminID, limits, now)

	return &QuotaResult{
		Allowed: allowed,
		Plan:    *plan,
	}, nil
}
