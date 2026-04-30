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

// --- fakeAdminLineLinkRepository ---

type fakeAdminLineLinkRepository struct {
	findByLineResult  *domain.AdminLineLink
	findByLineErr     error
	findByAdminCalls  []string
	findByAdminResult []domain.AdminLineLink
	findByAdminErr    error
	createCalls       []domain.AdminLineLink
	createErr         error
	createID          string
}

func (f *fakeAdminLineLinkRepository) FindByLineUserID(_ context.Context, _ string) (*domain.AdminLineLink, error) {
	return f.findByLineResult, f.findByLineErr
}

func (f *fakeAdminLineLinkRepository) FindByAdminID(_ context.Context, adminID string) ([]domain.AdminLineLink, error) {
	f.findByAdminCalls = append(f.findByAdminCalls, adminID)
	return f.findByAdminResult, f.findByAdminErr
}

func (f *fakeAdminLineLinkRepository) Create(_ context.Context, link domain.AdminLineLink) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	f.createCalls = append(f.createCalls, link)
	id := f.createID
	if id == "" {
		id = "fake-link-id"
	}
	return id, nil
}

// --- fakeAdminNotifier ---

type fakeAdminNotifier struct {
	pushCalls     []notifyCall
	pushErr       error
	profileResult *usecase.LineProfile
	profileErr    error
}

type notifyCall struct {
	lineUserID string
	text       string
}

func (f *fakeAdminNotifier) Push(_ context.Context, lineUserID, text string) error {
	if f.pushErr != nil {
		return f.pushErr
	}
	f.pushCalls = append(f.pushCalls, notifyCall{lineUserID, text})
	return nil
}

func (f *fakeAdminNotifier) GetProfile(_ context.Context, lineUserID string) (*usecase.LineProfile, error) {
	if f.profileErr != nil {
		return nil, f.profileErr
	}
	if f.profileResult == nil {
		return &usecase.LineProfile{UserID: lineUserID, DisplayName: ""}, nil
	}
	return f.profileResult, nil
}

// --- fakeConversationRepository ---

type fakeConversationRepository struct {
	recentResult    []domain.Conversation
	recentErr       error
	createUserCalls []conversationCall
	createUserErr   error
	createAsstCalls []assistantCall
	createAsstErr   error
}

type conversationCall struct {
	lineUserID string
	content    string
}

type assistantCall struct {
	lineUserID string
	content    string
	meta       domain.AssistantMeta
}

func (f *fakeConversationRepository) Recent(_ context.Context, _ string, _ time.Time, _ int) ([]domain.Conversation, error) {
	if f.recentErr != nil {
		return nil, f.recentErr
	}
	return f.recentResult, nil
}

func (f *fakeConversationRepository) CreateUser(_ context.Context, lineUserID, content string) error {
	if f.createUserErr != nil {
		return f.createUserErr
	}
	f.createUserCalls = append(f.createUserCalls, conversationCall{lineUserID, content})
	return nil
}

func (f *fakeConversationRepository) CreateAssistant(_ context.Context, lineUserID, content string, meta domain.AssistantMeta) error {
	if f.createAsstErr != nil {
		return f.createAsstErr
	}
	f.createAsstCalls = append(f.createAsstCalls, assistantCall{lineUserID, content, meta})
	return nil
}

// --- fakeClaudeGateway ---

type fakeClaudeGateway struct {
	completeResult *usecase.ClaudeResponse
	completeErr    error
	completeCalls  []usecase.ClaudeRequest
}

func (f *fakeClaudeGateway) Complete(_ context.Context, req usecase.ClaudeRequest) (*usecase.ClaudeResponse, error) {
	f.completeCalls = append(f.completeCalls, req)
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	if f.completeResult == nil {
		return &usecase.ClaudeResponse{Content: "ok", Model: "claude-test"}, nil
	}
	return f.completeResult, nil
}

func TestRespondToIncomingMessage_Execute(t *testing.T) {
	const (
		lineUserID = "U1234567890abcdef"
		userUUID   = "11111111-1111-1111-1111-111111111111"
		replyToken = "test-reply-token-001"
		adminID    = "00000000-0000-0000-0000-000000000001"
	)

	now := time.Now()
	activeUser := &domain.LineUser{
		ID:         userUUID,
		AdminID:    adminID,
		LineUserID: lineUserID,
		CreatedAt:  now.Add(-1 * time.Hour),
	}

	validToken := &domain.RegisterToken{
		Token:     "123456",
		AdminID:   adminID,
		ExpiresAt: now.Add(1 * time.Hour),
	}

	expiredToken := &domain.RegisterToken{
		Token:     "123456",
		AdminID:   adminID,
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	tests := []struct {
		name           string
		activeUser     *domain.LineUser
		activeErr      error
		revokedResult  bool
		findToken      *domain.RegisterToken
		claudeResp     *usecase.ClaudeResponse
		claudeErr      error
		text           string
		wantReplyText  string
		wantClaudeCall int
		wantUserSave   int
		wantAsstSave   int
		wantErrSubstr  string
	}{
		{
			name:           "現役ユーザー + 通常メッセージ → Claude 応答",
			activeUser:     activeUser,
			text:           "iPhoneで写真を送りたい",
			claudeResp:     &usecase.ClaudeResponse{Content: "「写真」アプリを開いてください。", Model: "claude-sonnet-4-6"},
			wantReplyText:  "「写真」アプリを開いてください。",
			wantClaudeCall: 1,
			wantUserSave:   1,
			wantAsstSave:   1,
		},
		{
			name:          "現役ユーザー + #新しい質問 → セッションリセット",
			activeUser:    activeUser,
			text:          "#新しい質問",
			wantReplyText: "新しい質問をどうぞ。前のお話はいったんおしまいにしますね。",
		},
		{
			name:          "現役ユーザー + #解決しなかった → フィードバック保存",
			activeUser:    activeUser,
			text:          "#解決しなかった",
			wantReplyText: "ごめんなさい、お役に立てませんでした。お孫さんに「mago.ai でうまく解決しなかった」とお伝えください。直接お電話などでサポートしてもらえると思います。",
			wantUserSave:  1,
			wantAsstSave:  1,
		},
		{
			name:           "Claude 失敗 → エラーメッセージ + assistant 行は保存しない",
			activeUser:     activeUser,
			text:           "助けて",
			claudeErr:      errors.New("rate limit exceeded"),
			wantReplyText:  "うまくお答えできませんでした。少し時間を置いてからもう一度送ってみてください。",
			wantClaudeCall: 1,
			wantUserSave:   1,
			wantAsstSave:   0,
			wantErrSubstr:  "rate limit",
		},
		{
			name:          "未登録 + 6桁数字 + 有効トークン → 登録成功",
			activeUser:    nil,
			revokedResult: false,
			findToken:     validToken,
			text:          "123456",
			wantReplyText: "登録しました。これからお手伝いさせてくださいね。",
		},
		{
			name:          "未登録 + 6桁数字 + トークン未発行",
			activeUser:    nil,
			findToken:     nil,
			text:          "999999",
			wantReplyText: "コードが違うみたいです。もう一度確認して送ってください。",
		},
		{
			name:          "未登録 + 6桁数字 + 期限切れ",
			activeUser:    nil,
			findToken:     expiredToken,
			text:          "123456",
			wantReplyText: "このコードは期限が切れています。お孫さんに新しいコードをもらってください。",
		},
		{
			name:          "未登録 + 6桁数字でない → 登録案内",
			activeUser:    nil,
			revokedResult: false,
			text:          "こんにちは",
			wantReplyText: "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。",
		},
		{
			name:          "取り消し済み + 6桁数字でない → 解除メッセージ",
			activeUser:    nil,
			revokedResult: true,
			text:          "こんにちは",
			wantReplyText: "登録が解除されています。再度ご利用される場合は、お孫さんに新しいコードをもらってください。",
		},
		{
			name:          "取り消し済み + 6桁数字 + 有効トークン → 復活成功",
			activeUser:    nil,
			revokedResult: true,
			findToken:     validToken,
			text:          "123456",
			wantReplyText: "登録しました。これからお手伝いさせてくださいね。",
		},
		{
			name:          "FindActive でエラー",
			activeErr:     errors.New("db connection failed"),
			text:          "こんにちは",
			wantReplyText: "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。",
			wantErrSubstr: "db connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineUsers := &fakeLineUserRepository{
				findActiveResult: tt.activeUser,
				findActiveErr:    tt.activeErr,
				revokedResult:    tt.revokedResult,
			}
			registerTokens := &fakeRegisterTokenRepository{
				findResult: tt.findToken,
			}
			conversations := &fakeConversationRepository{}
			adminLinks := &fakeAdminLineLinkRepository{}
			line := &fakeLineGateway{}
			claudeGateway := &fakeClaudeGateway{
				completeResult: tt.claudeResp,
				completeErr:    tt.claudeErr,
			}
			notifier := &fakeAdminNotifier{}

			registerUC := usecase.NewRegisterLineUserByToken(lineUsers, registerTokens, line)
			uc := usecase.NewRespondToIncomingMessage(
				lineUsers,
				conversations,
				adminLinks,
				line,
				claudeGateway,
				notifier,
				registerUC,
				24*time.Hour,
			)

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

			if tt.wantClaudeCall > 0 {
				require.Len(t, claudeGateway.completeCalls, tt.wantClaudeCall)
			}
			require.Len(t, conversations.createUserCalls, tt.wantUserSave)
			require.Len(t, conversations.createAsstCalls, tt.wantAsstSave)
		})
	}
}
