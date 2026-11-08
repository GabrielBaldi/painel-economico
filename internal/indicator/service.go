package indicator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

// ErrNotFound is returned for an indicator ID that is not in the catalog.
var ErrNotFound = errors.New("indicator not found")

// HistoryStart is the first date the service fetches. Older data mixes
// currencies (the real was born in 1994) and hyperinflation-era values that
// would flatten every chart.
var HistoryStart = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// Brasilia is the time zone the upstreams publish in. Brazil has had no
// daylight saving time since 2019, so a fixed offset is exact.
var Brasilia = time.FixedZone("BRT", -3*60*60)

// fetchTimeout bounds one full refresh of a series, retries included.
const fetchTimeout = 45 * time.Second

// Snapshot is the full history of one indicator as fetched at FetchedAt.
type Snapshot struct {
	Points    []Point
	FetchedAt time.Time
}

// Cache stores snapshots by indicator ID. Get also returns expired entries
// (fresh == false) so the service can fall back to them.
type Cache interface {
	Get(key string) (value Snapshot, fresh, ok bool)
	Set(key string, value Snapshot)
}

// Result is a slice of an indicator's history, ready to be encoded as JSON.
type Result struct {
	Indicator Indicator `json:"indicator"`
	Points    []Point   `json:"points"`
	Stale     bool      `json:"stale"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// Service answers indicator queries from the cache, refreshing a series from
// its upstream when the cached copy has expired.
type Service struct {
	catalog []Indicator
	defs    map[string]Definition
	cache   Cache
	log     *slog.Logger
	now     func() time.Time
	group   singleflight.Group
}

// NewService builds a service for the given definitions, in catalog order.
func NewService(defs []Definition, cache Cache, log *slog.Logger) *Service {
	s := &Service{
		defs:  make(map[string]Definition, len(defs)),
		cache: cache,
		log:   log,
		now:   func() time.Time { return time.Now().In(Brasilia) },
	}
	for _, d := range defs {
		s.catalog = append(s.catalog, d.Indicator)
		s.defs[d.ID] = d
	}
	return s
}

// Catalog lists the available indicators.
func (s *Service) Catalog() []Indicator {
	return s.catalog
}

// Series returns the observations of indicator id between from and to
// (inclusive). If the upstream fails but an older copy is cached, it returns
// that copy with Stale set instead of an error.
func (s *Service) Series(ctx context.Context, id string, from, to time.Time) (Result, error) {
	def, ok := s.defs[id]
	if !ok {
		return Result{}, ErrNotFound
	}
	snap, stale, err := s.snapshot(ctx, def)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Indicator: def.Indicator,
		Points:    between(snap.Points, from.Format(time.DateOnly), to.Format(time.DateOnly)),
		Stale:     stale,
		FetchedAt: snap.FetchedAt,
	}, nil
}

func (s *Service) snapshot(ctx context.Context, def Definition) (snap Snapshot, stale bool, err error) {
	cached, fresh, found := s.cache.Get(def.ID)
	if found && fresh {
		return cached, false, nil
	}

	// Concurrent requests for the same indicator share one upstream fetch.
	ch := s.group.DoChan(def.ID, func() (any, error) { return s.refresh(def) })
	select {
	case <-ctx.Done():
		return Snapshot{}, false, ctx.Err()
	case res := <-ch:
		if res.Err == nil {
			return res.Val.(Snapshot), false, nil
		}
		if found {
			s.log.Warn("upstream failed, serving stale data", "indicator", def.ID, "error", res.Err)
			return cached, true, nil
		}
		return Snapshot{}, false, res.Err
	}
}

// refresh fetches the whole history of a series and caches it. It does not
// use the caller's context: the fetch is shared, so one client giving up
// must not cancel it for the others.
func (s *Service) refresh(def Definition) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	now := s.now()
	points, err := def.Upstream.Fetch(ctx, HistoryStart, now)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch %s: %w", def.ID, err)
	}
	slices.SortFunc(points, func(a, b Point) int { return strings.Compare(a.Date, b.Date) })

	snap := Snapshot{Points: points, FetchedAt: now}
	s.cache.Set(def.ID, snap)
	s.log.Info("series refreshed", "indicator", def.ID, "points", len(points))
	return snap, nil
}

// between returns the points dated from..to (inclusive) of a sorted slice.
// The result shares memory with points and must not be modified.
func between(points []Point, from, to string) []Point {
	start := sort.Search(len(points), func(i int) bool { return points[i].Date >= from })
	end := sort.Search(len(points), func(i int) bool { return points[i].Date > to })
	if end <= start {
		return []Point{}
	}
	return points[start:end]
}
