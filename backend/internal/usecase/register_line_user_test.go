package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/stretchr/testify/require"
)

// --- fakeLineUserRepository ---

type fakeLineUserRepository struct {
	findActiveResult  *domain.LineUser
	findActiveErr     error
	revokedResult     bool
	revokedErr        error
	countActiveResult int
	countActiveErr    error
	countActiveCalls  []string
	upsertCalls       []domain.LineUser
	upsertErr         error
	updateResetCalls  []string
	updateResetErr    error
}

func (f *fakeLineUserRepository) FindActiveByLineUserID(_ context.Context, _ string) (*domain.LineUser, error) {
	return f.findActiveResult, f.findActiveErr
}

func (f *fakeLineUserRepository) ExistsRevokedByLineUserID(_ context.Context, _ string) (bool, error) {
	return f.revokedResult, f.revokedErr
}

func (f *fakeLineUserRepository) CountActiveByAdminID(_ context.Context, adminID string) (int, error) {
	f.countActiveCalls = append(f.countActiveCalls, adminID)
	if f.countActiveErr != nil {
		return 0, f.countActiveErr
	}
	return f.countActiveResult, nil
}

func (f *fakeLineUserRepository) Upsert(_ context.Context, user domain.LineUser) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upsertCalls = append(f.upsertCalls, user)
	return nil
}

func (f *fakeLineUserRepository) UpdateSessionResetAt(_ context.Context, lineUserID string) error {
	if f.updateResetErr != nil {
		return f.updateResetErr
	}
	f.updateResetCalls = append(f.updateResetCalls, lineUserID)
	return nil
}

// --- fakeRegisterTokenRepository ---

type fakeRegisterTokenRepository struct {
	findResult    *domain.RegisterToken
	findErr       error
	markUsedCalls []markUsedCall
	markUsedErr   error
}

type markUsedCall struct {
	token      string
	lineUserID string
}

func (f *fakeRegisterTokenRepository) FindUnusedByToken(_ context.Context, _ string) (*domain.RegisterToken, error) {
	return f.findResult, f.findErr
}

func (f *fakeRegisterTokenRepository) MarkUsed(_ context.Context, token string, lineUserID string) error {
	if f.markUsedErr != nil {
		return f.markUsedErr
	}
	f.markUsedCalls = append(f.markUsedCalls, markUsedCall{token, lineUserID})
	return nil
}

func TestRegisterLineUserByToken_Execute(t *testing.T) {
	const (
		lineUserID = "U1234567890abcdef"
		token      = "123456"
		adminID    = "00000000-0000-0000-0000-000000000001"
	)

	validToken := &domain.RegisterToken{
		Token:     token,
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	expiredToken := &domain.RegisterToken{
		Token:     token,
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}

	tests := []struct {
		name              string
		findResult        *domain.RegisterToken
		findErr           error
		upsertErr         error
		markUsedErr       error
		profileResult     *usecase.LineProfile
		profileErr        error
		planResult        *domain.Plan
		planErr           error
		countActiveResult int
		countActiveErr    error
		wantErr           error
		wantUpsertCalls   int
		wantMarkUsedCalls int
		wantDisplayName   *string
	}{
		{
			name:              "正常系：有効トークン + 枠空き + プロフィール取得成功 → 登録成功",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: "田中花子"},
			planResult:        &testFreePlan,
			countActiveResult: 0,
			wantUpsertCalls:   1,
			wantMarkUsedCalls: 1,
			wantDisplayName:   strPtr("田中花子"),
		},
		{
			name:              "プロフィール取得失敗 → 登録は成功（display_name は null）",
			findResult:        validToken,
			profileErr:        errors.New("get profile failed"),
			planResult:        &testFreePlan,
			countActiveResult: 0,
			wantUpsertCalls:   1,
			wantMarkUsedCalls: 1,
			wantDisplayName:   nil,
		},
		{
			name:              "プロフィールの DisplayName が空 → display_name は null",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: ""},
			planResult:        &testFreePlan,
			countActiveResult: 0,
			wantUpsertCalls:   1,
			wantMarkUsedCalls: 1,
			wantDisplayName:   nil,
		},
		{
			name:              "トークンが見つからない",
			findResult:        nil,
			wantErr:           usecase.ErrTokenNotFound,
			wantUpsertCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "トークンが期限切れ",
			findResult:        expiredToken,
			wantErr:           usecase.ErrTokenExpired,
			wantUpsertCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "プラン上限到達（現役 = max）→ ErrPlanMaxLineUsersReached",
			findResult:        validToken,
			planResult:        &testFreePlan, // MaxLineUsers: 1
			countActiveResult: 1,
			wantErr:           usecase.ErrPlanMaxLineUsersReached,
			wantUpsertCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "プラン上限超過（現役 > max）→ ErrPlanMaxLineUsersReached",
			findResult:        validToken,
			planResult:        &testFreePlan,
			countActiveResult: 5,
			wantErr:           usecase.ErrPlanMaxLineUsersReached,
			wantUpsertCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "Upsert で衝突 (現役ユーザー)",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: "田中花子"},
			planResult:        &testFreePlan,
			countActiveResult: 0,
			upsertErr:         usecase.ErrLineUserExists,
			wantErr:           usecase.ErrLineUserExists,
			wantUpsertCalls:   0,
			wantMarkUsedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				countActiveResult: tt.countActiveResult,
				countActiveErr:    tt.countActiveErr,
				upsertErr:         tt.upsertErr,
			}
			registerTokens := &fakeRegisterTokenRepository{
				findResult:  tt.findResult,
				findErr:     tt.findErr,
				markUsedErr: tt.markUsedErr,
			}
			line := &fakeLineGateway{
				profileResult: tt.profileResult,
				profileErr:    tt.profileErr,
			}
			plans := &fakePlanRepository{
				plan: tt.planResult,
				err:  tt.planErr,
			}

			uc := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens, plans, line)
			err := uc.Execute(context.Background(), lineUserID, token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, lineUsers.upsertCalls, tt.wantUpsertCalls)
			require.Len(t, registerTokens.markUsedCalls, tt.wantMarkUsedCalls)

			if tt.wantUpsertCalls > 0 {
				got := lineUsers.upsertCalls[0]
				require.Equal(t, lineUserID, got.LineUserID)
				require.Equal(t, adminID, got.AdminID)
				if tt.wantDisplayName == nil {
					require.Nil(t, got.DisplayName)
				} else {
					require.NotNil(t, got.DisplayName)
					require.Equal(t, *tt.wantDisplayName, *got.DisplayName)
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
