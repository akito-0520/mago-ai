package domain

import "time"

// AdminLineLink は管理者（孫）の LINE User ID と admin（auth.users）の紐付けレコード。
// 通知 Bot 経由で Push API を送る相手の情報を保持する。
type AdminLineLink struct {
	ID          string
	AdminID     string
	LineUserID  string  // 孫の LINE User ID（"U..." で始まる）
	DisplayName *string // LINE プロフィールから取得
	CreatedAt   time.Time
}

// AdminLinkToken は孫の LINE 連携用ワンタイムトークン。
// generate_admin_link_token() RPC で発行され、孫が通知 Bot に送信して連携完了。
type AdminLinkToken struct {
	Token     string // 6 桁の数字文字列
	AdminID   string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    *string // admin_line_links.id (UUID)
	CreatedAt time.Time
}
