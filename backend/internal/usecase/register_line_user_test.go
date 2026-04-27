package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/stretchr/testify/require"
)

// --- fakeLineUserRepository ---

type fakeLineUserRepository struct {
	existsResult bool
	existsErr    error
	createCalls  []domain.LineUser
	createErr    error
}

func (f *fakeLineUserRepository) ExistsByLineUserID(ctx context.Context, lineUserID string) (bool, error) {
	return f.existsResult, f.existsErr
}

func (f *fakeLineUserRepository) Create(ctx context.Context, user domain.LineUser) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createCalls = append(f.createCalls, user)
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

func (f *fakeRegisterTokenRepository) FindUnusedByToken(ctx context.Context, token string) (*domain.RegisterToken, error) {
	return f.findResult, f.findErr
}

func (f *fakeRegisterTokenRepository) MarkUsed(ctx context.Context, token string, lineUserID string) error {
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
		UsedAt:    nil,
		UsedBy:    nil,
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	// 期限切れ状態のトークン
	expiredToken := &domain.RegisterToken{
		Token:     token,
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // 過去
		UsedAt:    nil,
		UsedBy:    nil,
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}

	tests := []struct {
		name              string
		existsResult      bool
		existsErr         error
		findResult        *domain.RegisterToken
		findErr           error
		markUsedErr       error
		createErr         error
		wantErr           error
		wantCreateCalls   int
		wantMarkUsedCalls int
	}{
		{
			name:              "正常系：未登録ユーザー + 有効トークン → 登録成功",
			existsResult:      false,
			findResult:        validToken,
			wantErr:           nil,
			wantCreateCalls:   1,
			wantMarkUsedCalls: 1,
		},
		{
			name:              "既に登録済みのユーザー",
			existsResult:      true,
			wantErr:           usecase.ErrLineUserExists,
			wantCreateCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "トークンが見つからない（不正なコード）",
			existsResult:      false,
			findResult:        nil, // 見つからない = nil
			wantErr:           usecase.ErrTokenNotFound,
			wantCreateCalls:   0,
			wantMarkUsedCalls: 0,
		},
		{
			name:              "トークンが期限切れ",
			existsResult:      false,
			findResult:        expiredToken,
			wantErr:           usecase.ErrTokenExpired,
			wantCreateCalls:   0,
			wantMarkUsedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				existsResult: tt.existsResult,
				existsErr:    tt.existsErr,
				createErr:    tt.createErr,
			}
			registerTokens := &fakeRegisterTokenRepository{
				findResult:  tt.findResult,
				findErr:     tt.findErr,
				markUsedErr: tt.markUsedErr,
			}

			uc := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens)
			err := uc.Execute(context.Background(), lineUserID, token)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, lineUsers.createCalls, tt.wantCreateCalls)
			require.Len(t, registerTokens.markUsedCalls, tt.wantMarkUsedCalls)
		})
	}
}
