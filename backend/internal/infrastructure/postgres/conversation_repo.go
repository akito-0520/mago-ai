package postgres

import (
	"context"
	"time"

	"github.com/akito-0520/mago-ai/backend/internal/domain"
	"github.com/akito-0520/mago-ai/backend/internal/usecase"

	"github.com/jmoiron/sqlx"
)

// ConversationRepository は usecase.ConversationRepository の Postgres 実装。
type ConversationRepository struct {
	db *sqlx.DB
}

var _ usecase.ConversationRepository = (*ConversationRepository)(nil)

// NewConversationRepository は ConversationRepository を生成する。
func NewConversationRepository(db *sqlx.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Recent は cutoff より新しい会話を最大 limit 件、新しい順で取得する。
func (r *ConversationRepository) Recent(
	ctx context.Context,
	lineUserID string,
	since time.Time,
	limit int,
) ([]domain.Conversation, error) {
	rows := []domain.Conversation{}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, line_user_id, role, content,
                latency_ms, model, input_tokens, output_tokens,
                cache_read_input_tokens, cache_creation_input_tokens, created_at
           FROM conversations
          WHERE line_user_id = $1
            AND created_at > $2
          ORDER BY created_at DESC
          LIMIT $3`,
		lineUserID, since, limit,
	)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateUser は user role の発言を保存する。
func (r *ConversationRepository) CreateUser(
	ctx context.Context,
	lineUserID string,
	content string,
) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conversations (line_user_id, role, content)
         VALUES ($1, 'user', $2)`,
		lineUserID, content,
	)
	return err
}

// CreateAssistant は assistant role の発言をメタ情報とともに保存する。
func (r *ConversationRepository) CreateAssistant(
	ctx context.Context,
	lineUserID string,
	content string,
	meta domain.AssistantMeta,
) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conversations (
            line_user_id, role, content,
            latency_ms, model,
            input_tokens, output_tokens,
            cache_read_input_tokens, cache_creation_input_tokens
        )
        VALUES (
            $1, 'assistant', $2,
            $3, $4,
            $5, $6,
            $7, $8
        )`,
		lineUserID, content,
		meta.LatencyMs, meta.Model,
		meta.InputTokens, meta.OutputTokens,
		meta.CacheReadInputTokens, meta.CacheCreationInputTokens,
	)
	return err
}
