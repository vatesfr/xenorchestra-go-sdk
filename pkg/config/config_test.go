package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientTimeoutEnvVar(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		unset   bool
		want    time.Duration
	}{
		{
			name:  "not set defaults to 30s",
			unset: true,
			want:  30 * time.Second,
		},
		{
			name:    "60s",
			timeout: "60s",
			want:    1 * time.Minute,
		},
		{
			name:    "5s",
			timeout: "5s",
			want:    5 * time.Second,
		},
		{
			name:    "2m",
			timeout: "2m",
			want:    2 * time.Minute,
		},
		{
			name:    "invalid value falls back to 30s",
			timeout: "not-a-duration",
			want:    30 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Required env vars for New()
			t.Setenv("XOA_URL", "http://example.com")
			t.Setenv("XOA_TOKEN", "abc")

			if tc.unset {
				os.Unsetenv("XOA_CLIENT_TIMEOUT")
			} else {
				t.Setenv("XOA_CLIENT_TIMEOUT", tc.timeout)
			}

			cfg, _ := New()
			assert.Equal(t, tc.want, cfg.ClientTimeout)
		})
	}
}
