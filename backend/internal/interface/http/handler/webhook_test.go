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

type fakeMessageResponder struct {
	mu    sync.Mutex
	calls []respondCall
	err   error
}

type respondCall struct {
	lineUserID string
	replyToken string
	text       string
}

func (f *fakeMessageResponder) Execute(_ context.Context, lineUserID, replyToken, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, respondCall{lineUserID, replyToken, text})
	return f.err
}

func (f *fakeMessageResponder) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// 署名計算 HMAC-SHA256 で body をハッシュ化（鍵は secret）した後に base64 にエンコードしている
func computeSignature(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWebhook(t *testing.T) {
	secret := "test-secret"
	textMessageBody := `{"events":[{"type":"message","timestamp":1,"source":{"type":"user","userId":"U1234567890abcdef"},"replyToken":"abc","message":{"type":"text","id":"1","text":"hello"}}]}`

	tests := []struct {
		name             string
		body             string
		signature        string
		respondErr       error
		wantStatus       int
		wantRespondCalls int
	}{
		{
			name:             "正常系",
			body:             textMessageBody,
			signature:        computeSignature(textMessageBody, secret),
			wantStatus:       200,
			wantRespondCalls: 1,
		},
		{
			name:             "signature が空",
			body:             "{}",
			signature:        "",
			wantStatus:       401,
			wantRespondCalls: 0,
		},
		{
			name:             "signature の不一致",
			body:             "{}",
			signature:        "invalid",
			wantStatus:       401,
			wantRespondCalls: 0,
		},
		{
			name:             "events が空",
			body:             `{"events":[]}`,
			signature:        computeSignature(`{"events":[]}`, secret),
			wantStatus:       200,
			wantRespondCalls: 0,
		},
		{
			name:             "respond.Execute が失敗",
			body:             textMessageBody,
			signature:        computeSignature(textMessageBody, secret),
			respondErr:       errors.New("respond down"),
			wantStatus:       200,
			wantRespondCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respond := &fakeMessageResponder{err: tt.respondErr}

			// echoインスタンスの作成
			e := echo.New()
			e.POST("/webhook", handler.Webhook(secret, respond))

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
			require.Equal(t, tt.wantRespondCalls, respond.CallCount())
		})
	}
}
