package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
)

// メッセージ定数（reply で返す文言）
const (
	msgRegistered            = "登録しました。これからお手伝いさせてくださいね。"
	msgRegistrationRequired  = "まだ登録が済んでいません。お孫さんからもらった 6 桁のコードを送ってください。"
	msgTokenNotFound         = "コードが違うみたいです。もう一度確認して送ってください。"          //nolint:gosec // user-facing message, not a credential
	msgTokenExpired          = "このコードは期限が切れています。お孫さんに新しいコードをもらってください。" //nolint:gosec // user-facing message, not a credential
	msgRevoked               = "登録が解除されています。再度ご利用される場合は、お孫さんに新しいコードをもらってください。"
	msgInternalError         = "ごめんなさい、ちょっと調子が悪いみたいです。少し待ってからもう一度送ってください。"
	msgClaudeError           = "うまくお答えできませんでした。少し時間を置いてからもう一度送ってみてください。"
	msgSessionReset          = "新しい質問をどうぞ。前のお話はいったんおしまいにしますね。"
	msgUnresolved            = "ごめんなさい、お役に立てませんでした。お孫さんに「まごAI でうまく解決しなかった」とお伝えください。直接LINEやお電話などでサポートしてもらえると思います。"
	msgRespondingPlaceholder = "メッセージありがとうございます。お返事の機能はまだ準備中です。"
	msgPlanCapacityFull      = "ごめんなさい、登録できませんでした。お孫さんに「まごAI の登録枠がいっぱいです」とお伝えください。"
)

// SessionResetCommand は会話履歴リセット用の特殊コマンド（Rich Menu から送信される）。
const SessionResetCommand = "#新しい質問"

// UnresolvedCommand は「解決しなかった」フィードバック用の特殊コマンド（Rich Menu から送信される）。
// 会話履歴に残り、管理画面で孫が確認できる。
const UnresolvedCommand = "#解決しなかった"

// 会話履歴の取得上限（直近 N ターン）
const conversationHistoryLimit = 20

// RespondToIncomingMessage はおばあちゃんから受信したメッセージに応じて適切な返信を返す usecase。
//
//   - 現役ユーザー + #新しい質問 → session_reset_at 更新 + 案内
//   - 現役ユーザー + 通常メッセージ → Claude 応答 + 会話履歴保存
//   - 6 桁数字（未登録 / 取り消し済み）→ 登録試行（新規 or 復活）
//   - 取り消し済み + 通常メッセージ → 解除メッセージ
//   - 未登録 + 通常メッセージ → 登録案内
type RespondToIncomingMessage struct {
	lineUsers          LineUserRepository
	conversations      ConversationRepository
	adminLinks         AdminLineLinkRepository
	line               LineGateway
	claude             ClaudeGateway
	notifier           AdminNotifier
	quota              QuotaService
	register           *RegisterLineUserByToken
	conversationWindow time.Duration
}

// NewRespondToIncomingMessage は依存を注入して RespondToIncomingMessage を生成する。
func NewRespondToIncomingMessage(
	lineUsers LineUserRepository,
	conversations ConversationRepository,
	adminLinks AdminLineLinkRepository,
	line LineGateway,
	claude ClaudeGateway,
	notifier AdminNotifier,
	quota QuotaService,
	register *RegisterLineUserByToken,
	conversationWindow time.Duration,
) *RespondToIncomingMessage {
	return &RespondToIncomingMessage{
		lineUsers:          lineUsers,
		conversations:      conversations,
		adminLinks:         adminLinks,
		line:               line,
		claude:             claude,
		notifier:           notifier,
		quota:              quota,
		register:           register,
		conversationWindow: conversationWindow,
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
	user, err := uc.lineUsers.FindActiveByLineUserID(ctx, lineUserID)
	if err != nil {
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
	if user != nil {
		return uc.respondToActive(ctx, user, replyToken, text)
	}

	// 2. 未登録 / 取り消し済み + 6 桁数字なら登録試行
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

// respondToActive は現役ユーザーに対する応答（特殊コマンドまたは Claude 応答）。
func (uc *RespondToIncomingMessage) respondToActive(
	ctx context.Context,
	user *domain.LineUser,
	replyToken, text string,
) error {
	// #新しい質問 → session_reset_at 更新
	if text == SessionResetCommand {
		if err := uc.lineUsers.UpdateSessionResetAt(ctx, user.LineUserID); err != nil {
			_ = uc.line.Reply(ctx, replyToken, msgInternalError)
			return err
		}
		return uc.line.Reply(ctx, replyToken, msgSessionReset)
	}

	// #解決しなかった → フィードバックとして会話履歴に残し、案内を返す
	if text == UnresolvedCommand {
		return uc.handleUnresolved(ctx, user, replyToken, text)
	}

	// Claude 応答
	return uc.respondWithClaude(ctx, user, replyToken, text)
}

// handleUnresolved は「解決しなかった」フィードバックを記録し、案内を返す。
// 会話履歴に user / assistant 双方の行を残し、紐付けされた孫の LINE に通知を送る。
func (uc *RespondToIncomingMessage) handleUnresolved(
	ctx context.Context,
	user *domain.LineUser,
	replyToken, text string,
) error {
	// user 行として保存（孫が会話ログで確認できるように）
	if err := uc.conversations.CreateUser(ctx, user.ID, text); err != nil {
		slog.Error("save unresolved user message failed", "err", err)
		// 保存失敗してもユーザーには返信を返す
	}

	// assistant 行として案内を保存（メタ情報なし）
	if err := uc.conversations.CreateAssistant(ctx, user.ID, msgUnresolved, domain.AssistantMeta{}); err != nil {
		slog.Error("save unresolved assistant message failed", "err", err)
	}

	// 孫の LINE に通知（ベストエフォート、失敗してもおばあちゃんには返信する）
	uc.notifyAdminUnresolved(ctx, user)

	return uc.line.Reply(ctx, replyToken, msgUnresolved)
}

// notifyAdminUnresolved は admin に紐付いた全 LINE 連携先に通知を送る。
// エラーはログするだけ。おばあちゃん側の応答はブロックしない。
func (uc *RespondToIncomingMessage) notifyAdminUnresolved(
	ctx context.Context,
	user *domain.LineUser,
) {
	links, err := uc.adminLinks.FindByAdminID(ctx, user.AdminID)
	if err != nil {
		slog.Error("find admin links failed", "err", err)
		return
	}
	if len(links) == 0 {
		return // 連携なし
	}

	name := "おばあちゃん"
	if user.DisplayName != nil && *user.DisplayName != "" {
		name = *user.DisplayName
	}

	msg := name + "さんが「うまく解決しなかった」と回答しました。\n管理画面の会話ログで詳細をご確認ください。"

	for _, link := range links {
		if err := uc.notifier.Push(ctx, link.LineUserID, msg); err != nil {
			slog.Error("notify admin failed",
				"err", err,
				"linkID", link.ID,
				"adminID", link.AdminID,
			)
		}
	}
}

// respondWithClaude は会話履歴を組み立てて Claude に問い合わせ、結果を返信＆保存する。
func (uc *RespondToIncomingMessage) respondWithClaude(
	ctx context.Context,
	user *domain.LineUser,
	replyToken, text string,
) error {
	// レート制限チェック（admin 単位、プラン依存）。
	// QuotaService エラー時は fail open（ログのみ）：
	// インフラ問題でユーザーをブロックしない。Claude 自体に API レートリミットがあるので
	// 仮に流量が漏れても上限暴走には至らない。
	now := time.Now()
	quotaResult, qerr := uc.quota.Allow(ctx, user.AdminID, now)
	if qerr != nil {
		slog.Warn("quota check failed (fail open)", "err", qerr, "adminID", user.AdminID)
	} else if !quotaResult.Allowed {
		slog.Info("quota exceeded",
			"adminID", user.AdminID,
			"plan", quotaResult.Plan.Code,
			"hourly", quotaResult.Plan.HourlyLimit,
			"daily", quotaResult.Plan.DailyLimit,
		)
		return uc.line.Reply(ctx, replyToken, quotaExceededMessage(quotaResult.Plan))
	}

	// 会話ウィンドウの起点：max(now - window, session_reset_at)
	cutoff := now.Add(-uc.conversationWindow)
	if user.SessionResetAt != nil && user.SessionResetAt.After(cutoff) {
		cutoff = *user.SessionResetAt
	}

	// 過去ターン取得（新しい順）→ 時系列順に並べ替え
	history, err := uc.conversations.Recent(ctx, user.ID, cutoff, conversationHistoryLimit)
	if err != nil {
		slog.Error("recent conversations failed", "err", err)
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
	reverseConversations(history)

	// user 行を先に保存（Claude 失敗時もユーザーの発言は残す）
	if err := uc.conversations.CreateUser(ctx, user.ID, text); err != nil {
		slog.Error("save user message failed", "err", err)
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}

	// Claude リクエスト構築
	messages := make([]ClaudeMessage, 0, len(history))
	for _, c := range history {
		messages = append(messages, ClaudeMessage{Role: c.Role, Content: c.Content})
	}

	resp, err := uc.claude.Complete(ctx, ClaudeRequest{
		SystemPrompt: SystemPrompt,
		History:      messages,
		UserMessage:  text,
	})
	if err != nil {
		slog.Error("claude complete failed", "err", err)
		_ = uc.line.Reply(ctx, replyToken, msgClaudeError)
		return err
	}

	// assistant 行を保存（失敗してもユーザーには応答するため、エラーはログのみ）
	if err := uc.conversations.CreateAssistant(ctx, user.ID, resp.Content, domain.AssistantMeta{
		LatencyMs:                resp.LatencyMs,
		Model:                    resp.Model,
		InputTokens:              resp.InputTokens,
		OutputTokens:             resp.OutputTokens,
		CacheReadInputTokens:     resp.CacheReadInputTokens,
		CacheCreationInputTokens: resp.CacheCreationInputTokens,
	}); err != nil {
		slog.Error("save assistant message failed", "err", err)
		// 続行：ユーザーには Claude の応答を返す
	}

	return uc.line.Reply(ctx, replyToken, resp.Content)
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
	case errors.Is(err, ErrPlanMaxLineUsersReached):
		return uc.line.Reply(ctx, replyToken, msgPlanCapacityFull)
	case errors.Is(err, ErrLineUserExists):
		// 安全網（FindActive で先に弾いてるので通常起きない）
		return uc.line.Reply(ctx, replyToken, msgRespondingPlaceholder)
	default:
		_ = uc.line.Reply(ctx, replyToken, msgInternalError)
		return err
	}
}

// quotaExceededMessage はプランの制限値を含めた拒否メッセージを生成する。
//
// 例: "（無料プランは 1 時間 5 回 / 1 日 30 回までです）"
func quotaExceededMessage(plan domain.Plan) string {
	return fmt.Sprintf(
		"たくさん使ってくれてありがとうございます。少し時間をおいてから、またお話しましょう。\n（%sプランは 1 時間 %d 回 / 1 日 %d 回までです）",
		plan.DisplayName, plan.HourlyLimit, plan.DailyLimit,
	)
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

// reverseConversations は新しい順の slice を時系列（古い順）に並べ替える。
func reverseConversations(s []domain.Conversation) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
