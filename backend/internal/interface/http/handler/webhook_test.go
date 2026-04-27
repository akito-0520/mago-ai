package handler_test

import (
	"context"         // Reply メソッドの第一引数
	"crypto/hmac"     // HMAC-SHA256 計算
	"crypto/sha256"   // SHA256 ハッシュ関数
	"encoding/base64" // 署名を Base64 エンコード
	"errors"
	"net/http"          // ステータスコード定数 (StatusOK 等)
	"net/http/httptest" // モック HTTP リクエスト/レスポンス
	"strings"           // strings.NewReader（body を io.Reader にする）
	"sync"              // データレース対策の Mutex
	"testing"           // *testing.T、Go 標準のテスト基盤
	"time"              // goroutine 完了待ちの time.Sleep

	"github.com/akito-0520/mago-ai/backend/internal/interface/http/handler" // テスト対象

	"github.com/labstack/echo/v4"         // echo.New(), Headers
	"github.com/stretchr/testify/require" // 検証ヘルパー (require.Equal 等)
)

type fakeLineGateway struct {
	mu         sync.Mutex
	replyCalls []replyCall
	replyErr   error
}

type replyCall struct {
	replyToken string
	text       string
}

func (f *fakeLineGateway) Reply(ctx context.Context, replyToken string, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replyCalls = append(f.replyCalls, replyCall{replyToken, text})
	return f.replyErr
}

// 署名計算 HMAC-SHA256 で body をハッシュ化（鍵は secret）した後に base64 にエンコードしている
func computeSignature(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (f *fakeLineGateway) ReplyCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.replyCalls)
}

func TestWebhook(t *testing.T) {
	secret := "test-secret"
	textMessageBody := `{"events":[{"type":"message","timestamp":1,"replyToken":"abc","message":{"type":"text","id":"1","text":"hello"}}]}`

	tests := []struct {
		name           string
		body           string
		signature      string
		replyErr       error
		wantStatus     int
		wantReplyCalls int
	}{
		{
			name:           "正常系",
			body:           textMessageBody,
			signature:      computeSignature(textMessageBody, secret),
			wantStatus:     200,
			wantReplyCalls: 1,
		},
		{
			name:           "signature が空",
			body:           "{}",
			signature:      "",
			wantStatus:     401,
			wantReplyCalls: 0,
		},
		{
			name:           "signature の不一致",
			body:           "{}",
			signature:      "invalid",
			wantStatus:     401,
			wantReplyCalls: 0,
		},
		{
			name:           "events が空",
			body:           `{"events":[]}`,
			signature:      computeSignature(`{"events":[]}`, secret),
			wantStatus:     200,
			wantReplyCalls: 0,
		},
		{
			name:           "Reply API が失敗",
			body:           textMessageBody,
			signature:      computeSignature(textMessageBody, secret),
			replyErr:       errors.New("LINE API down"),
			wantStatus:     200,
			wantReplyCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeLineGateway{replyErr: tt.replyErr}

			// echoインスタンスの作成
			e := echo.New()
			e.POST("/webhook", handler.Webhook(fake, secret))

			// リクエスト構築
			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON) // `Content-Type: application/json` ヘッダーを付与
			if tt.signature != "" {                                          // tt.signature が空の時は署名ヘッダーはつけない
				req.Header.Set("X-Line-Signature", tt.signature)
			}

			// `httptest.NewRecorder()` は偽の Response writer
			rec := httptest.NewRecorder()

			// リクエスト処理を実行
			// 結果は rec.Code（ステータス）, rec.Body（ボディ）, rec.Header()（ヘッダー）に入る
			e.ServeHTTP(rec, req)

			// ステータスコード検証
			require.Equal(t, tt.wantStatus, rec.Code)

			// goroutine の完了を待つ
			time.Sleep(50 * time.Millisecond)

			// reply の呼び出し回数検証
			require.Len(t, fake.replyCalls, fake.ReplyCallCount())
		})
	}
}
