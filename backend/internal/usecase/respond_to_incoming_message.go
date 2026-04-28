package usecase

import (
	"context"
	"errors"
)

// メッセージ定数（reply で返す文言）
const (
	msgRegistered            = "登録しました。これからお手伝いさせてくださいね。"
	msgRegistrationRequired  = "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。"
	msgTokenNotFound         = "コードが違うみたいです。もう一度確認して送ってください。"          //nolint:gosec // user-facing message, not a credential
	msgTokenExpired          = "このコードは期限が切れています。お孫さんに新しいコードをもらってください。" //nolint:gosec // user-facing message, not a credential
	msgRevoked               = "登録が解除されています。再度ご利用される場合は、お孫さんに新しいコードをもらってください。"
	msgInternalError         = "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。"
	msgRespondingPlaceholder = "メッセージありがとうございます。お返事の機能はまだ準備中です。"
)

// RespondToIncomingMessage はおばあちゃんから受信したメッセージに応じて適切な返信を返す usecase。
//
//   - 現役ユーザー         → Claude / placeholder
//   - 6 桁数字              → 登録試行（新規 or 取り消し済みの復活）
//   - 取り消し済み + 通常   → 解除メッセージ
//   - 未登録 + 通常         → 案内文
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
	// 1. 現役ユーザーかチェック
	activeExists, err := uc.lineUsers.ExistsActiveByLineUserID(ctx, lineUserID)
	if err != nil {
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
	if activeExists {
		return uc.line.Reply(ctx, replyToken, msgRespondingPlaceholder)
	}

	// 2. 6 桁数字なら登録試行（UPSERT が新規 or 復活を吸収）
	if isSixDigits(text) {
		return uc.tryRegister(ctx, lineUserID, replyToken, text)
	}

	// 3. 取り消し済みかチェックして案内を出し分け
	revokedExists, err := uc.lineUsers.ExistsRevokedByLineUserID(ctx, lineUserID)
	if err != nil {
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
	if revokedExists {
		return uc.line.Reply(ctx, replyToken, msgRevoked)
	}
	return uc.line.Reply(ctx, replyToken, msgRegistrationRequired)
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
		// 安全網（ExistsActive で先に弾いてるので通常起きない）
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
