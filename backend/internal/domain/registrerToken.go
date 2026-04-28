package domain

import "time"

// RegisterToken は発行されたトークンの管理を行うレコード。
// 有効期限や使用済みラベル等でトークンの状態を管理する。
type RegisterToken struct {
	Token     string
	AdminID   string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    *string
	CreatedAt time.Time
}
