package domain

import "time"

// LineUser は LINE Platform 上のユーザー（おばあちゃん）と管理者（孫）を紐付ける登録レコード。
// line_users テーブルの 1 行に対応する。
type LineUser struct {
	ID             string
	AdminID        string
	LineUserID     string
	DisplayName    *string
	SessionResetAt *time.Time
	CreatedAt      time.Time
}
