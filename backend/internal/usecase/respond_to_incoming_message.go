package usecase

import (
	"context"
	"errors"
)

// メッセージ定数（reply で返す文言）
const (
	msgRegistered            = "登録しました。これからお手伝いさせてくださいね。"
	msgRegistrationRequired  = "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。"
	msgTokenNotFound         = "コードが違うみたいです。もう一度確認して送ってください。"          // #nosec G101 -- not a real secret
	msgTokenExpired          = "このコードは期限が切れています。お孫さんに新しいコードをもらってください。" // #nosec G101 -- not a real secret
	msgInternalError         = "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。"
	msgRespondingPlaceholder = "メッセージありがとうございます。お返事の機能はまだ準備中です。"
)

// RespondToIncomingMessage はおばあちゃんから受信したメッセージに応じて適切な返信を返す usecase。
//
//   - 未登録 + 6桁数字   → 登録試行
//   - 未登録 + それ以外  → 案内文
//   - 登録済み           → Claude 応答
type RespondToIncomingMessage struct {
	lineUsers LineUserRepository
	line      LineGateway
	register  *RegisterLineUserByToken
}

// NewRespondToIncomingMessage は依存を注入して RespondToIncomingMessage を生成する。
func NewRespondToIncomingMessage(
	lineUsers LineUserRepository,
	line LineGateway,
	register *RegisterLineUserByToken,
) *RespondToIncomingMessage {
	return &RespondToIncomingMessage{
		lineUsers: lineUsers,
		line:      line,
		register:  register,
	}
}

// Execute は受信メッセージを解釈し、Reply API で返信する。
func (uc *RespondToIncomingMessage) Execute(
	ctx context.Context,
	lineUserID string,
	replyToken string,
	text string,
) error {
	exists, err := uc.lineUsers.ExistsByLineUserID(ctx, lineUserID)
	if err != nil {
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}

	// 登録済みユーザー
	if exists {
		return uc.line.Reply(ctx, replyToken, msgRespondingPlaceholder)
	}

	// 未登録 + 6桁数字でない
	if !isSixDigits(text) {
		return uc.line.Reply(ctx, replyToken, msgRegistrationRequired)
	}

	// 未登録 + 6桁数字 → 登録試行
	return uc.tryRegister(ctx, lineUserID, replyToken, text)
}

// tryRegister は登録ユースケースを呼び、結果に応じた返信を行う。
func (uc *RespondToIncomingMessage) tryRegister(
	ctx context.Context,
	lineUserID, replyToken, token string,
) error {
	err := uc.register.Execute(ctx, lineUserID, token)
	switch {
	case err == nil:
		return uc.line.Reply(ctx, replyToken, msgRegistered)
	case errors.Is(err, ErrTokenNotFound):
		return uc.line.Reply(ctx, replyToken, msgTokenNotFound)
	case errors.Is(err, ErrTokenExpired):
		return uc.line.Reply(ctx, replyToken, msgTokenExpired)
	case errors.Is(err, ErrLineUserExists):
		// 安全網
		return uc.line.Reply(ctx, replyToken, msgRespondingPlaceholder)
	default:
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
}

// isSixDigits は 6 桁の半角数字かを判定する。
func isSixDigits(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
