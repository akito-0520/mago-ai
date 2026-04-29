package domain

import "time"

// Conversation は LINE ユーザーと Bot の発言ログ 1 行に対応する。
// role が "assistant" のときだけメタデータ（latency / model / tokens）が埋まる。
type Conversation struct {
	ID                       int64
	LineUserID               string // line_users.id (UUID)
	Role                     string // "user" or "assistant"
	Content                  string
	LatencyMs                *int    // assistant のみ
	Model                    *string // assistant のみ
	InputTokens              *int    // assistant のみ
	OutputTokens             *int    // assistant のみ
	CacheReadInputTokens     *int    // assistant のみ
	CacheCreationInputTokens *int    // assistant のみ
	CreatedAt                time.Time
}

// AssistantMeta は assistant ロールの会話を保存するときのメタ情報。
type AssistantMeta struct {
	LatencyMs                int
	Model                    string
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
}
