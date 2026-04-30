package usecase

import (
	"context"
	"errors"
	"log/slog"
)

// 通知 Bot からの返信メッセージ
const (
	msgAdminLinked            = "連携が完了しました。これからおばあちゃんからのフィードバックがこちらに届きます。"
	msgAdminAlreadyLinked     = "すでに連携済みです。"
	msgAdminLinkTokenNotFound = "コードが違うみたいです。管理画面で発行した 6 桁のコードを送ってください。" //nolint:gosec // user-facing message, not a credential
	msgAdminLinkTokenExpired  = "このコードは期限が切れています。管理画面で新しいコードを発行してください。"  //nolint:gosec // user-facing message, not a credential
	msgAdminLinkUsage         = "管理画面で発行した 6 桁の連携コードを送信してください。"
	msgAdminInternalError     = "ごめんなさい、ちょっと調子が悪いみたいです。少し時間を置いてからもう一度試してください。"
)

// RespondToAdminLineMessage は通知 Bot 経由で受信した管理者メッセージを処理する usecase。
// 主な役割は連携用の 6 桁トークン受信。それ以外のメッセージには使い方を案内する。
type RespondToAdminLineMessage struct {
	notifier AdminNotifier
	link     *LinkAdminLineByToken
}

// NewRespondToAdminLineMessage はコンストラクタ。
func NewRespondToAdminLineMessage(
	notifier AdminNotifier,
	link *LinkAdminLineByToken,
) *RespondToAdminLineMessage {
	return &RespondToAdminLineMessage{
		notifier: notifier,
		link:     link,
	}
}

// Execute は通知 Bot で受信したメッセージを処理する。
// reply token を持つ Message Event 用に Reply ではなく Push で返す（通知 Bot は一方通行運用のため）。
//
// ただし LINE の Reply API は Reply Token のみ対応するので、ここでは Push を使う。
// 本来 reply token を活用する場合は Reply に変えても良い。
func (uc *RespondToAdminLineMessage) Execute(
	ctx context.Context,
	lineUserID string,
	text string,
) error {
	// 6 桁数字なら連携試行
	if isSixDigits(text) {
		return uc.tryLink(ctx, lineUserID, text)
	}

	// それ以外は使い方を案内
	if err := uc.notifier.Push(ctx, lineUserID, msgAdminLinkUsage); err != nil {
		slog.Error("admin push failed", "err", err)
		return err
	}
	return nil
}

// tryLink はトークンで連携を試みる。
func (uc *RespondToAdminLineMessage) tryLink(ctx context.Context, lineUserID, token string) error {
	err := uc.link.Execute(ctx, lineUserID, token)
	switch {
	case err == nil:
		return uc.notifier.Push(ctx, lineUserID, msgAdminLinked)
	case errors.Is(err, ErrAdminLineLinkExists):
		return uc.notifier.Push(ctx, lineUserID, msgAdminAlreadyLinked)
	case errors.Is(err, ErrAdminLinkTokenNotFound):
		return uc.notifier.Push(ctx, lineUserID, msgAdminLinkTokenNotFound)
	case errors.Is(err, ErrAdminLinkTokenExpired):
		return uc.notifier.Push(ctx, lineUserID, msgAdminLinkTokenExpired)
	default:
		slog.Error("admin link failed", "err", err)
		_ = uc.notifier.Push(ctx, lineUserID, msgAdminInternalError)
		return err
	}
}
