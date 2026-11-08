// Package upstream is an HTTP client for public APIs that fail now and then.
// Each attempt has a timeout, and failed attempts are retried with
// exponential backoff and full jitter.
package upstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"
	"unicode/utf8"
)

// maxBodyBytes caps how much of a response is read into memory.
const maxBodyBytes = 16 << 20

// Options configures a Client. Zero values fall back to the defaults below.
type Options struct {
	Timeout     time.Duration // per attempt; default 15s
	MaxAttempts int           // including the first one; default 3
	BaseDelay   time.Duration // backoff before the 2nd attempt; default 300ms
	MaxDelay    time.Duration // backoff cap; default 5s
	Logger      *slog.Logger  // where retries are reported; default: discarded
}

// Client fetches URLs with retries. It is safe for concurrent use.
type Client struct {
	http        *http.Client
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	random      func() float64 // in [0, 1); replaced in tests
	log         *slog.Logger
}

// New returns a Client configured by opts.
func New(opts Options) *Client {
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Client{
		http:        &http.Client{Timeout: or(opts.Timeout, 15*time.Second)},
		maxAttempts: or(opts.MaxAttempts, 3),
		baseDelay:   or(opts.BaseDelay, 300*time.Millisecond),
		maxDelay:    or(opts.MaxDelay, 5*time.Second),
		random:      rand.Float64,
		log:         log,
	}
}

// StatusError reports a response with a non-2xx status code.
type StatusError struct {
	StatusCode int
	URL        string
	Body       string // first bytes of the body, for logs
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: status %d: %s", e.URL, e.StatusCode, e.Body)
}

// Get fetches url and returns the response body. Network errors, timeouts,
// 429 and 5xx responses are retried; other errors are returned at once.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	for attempt := 1; ; attempt++ {
		body, err := c.get(ctx, url)
		switch {
		case err == nil:
			return body, nil
		case ctx.Err() != nil || !retryable(err):
			return nil, err
		case attempt == c.maxAttempts:
			return nil, fmt.Errorf("giving up after %d attempts: %w", attempt, err)
		}

		delay := c.backoff(attempt)
		c.log.Warn("upstream request failed, retrying",
			"url", url, "attempt", attempt, "retry_in", delay.String(), "error", err)
		if err := sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "painel-economico/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &StatusError{StatusCode: resp.StatusCode, URL: url, Body: truncate(string(body), 200)}
	}
	return body, nil
}

// backoff is the wait after the given failed attempt (>= 1): a random
// duration between 0 and base * 2^(attempt-1), capped at maxDelay. The
// randomness ("full jitter") keeps many clients from retrying in lockstep.
func (c *Client) backoff(attempt int) time.Duration {
	ceiling := c.maxDelay
	if attempt <= 30 { // beyond that the shift would overflow
		ceiling = min(c.baseDelay<<(attempt-1), c.maxDelay)
	}
	return time.Duration(c.random() * float64(ceiling))
}

func retryable(err error) bool {
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode >= 500 || statusErr.StatusCode == http.StatusTooManyRequests
	}
	return true // network errors and timeouts
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// or returns v, or fallback when v is not positive.
func or[T int | time.Duration](v, fallback T) T {
	if v > 0 {
		return v
	}
	return fallback
}

// truncate shortens s to at most n bytes without splitting a UTF-8 character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
