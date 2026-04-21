// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration values loaded from the environment.
type Config struct {
	Port                   string
	DatabaseURL            string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	AnthropicAPIKey        string
	ClaudeModel            string
	LineChannelSecret      string
	LineChannelAccessToken string
	ConversationWindow     time.Duration
}

const (
	defaultPort                    = "8080"
	defaultClaudeModel             = "claude-sonnet-4-6"
	defaultConversationWindowHours = 24
)

// Load reads environment variables and returns a populated Config.
// If any required variable is missing or empty, or any optional
// variable has an invalid value, it returns an error listing every
// problem so all can be fixed at once.
func Load() (*Config, error) {
	c := &Config{
		Port:                   getEnvOrDefault("PORT", defaultPort),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SupabaseURL:            os.Getenv("SUPABASE_URL"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		AnthropicAPIKey:        os.Getenv("ANTHROPIC_API_KEY"),
		ClaudeModel:            getEnvOrDefault("CLAUDE_MODEL", defaultClaudeModel),
		LineChannelSecret:      os.Getenv("LINE_CHANNEL_SECRET"),
		LineChannelAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	}

	var problems []string

	required := []struct {
		key string
		val string
	}{
		{"DATABASE_URL", c.DatabaseURL},
		{"SUPABASE_URL", c.SupabaseURL},
		{"SUPABASE_SERVICE_ROLE_KEY", c.SupabaseServiceRoleKey},
		{"ANTHROPIC_API_KEY", c.AnthropicAPIKey},
		{"LINE_CHANNEL_SECRET", c.LineChannelSecret},
		{"LINE_CHANNEL_ACCESS_TOKEN", c.LineChannelAccessToken},
	}
	for _, r := range required {
		if r.val == "" {
			problems = append(problems, r.key)
		}
	}

	window, err := parseConversationWindow(os.Getenv("CONVERSATION_WINDOW_HOURS"))
	if err != nil {
		problems = append(problems, fmt.Sprintf("CONVERSATION_WINDOW_HOURS (%s)", err))
	} else {
		c.ConversationWindow = window
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid env vars: %s", strings.Join(problems, ", "))
	}

	return c, nil
}

// parseConversationWindow parses the CONVERSATION_WINDOW_HOURS env value.
// An empty string returns the default (24 hours).
// Non-positive or non-integer values return an error.
func parseConversationWindow(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultConversationWindowHours * time.Hour, nil
	}
	hours, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be a positive integer, got %q", raw)
	}
	if hours <= 0 {
		return 0, fmt.Errorf("must be a positive integer, got %d", hours)
	}
	return time.Duration(hours) * time.Hour, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
