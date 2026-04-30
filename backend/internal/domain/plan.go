package domain

// Plan はサブスクリプションプランを表す。
//
// マスター値は plans テーブル（migration で管理）から読み込む。
// admin → plan の紐付けは admin_plans テーブルに保存され、
// 行が無い admin は無料プラン (DefaultPlanCode) として扱われる。
type Plan struct {
	Code         string // 識別子（"free" / "basic" / "premium"）
	DisplayName  string // 表示名（"無料" / "基本" / "上級"）
	MaxLineUsers int    // 同時に登録できる line_users 数（active のみカウント）
	HourlyLimit  int    // admin 全体で共有する 1 時間あたりの Claude 呼び出し回数
	DailyLimit   int    // admin 全体で共有する 24 時間あたりの Claude 呼び出し回数
}

// DefaultPlanCode は admin_plans に行が無い admin に適用されるプランコード。
const DefaultPlanCode = "free"
