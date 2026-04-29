// Package claude は usecase.ClaudeGateway の Anthropic SDK ベース実装を提供する。
package claude

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/akito-0520/mago-ai/backend/internal/usecase"
)

// 1 ターンあたりの最大出力トークン
const maxOutputTokens = 1024

// Client は usecase.ClaudeGateway の Anthropic SDK 実装。
type Client struct {
	api   anthropic.Client
	model string
}

var _ usecase.ClaudeGateway = (*Client)(nil)

// New は API キーとモデル名から Client を生成する。
func New(apiKey, model string) *Client {
	api := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &Client{
		api:   api,
		model: model,
	}
}

// Complete は会話履歴 + ユーザー発言を Claude に送って応答を取得する。
// システムプロンプトには cache_control: ephemeral を設定してプロンプトキャッシュを有効化する。
func (c *Client) Complete(ctx context.Context, req usecase.ClaudeRequest) (*usecase.ClaudeResponse, error) {
	// メッセージ履歴を SDK 形式に変換
	messages := make([]anthropic.MessageParam, 0, len(req.History)+1)
	for _, m := range req.History {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		default:
			// 未知の role はスキップ（防御的）
			continue
		}
	}
	messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserMessage)))

	// システムプロンプトに cache_control: ephemeral を付与
	system := []anthropic.TextBlockParam{
		{
			Text: req.SystemPrompt,
			CacheControl: anthropic.CacheControlEphemeralParam{
				Type: "ephemeral",
			},
		},
	}

	start := time.Now()
	resp, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: int64(maxOutputTokens),
		System:    system,
		Messages:  messages,
	})
	latency := time.Since(start)

	if err != nil {
		return nil, err
	}

	// 応答テキストを抽出
	content, err := extractText(resp)
	if err != nil {
		return nil, err
	}

	return &usecase.ClaudeResponse{
		Content:                  content,
		Model:                    resp.Model,
		LatencyMs:                int(latency.Milliseconds()),
		InputTokens:              int(resp.Usage.InputTokens),
		OutputTokens:             int(resp.Usage.OutputTokens),
		CacheReadInputTokens:     int(resp.Usage.CacheReadInputTokens),
		CacheCreationInputTokens: int(resp.Usage.CacheCreationInputTokens),
	}, nil
}

// extractText は Anthropic のレスポンスから text ブロックを連結して返す。
func extractText(resp *anthropic.Message) (string, error) {
	if resp == nil || len(resp.Content) == 0 {
		return "", errors.New("claude: empty response")
	}
	var out string
	for _, block := range resp.Content {
		if block.Type == "text" {
			out += block.Text
		}
	}
	if out == "" {
		return "", errors.New("claude: no text block in response")
	}
	return out, nil
}
