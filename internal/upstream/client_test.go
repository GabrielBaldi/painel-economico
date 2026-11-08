package upstream

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient returns a client with tiny delays so retries are fast.
func newTestClient(timeout time.Duration) *Client {
	return New(Options{Timeout: timeout, MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond})
}

func TestGetRetries(t *testing.T) {
	tests := []struct {
		name       string
		statuses   []int // response status per call; the last one repeats
		wantCalls  int32
		wantStatus int // 0 means success
	}{
		{name: "success on first try", statuses: []int{200}, wantCalls: 1},
		{name: "recovers after two 503s", statuses: []int{503, 503, 200}, wantCalls: 3},
		{name: "retries 429", statuses: []int{429, 200}, wantCalls: 2},
		{name: "gives up after max attempts", statuses: []int{500}, wantCalls: 3, wantStatus: 500},
		{name: "does not retry 404", statuses: []int{404}, wantCalls: 1, wantStatus: 404},
		{name: "does not retry 406", statuses: []int{406}, wantCalls: 1, wantStatus: 406},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(calls.Add(1))
				status := tt.statuses[min(n, len(tt.statuses))-1]
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer srv.Close()

			body, err := newTestClient(time.Second).Get(context.Background(), srv.URL)

			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("upstream called %d times, want %d", got, tt.wantCalls)
			}
			if tt.wantStatus == 0 {
				if err != nil || string(body) != `{"ok":true}` {
					t.Fatalf("Get() = (%q, %v), want the body and no error", body, err)
				}
				return
			}
			var statusErr *StatusError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != tt.wantStatus {
				t.Fatalf("Get() error = %v, want StatusError %d", err, tt.wantStatus)
			}
		})
	}
}

func TestGetRetriesTimeouts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			// First attempt hangs past the client timeout.
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
			return
		}
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	body, err := newTestClient(50*time.Millisecond).Get(context.Background(), srv.URL)
	if err != nil || string(body) != "[]" {
		t.Fatalf("Get() = (%q, %v), want success on the second attempt", body, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}

func TestGetStopsWhenContextIsCanceled(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(Options{MaxAttempts: 5, BaseDelay: time.Hour, MaxDelay: time.Hour})
	c.random = func() float64 { return 0.5 } // a 30-minute wait the context must cut short
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, srv.URL)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Get() error = %v, want context.DeadlineExceeded", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestBackoff(t *testing.T) {
	c := New(Options{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second})
	tests := []struct {
		attempt int
		random  float64
		want    time.Duration
	}{
		{attempt: 1, random: 0.5, want: 50 * time.Millisecond},
		{attempt: 2, random: 0.5, want: 100 * time.Millisecond},
		{attempt: 3, random: 0.5, want: 200 * time.Millisecond},
		{attempt: 4, random: 0.5, want: 400 * time.Millisecond},
		{attempt: 5, random: 0.5, want: 500 * time.Millisecond},  // capped at MaxDelay
		{attempt: 64, random: 0.5, want: 500 * time.Millisecond}, // no shift overflow
		{attempt: 3, random: 0, want: 0},
		{attempt: 3, random: 0.75, want: 300 * time.Millisecond},
	}
	for _, tt := range tests {
		c.random = func() float64 { return tt.random }
		if got := c.backoff(tt.attempt); got != tt.want {
			t.Errorf("backoff(%d) with random %.3f = %v, want %v", tt.attempt, tt.random, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{in: "curto", n: 10, want: "curto"},
		{in: "abcdef", n: 3, want: "abc…"},
		{in: "não", n: 2, want: "n…"}, // "ã" is two bytes; never cut it in half
	}
	for _, tt := range tests {
		if got := truncate(tt.in, tt.n); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}
