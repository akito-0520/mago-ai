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

// --- fakeLineGateway ---

type fakeLineGateway struct {
	replies       []replyCall
	err           error
	profileResult *usecase.LineProfile
	profileErr    error
}

type replyCall struct {
	replyToken string
	text       string
}

func (f *fakeLineGateway) Reply(_ context.Context, replyToken, text string) error {
	f.replies = append(f.replies, replyCall{replyToken, text})
	return f.err
}

func (f *fakeLineGateway) GetProfile(_ context.Context, lineUserID string) (*usecase.LineProfile, error) {
	if f.profileErr != nil {
		return nil, f.profileErr
	}
	if f.profileResult == nil {
		return &usecase.LineProfile{UserID: lineUserID, DisplayName: ""}, nil
	}
	return f.profileResult, nil
}

func TestRespondToIncomingMessage_Execute(t *testing.T) {
	const (
		lineUserID = "U1234567890abcdef"
		replyToken = "abcdef0123456789" //nolint:gosec // gitleaks:allow -- test fixture
		adminID    = "00000000-0000-0000-0000-000000000001"
	)

	validToken := &domain.RegisterToken{
		Token:     "123456",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	expiredToken := &domain.RegisterToken{
		Token:     "123456",
		AdminID:   adminID,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	tests := []struct {
		name          string
		activeResult  bool
		activeErr     error
		revokedResult bool
		findResult    *domain.RegisterToken
		text          string
		wantReplyText string
		wantErrSubstr string
	}{
		{
			name:          "登録済みユーザー",
			activeResult:  true,
			text:          "こんにちは",
			wantReplyText: "メッセージありがとうございます。お返事の機能はまだ準備中です。",
		},
		{
			name:          "未登録 + 6桁数字 + 有効トークン → 登録成功",
			activeResult:  false,
			revokedResult: false,
			findResult:    validToken,
			text:          "123456",
			wantReplyText: "登録しました。これからお手伝いさせてくださいね。",
		},
		{
			name:          "未登録 + 6桁数字 + トークン未発行",
			activeResult:  false,
			revokedResult: false,
			findResult:    nil,
			text:          "999999",
			wantReplyText: "コードが違うみたいです。もう一度確認して送ってください。",
		},
		{
			name:          "未登録 + 6桁数字 + 期限切れ",
			activeResult:  false,
			revokedResult: false,
			findResult:    expiredToken,
			text:          "123456",
			wantReplyText: "このコードは期限が切れています。お孫さんに新しいコードをもらってください。",
		},
		{
			name:          "未登録 + 6桁数字でない",
			activeResult:  false,
			revokedResult: false,
			text:          "こんにちは",
			wantReplyText: "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。",
		},
		{
			name:          "取り消し済み + 6桁数字でない → 解除メッセージ",
			activeResult:  false,
			revokedResult: true,
			text:          "こんにちは",
			wantReplyText: "登録が解除されています。再度ご利用される場合は、お孫さんに新しいコードをもらってください。",
		},
		{
			name:          "取り消し済み + 6桁数字 + 有効トークン → 復活成功",
			activeResult:  false,
			revokedResult: true,
			findResult:    validToken,
			text:          "123456",
			wantReplyText: "登録しました。これからお手伝いさせてくださいね。",
		},
		{
			name:          "取り消し済み + 6桁数字 + 不正トークン → コードが違う",
			activeResult:  false,
			revokedResult: true,
			findResult:    nil,
			text:          "999999",
			wantReplyText: "コードが違うみたいです。もう一度確認して送ってください。",
		},
		{
			name:          "ExistsActive でエラー",
			activeErr:     errors.New("db connection failed"),
			text:          "こんにちは",
			wantReplyText: "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。",
			wantErrSubstr: "db connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				activeResult:  tt.activeResult,
				activeErr:     tt.activeErr,
				revokedResult: tt.revokedResult,
			}
			registerTokens := &fakeRegisterTokenRepository{
				findResult: tt.findResult,
			}
			line := &fakeLineGateway{}

			registerUC := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens, line)
			uc := usecase.NewRespondToIncomingMessage(lineUsers, line, registerUC)

			err := uc.Execute(context.Background(), lineUserID, replyToken, tt.text)

			if tt.wantErrSubstr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrSubstr)
			}

			require.Len(t, line.replies, 1)
			require.Equal(t, replyToken, line.replies[0].replyToken)
			require.Equal(t, tt.wantReplyText, line.replies[0].text)
		})
	}
}
