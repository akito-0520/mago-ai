package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akito-0520/mago-ai/backend/internal/config"
)

func TestLoad(t *testing.T) {
	requiredEnvs := map[string]string{
		"DATABASE_URL":              "postgres://u:p@127.0.0.1:5432/db",
		"SUPABASE_URL":              "http://127.0.0.1:54321",
		"SUPABASE_SERVICE_ROLE_KEY": "service-role-key",
		"ANTHROPIC_API_KEY":         "sk-ant-xxx",
		"LINE_CHANNEL_SECRET":       "line-secret",
		"LINE_CHANNEL_ACCESS_TOKEN": "line-token",
	}

	tests := []struct {
		name     string
		envs     map[string]string
		wantErr  bool
		errMatch []string
		assert   func(t *testing.T, c *config.Config)
	}{
		{
			name: "全ての必須 env + 任意 env が揃えばその値が入る",
			envs: merge(requiredEnvs, map[string]string{
				"PORT":         "9090",
				"CLAUDE_MODEL": "claude-opus-4-7",
			}),
			assert: func(t *testing.T, c *config.Config) {
				require.Equal(t, "9090", c.Port)
				require.Equal(t, "postgres://u:p@127.0.0.1:5432/db", c.DatabaseURL)
				require.Equal(t, "http://127.0.0.1:54321", c.SupabaseURL)
				require.Equal(t, "service-role-key", c.SupabaseServiceRoleKey)
				require.Equal(t, "sk-ant-xxx", c.AnthropicAPIKey)
				require.Equal(t, "claude-opus-4-7", c.ClaudeModel)
				require.Equal(t, "line-secret", c.LineChannelSecret)
				require.Equal(t, "line-token", c.LineChannelAccessToken)
			},
		},
		{
			name: "PORT 未設定なら 8080 がデフォルト",
			envs: requiredEnvs,
			assert: func(t *testing.T, c *config.Config) {
				require.Equal(t, "8080", c.Port)
			},
		},
		{
			name: "CLAUDE_MODEL 未設定なら claude-sonnet-4-6 がデフォルト",
			envs: requiredEnvs,
			assert: func(t *testing.T, c *config.Config) {
				require.Equal(t, "claude-sonnet-4-6", c.ClaudeModel)
			},
		},
		{
			name:     "必須 env が 1 つ欠けるとエラー（DATABASE_URL）",
			envs:     except(requiredEnvs, "DATABASE_URL"),
			wantErr:  true,
			errMatch: []string{"DATABASE_URL"},
		},
		{
			name:     "必須 env が複数欠けるとエラーに全て含まれる",
			envs:     except(requiredEnvs, "DATABASE_URL", "ANTHROPIC_API_KEY", "LINE_CHANNEL_SECRET"),
			wantErr:  true,
			errMatch: []string{"DATABASE_URL", "ANTHROPIC_API_KEY", "LINE_CHANNEL_SECRET"},
		},
		{
			name:     "全部未設定なら全ての必須キーがエラーに含まれる",
			envs:     map[string]string{},
			wantErr:  true,
			errMatch: []string{"DATABASE_URL", "SUPABASE_URL", "SUPABASE_SERVICE_ROLE_KEY", "ANTHROPIC_API_KEY", "LINE_CHANNEL_SECRET", "LINE_CHANNEL_ACCESS_TOKEN"},
		},
		{
			name:     "必須 env が空文字のみの場合も未設定扱いでエラー",
			envs:     merge(requiredEnvs, map[string]string{"DATABASE_URL": ""}),
			wantErr:  true,
			errMatch: []string{"DATABASE_URL"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnvs(t)
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			c, err := config.Load()

			if tc.wantErr {
				require.Error(t, err)
				for _, want := range tc.errMatch {
					require.Contains(t, err.Error(), want, "error should mention %q", want)
				}
				require.Nil(t, c)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, c)
			if tc.assert != nil {
				tc.assert(t, c)
			}
		})
	}
}

// clearEnvs unsets all env vars Load() reads so tests start from a clean slate.
func clearEnvs(t *testing.T) {
	t.Helper()
	keys := []string{
		"PORT",
		"DATABASE_URL",
		"SUPABASE_URL",
		"SUPABASE_SERVICE_ROLE_KEY",
		"ANTHROPIC_API_KEY",
		"CLAUDE_MODEL",
		"LINE_CHANNEL_SECRET",
		"LINE_CHANNEL_ACCESS_TOKEN",
	}
	for _, k := range keys {
		t.Setenv(k, "")
		// t.Setenv sets the value to "" but keeps the var defined; that's fine
		// because Load() treats empty strings as unset for required keys.
		_ = strings.TrimSpace(k)
	}
}

func merge(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func except(base map[string]string, keys ...string) map[string]string {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}
