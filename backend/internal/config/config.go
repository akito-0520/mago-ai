// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
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
}

const (
	defaultPort        = "8080"
	defaultClaudeModel = "claude-sonnet-4-6"
)

// Load reads environment variables and returns a populated Config.
// If any required variable is missing or empty, it returns an error
// that lists every missing key so all can be fixed at once.
func Load() (*Config, error) {
	required := []struct {
		key  string
		dest *string
	}{
		{"DATABASE_URL", nil},
		{"SUPABASE_URL", nil},
		{"SUPABASE_SERVICE_ROLE_KEY", nil},
		{"ANTHROPIC_API_KEY", nil},
		{"LINE_CHANNEL_SECRET", nil},
		{"LINE_CHANNEL_ACCESS_TOKEN", nil},
	}

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

	required[0].dest = &c.DatabaseURL
	required[1].dest = &c.SupabaseURL
	required[2].dest = &c.SupabaseServiceRoleKey
	required[3].dest = &c.AnthropicAPIKey
	required[4].dest = &c.LineChannelSecret
	required[5].dest = &c.LineChannelAccessToken

	var missing []string
	for _, r := range required {
		if *r.dest == "" {
			missing = append(missing, r.key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required env vars are missing: %s", strings.Join(missing, ", "))
	}

	return c, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
