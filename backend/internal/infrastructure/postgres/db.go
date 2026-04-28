package postgres

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx ドライバ登録
)

// New は DSN から *sqlx.DB を生成する。
// 接続プール設定と snake_case ↔ CamelCase の変換を組み込む。
func New(dsn string) (*sqlx.DB, error) {
	// 接続オブジェクトの作成（プールマネージャー）
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// コネクションプール設定
	db.SetMaxOpenConns(20)                 // 最大同時接続数
	db.SetMaxIdleConns(5)                  // プールに保持する idle 接続の最大数
	db.SetConnMaxIdleTime(5 * time.Minute) // idle 接続の最大保持時間

	db.MapperFunc(toSnakeCase)

	return db, nil
}

// toSnakeCase は Go フィールド名（CamelCase）を DB カラム名（snake_case）に変換する。
// 例：LineUserID → line_user_id
func toSnakeCase(s string) string {
	runes := []rune(s)
	var b strings.Builder

	for i, r := range runes {
		if i > 0 && isUpper(r) {
			prevLower := isLower(runes[i-1])
			nextLower := i+1 < len(runes) && isLower(runes[i+1])
			if prevLower || nextLower {
				b.WriteByte('_')
			}
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
