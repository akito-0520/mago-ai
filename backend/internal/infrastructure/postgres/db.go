package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// New は DSN から *sqlx.DB を生成する。
// 接続プール設定、snake_case ↔ CamelCase の変換、
// および pgbouncer (Supabase pooler) 互換のための simple protocol モードを設定する。
func New(dsn string) (*sqlx.DB, error) {
	// pgx config をパース。Exec モード（prepared statement キャッシュなし）を有効にする。
	//
	// 理由：Supabase の Transaction pooler (port 6543) は pgbouncer ベースで、
	// 接続が複数クライアント間で使い回される。pgx のデフォルト (CacheStatement) は
	// prepared statement を named キャッシュするため、別クライアントが同名の statement を
	// 作ると衝突する（SQLSTATE 42P05）。
	//
	// QueryExecModeExec は extended protocol を維持しつつ、毎クエリ anonymous prepared
	// statement を使うため、pgbouncer と衝突せず、型変換・バイナリ転送など pgx の
	// 利点も保てる。simple protocol よりわずかに速い。
	pgxConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres parse config: %w", err)
	}
	pgxConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	// pgx config から *sql.DB を生成し、sqlx でラップする
	sqlDB := stdlib.OpenDB(*pgxConfig)

	// 疎通確認
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	db := sqlx.NewDb(sqlDB, "pgx")

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
