// Package config reads the server settings from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Config holds every setting the server reads at startup.
type Config struct {
	Port            string        // PORT
	CacheTTL        time.Duration // CACHE_TTL
	UpstreamTimeout time.Duration // UPSTREAM_TIMEOUT, per attempt
	MaxAttempts     int           // UPSTREAM_MAX_ATTEMPTS
	LogLevel        slog.Level    // LOG_LEVEL: debug, info, warn or error
}

// Load builds a Config from getenv (os.Getenv outside of tests). Unset
// variables take sane defaults; malformed ones are an error.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:            "8103",
		CacheTTL:        time.Hour,
		UpstreamTimeout: 15 * time.Second,
		MaxAttempts:     3,
		LogLevel:        slog.LevelInfo,
	}
	if v := getenv("PORT"); v != "" {
		if _, err := strconv.ParseUint(v, 10, 16); err != nil {
			return Config{}, fmt.Errorf("PORT: %q is not a port number", v)
		}
		cfg.Port = v
	}
	if err := duration(getenv, "CACHE_TTL", &cfg.CacheTTL); err != nil {
		return Config{}, err
	}
	if err := duration(getenv, "UPSTREAM_TIMEOUT", &cfg.UpstreamTimeout); err != nil {
		return Config{}, err
	}
	if v := getenv("UPSTREAM_MAX_ATTEMPTS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 10 {
			return Config{}, fmt.Errorf("UPSTREAM_MAX_ATTEMPTS: %q is not a number between 1 and 10", v)
		}
		cfg.MaxAttempts = n
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
		}
	}
	return cfg, nil
}

// duration parses the variable key, if set, as a positive Go duration ("90s", "1h").
func duration(getenv func(string) string, key string, dst *time.Duration) error {
	v := getenv(key)
	if v == "" {
		return nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fmt.Errorf("%s: %q is not a positive duration such as 30s or 1h", key, v)
	}
	*dst = d
	return nil
}
