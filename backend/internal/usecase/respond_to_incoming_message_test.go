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

// 既存の fakeLineUserRepository / fakeRegisterTokenRepository は再利用

// 新規：fakeLineGateway
type fakeLineGateway struct {
	replies []replyCall
	err     error
}

type replyCall struct {
	replyToken string
	text       string
}

func (f *fakeLineGateway) Reply(_ context.Context, replyToken, text string) error {
	f.replies = append(f.replies, replyCall{replyToken, text})
	return f.err
}

func TestRespondToIncomingMessage_Execute(t *testing.T) {
	const (
		lineUserID = "U1234567890abcdef"
		replyToken = "abcdef0123456789"
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
		existsResult  bool
		existsErr     error
		findResult    *domain.RegisterToken
		text          string
		wantReplyText string
		wantErrSubstr string // 空ならエラーなし期待
	}{
		{
			name:          "登録済みユーザー",
			existsResult:  true,
			text:          "こんにちは",
			wantReplyText: "メッセージありがとうございます。お返事の機能はまだ準備中です。",
		},
		{
			name:          "未登録 + 6桁数字 + 有効トークン → 登録成功",
			existsResult:  false,
			findResult:    validToken,
			text:          "123456",
			wantReplyText: "登録しました。これからお手伝いさせてくださいね。",
		},
		{
			name:          "未登録 + 6桁数字 + トークン未発行",
			existsResult:  false,
			findResult:    nil,
			text:          "999999",
			wantReplyText: "コードが違うみたいです。もう一度確認して送ってください。",
		},
		{
			name:          "未登録 + 6桁数字 + 期限切れ",
			existsResult:  false,
			findResult:    expiredToken,
			text:          "123456",
			wantReplyText: "このコードは期限が切れています。お孫さんに新しいコードをもらってください。",
		},
		{
			name:          "未登録 + 6桁数字でない",
			existsResult:  false,
			text:          "こんにちは",
			wantReplyText: "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。",
		},
		{
			name:          "Exists でエラー",
			existsErr:     errors.New("db connection failed"),
			text:          "こんにちは",
			wantReplyText: "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。",
			wantErrSubstr: "db connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				existsResult: tt.existsResult,
				existsErr:    tt.existsErr,
			}
			registerTokens := &fakeRegisterTokenRepository{
				findResult: tt.findResult,
			}
			line := &fakeLineGateway{}

			registerUC := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens)
			uc := usecase.NewRespondToIncomingMessage(lineUsers, line, registerUC)

			err := uc.Execute(context.Background(), lineUserID, replyToken, tt.text)

			// エラー検証
			if tt.wantErrSubstr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrSubstr)
			}

			// Reply 内容検証
			require.Len(t, line.replies, 1)
			require.Equal(t, replyToken, line.replies[0].replyToken)
			require.Equal(t, tt.wantReplyText, line.replies[0].text)
		})
	}
}
