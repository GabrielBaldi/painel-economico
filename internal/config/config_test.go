package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	defaults := Config{
		Port:            "8103",
		CacheTTL:        time.Hour,
		UpstreamTimeout: 15 * time.Second,
		MaxAttempts:     3,
		LogLevel:        slog.LevelInfo,
	}
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{name: "defaults", env: nil, want: defaults},
		{
			name: "all overridden",
			env: map[string]string{
				"PORT": "10000", "CACHE_TTL": "30m", "UPSTREAM_TIMEOUT": "5s",
				"UPSTREAM_MAX_ATTEMPTS": "5", "LOG_LEVEL": "debug",
			},
			want: Config{Port: "10000", CacheTTL: 30 * time.Minute, UpstreamTimeout: 5 * time.Second, MaxAttempts: 5, LogLevel: slog.LevelDebug},
		},
		{name: "port is not a number", env: map[string]string{"PORT": "http"}, wantErr: true},
		{name: "ttl without unit", env: map[string]string{"CACHE_TTL": "60"}, wantErr: true},
		{name: "negative timeout", env: map[string]string{"UPSTREAM_TIMEOUT": "-1s"}, wantErr: true},
		{name: "zero attempts", env: map[string]string{"UPSTREAM_MAX_ATTEMPTS": "0"}, wantErr: true},
		{name: "unknown log level", env: map[string]string{"LOG_LEVEL": "verbose"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(func(key string) string { return tt.env[key] })
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() = %+v, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
