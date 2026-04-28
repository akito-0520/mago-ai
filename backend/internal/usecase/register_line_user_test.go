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
	activeResult  bool // ExistsActive の結果
	activeErr     error
	revokedResult bool // ExistsRevoked の結果
	revokedErr    error
	upsertCalls   []domain.LineUser
	upsertErr     error
}

func (f *fakeLineUserRepository) ExistsActiveByLineUserID(_ context.Context, _ string) (bool, error) {
	return f.activeResult, f.activeErr
}

func (f *fakeLineUserRepository) ExistsRevokedByLineUserID(_ context.Context, _ string) (bool, error) {
	return f.revokedResult, f.revokedErr
}

func (f *fakeLineUserRepository) Upsert(_ context.Context, user domain.LineUser) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upsertCalls = append(f.upsertCalls, user)
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

	// 有効な状態のトークン
	validToken := &domain.RegisterToken{
		Token:     token,
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	// 期限切れ状態のトークン
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
		wantErr           error
		wantUpsertCalls   int
		wantMarkUsedCalls int
		wantDisplayName   *string // upsertCalls[0].DisplayName を検証する場合
	}{
		{
			name:              "正常系：有効トークン + プロフィール取得成功 → 登録成功",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: "田中花子"},
			wantUpsertCalls:   1,
			wantMarkUsedCalls: 1,
			wantDisplayName:   strPtr("田中花子"),
		},
		{
			name:              "プロフィール取得失敗 → 登録は成功（display_name は null）",
			findResult:        validToken,
			profileErr:        errors.New("get profile failed"),
			wantUpsertCalls:   1,
			wantMarkUsedCalls: 1,
			wantDisplayName:   nil,
		},
		{
			name:              "プロフィールの DisplayName が空 → display_name は null",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: ""},
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
			name:              "Upsert で衝突 (現役ユーザーが既に存在)",
			findResult:        validToken,
			profileResult:     &usecase.LineProfile{UserID: lineUserID, DisplayName: "田中花子"},
			upsertErr:         usecase.ErrLineUserExists,
			wantErr:           usecase.ErrLineUserExists,
			wantUpsertCalls:   0, // upsertErr が返るので append されない
			wantMarkUsedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				upsertErr: tt.upsertErr,
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

			uc := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens, line)
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
