package ratelimit_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/ratelimit"

	"github.com/stretchr/testify/require"
)

// 標準の Limits（テストで再利用する）
var stdLimits = ratelimit.Limits{Hourly: 5, Daily: 30}

func TestLimiter_Allow(t *testing.T) {
	t.Run("初回呼び出しは Allow", func(t *testing.T) {
		l := ratelimit.New()
		now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		require.True(t, l.Allow("admin-A", stdLimits, now))
	})

	t.Run("hourly limit ちょうどは全て Allow", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		for i := 0; i < stdLimits.Hourly; i++ {
			now := base.Add(time.Duration(i) * time.Minute)
			require.True(t, l.Allow("admin-A", stdLimits, now), "i=%d", i)
		}
	})

	t.Run("hourly limit + 1 回目は Deny", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		for i := 0; i < stdLimits.Hourly; i++ {
			require.True(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(i)*time.Minute)))
		}
		require.False(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(stdLimits.Hourly)*time.Minute)))
	})

	t.Run("hourly 制限後、1h 経過で復活", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		for i := 0; i < stdLimits.Hourly; i++ {
			require.True(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(i)*time.Minute)))
		}
		// すぐは Deny
		require.False(t, l.Allow("admin-A", stdLimits, base.Add(30*time.Minute)))
		// 1 番古い (12:00) が hourAgo より古くなる時刻 = 13:00 + 1ns
		recovered := base.Add(time.Hour).Add(time.Nanosecond)
		require.True(t, l.Allow("admin-A", stdLimits, recovered))
	})

	t.Run("daily limit + 1 回目は Deny", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
		// 24h 以内に daily limit ちょうど叩き込む。
		// 30 件を 40 分間隔で打つ → 全体スパン 29 * 40min = 19h20min（< 24h）
		// hourly 制限にも引っかからない（40min 間隔なので 1h 内には最大 2 件）
		for i := 0; i < stdLimits.Daily; i++ {
			now := base.Add(time.Duration(i) * 40 * time.Minute)
			require.True(t, l.Allow("admin-A", stdLimits, now), "i=%d", i)
		}
		// 最後の打刻から 1h 以上経過した時刻で +1 すると、hourly は通るが daily で落ちる
		over := base.Add(time.Duration(stdLimits.Daily-1)*40*time.Minute + 2*time.Hour)
		require.False(t, l.Allow("admin-A", stdLimits, over))
	})

	t.Run("daily 制限後、24h 経過で復活", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
		for i := 0; i < stdLimits.Daily; i++ {
			require.True(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(i)*40*time.Minute)))
		}
		// 全タイムスタンプが 24h より古くなる時刻
		lastTs := base.Add(time.Duration(stdLimits.Daily-1) * 40 * time.Minute)
		recovered := lastTs.Add(24 * time.Hour).Add(time.Nanosecond)
		require.True(t, l.Allow("admin-A", stdLimits, recovered))
	})

	t.Run("別 adminID は独立してカウント", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		// admin-A を hourly 上限まで使い切る
		for i := 0; i < stdLimits.Hourly; i++ {
			require.True(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(i)*time.Minute)))
		}
		require.False(t, l.Allow("admin-A", stdLimits, base.Add(10*time.Minute)))
		// admin-B は別カウント
		require.True(t, l.Allow("admin-B", stdLimits, base.Add(10*time.Minute)))
	})

	t.Run("Limits 違い（プラン違い）は要求側で渡したものに従う", func(t *testing.T) {
		l := ratelimit.New()
		base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		bigLimits := ratelimit.Limits{Hourly: 20, Daily: 200}
		// 6 回叩くケース：std だと Deny だが big だと Allow
		for i := 0; i < 10; i++ {
			require.True(t, l.Allow("admin-rich", bigLimits, base.Add(time.Duration(i)*time.Minute)), "i=%d", i)
		}
	})

	t.Run("並行アクセスでも race にならない", func(t *testing.T) {
		l := ratelimit.New()
		now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		const goroutines = 50
		done := make(chan struct{})
		for g := 0; g < goroutines; g++ {
			go func(id int) {
				adminID := "admin-" + strconv.Itoa(id)
				for i := 0; i < 10; i++ {
					_ = l.Allow(adminID, stdLimits, now.Add(time.Duration(i)*time.Minute))
				}
				done <- struct{}{}
			}(g)
		}
		for g := 0; g < goroutines; g++ {
			<-done
		}
	})
}

func TestLimiter_Allow_Deny時はカウント追加しない(t *testing.T) {
	l := ratelimit.New()
	base := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)

	// 5 回打って上限到達
	for i := 0; i < stdLimits.Hourly; i++ {
		require.True(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(i)*time.Minute)))
	}
	// 6 回目は Deny。何度試しても Deny のまま。
	for i := 0; i < 3; i++ {
		require.False(t, l.Allow("admin-A", stdLimits, base.Add(time.Duration(stdLimits.Hourly+i)*time.Minute)))
	}
	// 1h 後（12:01 が最古）→ 13:01:00.000...001 で復活
	// もし Deny 時にもタイムスタンプが追加されていたら、6 番目以降の打刻が干渉して
	// 復活時刻がずれる。ここで 13:01:00.000...001 で Allow になるかで検証。
	recovered := base.Add(time.Minute).Add(time.Hour).Add(time.Nanosecond)
	require.True(t, l.Allow("admin-A", stdLimits, recovered))
}
