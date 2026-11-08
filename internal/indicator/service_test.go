package indicator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"painel-economico/internal/cache"
)

// fakeSource returns canned points or an error and counts its calls.
// If release is set, Fetch blocks until it is closed.
type fakeSource struct {
	mu      sync.Mutex
	calls   int
	points  []Point
	err     error
	release chan struct{}
}

func (f *fakeSource) Fetch(ctx context.Context, from, to time.Time) ([]Point, error) {
	f.mu.Lock()
	f.calls++
	points, err, release := f.points, f.err, f.release
	f.mu.Unlock()
	if release != nil {
		<-release
	}
	return append([]Point(nil), points...), err
}

func (f *fakeSource) set(points []Point, err error) {
	f.mu.Lock()
	f.points, f.err = points, err
	f.mu.Unlock()
}

func (f *fakeSource) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

var (
	jan = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	dec = time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
)

// newTestService wires a service with one indicator ("selic") and a cache
// whose clock the test controls through the returned pointer.
func newTestService(src *fakeSource) (*Service, *time.Time) {
	now := time.Date(2026, time.November, 8, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	defs := []Definition{{Indicator: Indicator{ID: "selic", Name: "Selic"}, Upstream: src}}
	svc := NewService(defs, cache.NewMemory[Snapshot](time.Hour, clock), slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.now = clock
	return svc, &now
}

func dates(points []Point) []string {
	out := []string{}
	for _, p := range points {
		out = append(out, p.Date)
	}
	return out
}

func TestSeriesFiltersAndSortsPoints(t *testing.T) {
	src := &fakeSource{points: []Point{
		{Date: "2026-03-01", Value: 3},
		{Date: "2025-12-01", Value: 0},
		{Date: "2026-01-01", Value: 1},
		{Date: "2026-02-01", Value: 2},
	}}
	svc, _ := newTestService(src)

	from := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.February, 15, 0, 0, 0, 0, time.UTC)
	res, err := svc.Series(context.Background(), "selic", from, to)
	if err != nil {
		t.Fatalf("Series() error = %v", err)
	}
	if got, want := dates(res.Points), []string{"2026-01-01", "2026-02-01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("points = %v, want %v", got, want)
	}
	if res.Indicator.ID != "selic" || res.Stale {
		t.Errorf("got indicator %q stale=%v, want selic and not stale", res.Indicator.ID, res.Stale)
	}
}

func TestSeriesUnknownIndicator(t *testing.T) {
	svc, _ := newTestService(&fakeSource{})
	_, err := svc.Series(context.Background(), "bitcoin", jan, dec)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Series() error = %v, want ErrNotFound", err)
	}
}

func TestSeriesCaching(t *testing.T) {
	upstreamDown := errors.New("upstream down")
	old := []Point{{Date: "2026-01-01", Value: 1}}
	updated := []Point{{Date: "2026-01-01", Value: 1}, {Date: "2026-02-01", Value: 2}}

	tests := []struct {
		name string
		// The first call always succeeds with old; then the clock moves by
		// elapsed and the source starts returning (next, nextErr).
		elapsed    time.Duration
		next       []Point
		nextErr    error
		wantDates  []string
		wantStale  bool
		wantErr    error
		wantCalls  int
		skipWarmup bool
	}{
		{
			name: "fresh cache skips the upstream", elapsed: 30 * time.Minute,
			next: updated, wantDates: []string{"2026-01-01"}, wantCalls: 1,
		},
		{
			name: "expired cache refetches", elapsed: 2 * time.Hour,
			next: updated, wantDates: []string{"2026-01-01", "2026-02-01"}, wantCalls: 2,
		},
		{
			name: "upstream failure serves stale data", elapsed: 2 * time.Hour,
			nextErr: upstreamDown, wantDates: []string{"2026-01-01"}, wantStale: true, wantCalls: 2,
		},
		{
			name: "upstream failure with empty cache is an error", skipWarmup: true,
			nextErr: upstreamDown, wantErr: upstreamDown, wantCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &fakeSource{points: old}
			svc, now := newTestService(src)
			if !tt.skipWarmup {
				if _, err := svc.Series(context.Background(), "selic", jan, dec); err != nil {
					t.Fatalf("warm-up Series() error = %v", err)
				}
			}
			*now = now.Add(tt.elapsed)
			src.set(tt.next, tt.nextErr)

			res, err := svc.Series(context.Background(), "selic", jan, dec)

			if got := src.callCount(); got != tt.wantCalls {
				t.Errorf("source called %d times, want %d", got, tt.wantCalls)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Series() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Series() error = %v", err)
			}
			if got := dates(res.Points); !reflect.DeepEqual(got, tt.wantDates) {
				t.Errorf("points = %v, want %v", got, tt.wantDates)
			}
			if res.Stale != tt.wantStale {
				t.Errorf("stale = %v, want %v", res.Stale, tt.wantStale)
			}
		})
	}
}

func TestSeriesDeduplicatesConcurrentFetches(t *testing.T) {
	src := &fakeSource{points: []Point{{Date: "2026-01-01", Value: 1}}, release: make(chan struct{})}
	svc, _ := newTestService(src)

	const callers = 20
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Go(func() {
			_, err := svc.Series(context.Background(), "selic", jan, dec)
			errs <- err
		})
	}
	// Give every caller time to join the in-flight fetch, then let it finish.
	time.Sleep(50 * time.Millisecond)
	close(src.release)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Series() error = %v", err)
		}
	}
	if got := src.callCount(); got != 1 {
		t.Errorf("source called %d times for %d concurrent requests, want 1", got, callers)
	}
}

func TestSeriesCallerCancelDoesNotAbortSharedFetch(t *testing.T) {
	src := &fakeSource{points: []Point{{Date: "2026-01-01", Value: 1}}, release: make(chan struct{})}
	svc, _ := newTestService(src)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Series(ctx, "selic", jan, dec); !errors.Is(err, context.Canceled) {
		t.Fatalf("Series() with canceled context error = %v, want context.Canceled", err)
	}

	// The fetch started by the canceled caller still completes and is cached.
	close(src.release)
	res, err := svc.Series(context.Background(), "selic", jan, dec)
	if err != nil || len(res.Points) != 1 {
		t.Fatalf("Series() = (%v, %v), want one point", res.Points, err)
	}
	if got := src.callCount(); got != 1 {
		t.Errorf("source called %d times, want 1", got)
	}
}

func TestBetween(t *testing.T) {
	points := []Point{{Date: "2026-01-01"}, {Date: "2026-02-01"}, {Date: "2026-03-01"}}
	tests := []struct {
		name     string
		from, to string
		want     []string
	}{
		{name: "inclusive bounds", from: "2026-01-01", to: "2026-02-01", want: []string{"2026-01-01", "2026-02-01"}},
		{name: "bounds between points", from: "2026-01-15", to: "2026-12-31", want: []string{"2026-02-01", "2026-03-01"}},
		{name: "range before the data", from: "2020-01-01", to: "2020-12-31", want: []string{}},
		{name: "range after the data", from: "2027-01-01", to: "2027-12-31", want: []string{}},
		{name: "inverted range", from: "2026-03-01", to: "2026-01-01", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dates(between(points, tt.from, tt.to)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("between(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
