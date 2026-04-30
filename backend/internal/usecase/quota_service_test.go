package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/ratelimit"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/stretchr/testify/require"
)

// --- fakePlanRepository ---

type fakePlanRepository struct {
	plan *domain.Plan
	err  error

	calls []string // 呼ばれた adminID の履歴
}

func (f *fakePlanRepository) FindByAdminID(_ context.Context, adminID string) (*domain.Plan, error) {
	f.calls = append(f.calls, adminID)
	if f.err != nil {
		return nil, f.err
	}
	return f.plan, nil
}

// テスト用の固定プラン
var (
	freePlan = domain.Plan{
		Code: "free", DisplayName: "無料",
		MaxLineUsers: 1, HourlyLimit: 5, DailyLimit: 30,
	}
)

func TestQuotaService_Allow(t *testing.T) {
	t.Run("初回呼び出しは Allowed=true", func(t *testing.T) {
		plans := &fakePlanRepository{plan: &freePlan}
		limiter := ratelimit.New()
		svc := usecase.NewQuotaService(plans, limiter)

		now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		res, err := svc.Allow(context.Background(), "admin-A", now)

		require.NoError(t, err)
		require.True(t, res.Allowed)
		require.Equal(t, freePlan, res.Plan)
		require.Equal(t, []string{"admin-A"}, plans.calls)
	})

	t.Run("hourly limit を超えると Allowed=false", func(t *testing.T) {
		plans := &fakePlanRepository{plan: &freePlan}
		limiter := ratelimit.New()
		svc := usecase.NewQuotaService(plans, limiter)

		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		ctx := context.Background()

		// hourly 5 回まで OK
		for i := 0; i < freePlan.HourlyLimit; i++ {
			res, err := svc.Allow(ctx, "admin-A", base.Add(time.Duration(i)*time.Minute))
			require.NoError(t, err)
			require.True(t, res.Allowed, "i=%d", i)
		}
		// 6 回目は Deny
		res, err := svc.Allow(ctx, "admin-A", base.Add(6*time.Minute))
		require.NoError(t, err)
		require.False(t, res.Allowed)
		require.Equal(t, freePlan, res.Plan, "Deny でも Plan を返す（メッセージ生成用）")
	})

	t.Run("PlanRepository がエラーなら error を返す", func(t *testing.T) {
		plans := &fakePlanRepository{err: errors.New("db down")}
		limiter := ratelimit.New()
		svc := usecase.NewQuotaService(plans, limiter)

		res, err := svc.Allow(context.Background(), "admin-A", time.Now())

		require.Error(t, err)
		require.Nil(t, res)
		require.Contains(t, err.Error(), "db down")
	})

	t.Run("プランごとに別の制限値が適用される", func(t *testing.T) {
		bigPlan := domain.Plan{
			Code: "premium", DisplayName: "上級",
			MaxLineUsers: 10, HourlyLimit: 50, DailyLimit: 1000,
		}
		plans := &fakePlanRepository{plan: &bigPlan}
		limiter := ratelimit.New()
		svc := usecase.NewQuotaService(plans, limiter)

		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		ctx := context.Background()

		// 無料プランなら 6 回目で落ちるところを、premium なら通る
		for i := 0; i < 10; i++ {
			res, err := svc.Allow(ctx, "admin-rich", base.Add(time.Duration(i)*time.Minute))
			require.NoError(t, err)
			require.True(t, res.Allowed, "i=%d", i)
			require.Equal(t, "premium", res.Plan.Code)
		}
	})

	t.Run("別 adminID は独立してカウント", func(t *testing.T) {
		plans := &fakePlanRepository{plan: &freePlan}
		limiter := ratelimit.New()
		svc := usecase.NewQuotaService(plans, limiter)

		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		ctx := context.Background()

		// admin-A を hourly 上限まで使い切る
		for i := 0; i < freePlan.HourlyLimit; i++ {
			res, _ := svc.Allow(ctx, "admin-A", base.Add(time.Duration(i)*time.Minute))
			require.True(t, res.Allowed)
		}
		resA, _ := svc.Allow(ctx, "admin-A", base.Add(10*time.Minute))
		require.False(t, resA.Allowed)

		// admin-B は別カウント
		resB, _ := svc.Allow(ctx, "admin-B", base.Add(10*time.Minute))
		require.True(t, resB.Allowed)
	})
}
