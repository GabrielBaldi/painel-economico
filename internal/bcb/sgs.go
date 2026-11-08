// Package bcb reads time series from the SGS API of the Banco Central do
// Brasil (https://dadosabertos.bcb.gov.br).
package bcb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"painel-economico/internal/indicator"
	"painel-economico/internal/upstream"
)

// DefaultBaseURL is the public SGS endpoint.
const DefaultBaseURL = "https://api.bcb.gov.br/dados/serie"

// maxWindowYears is the widest range SGS accepts for daily series; beyond it
// the API answers 406. Every query is split into windows of at most this
// size, which also works for monthly series.
const maxWindowYears = 10

// sgsDate is the dd/MM/yyyy layout SGS uses in queries and responses.
const sgsDate = "02/01/2006"

// Getter fetches a URL and returns the response body.
type Getter interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

// Client builds sources for SGS series.
type Client struct {
	baseURL string
	http    Getter
}

// New returns a Client that queries baseURL through http.
func New(baseURL string, http Getter) *Client {
	return &Client{baseURL: baseURL, http: http}
}

// Series returns a source for the SGS series with the given code.
func (c *Client) Series(code int) Series {
	return Series{client: c, code: code}
}

// Series is one SGS series. It implements indicator.Source.
type Series struct {
	client *Client
	code   int
}

// Fetch returns the observations between from and to, querying each window
// concurrently and joining the results in date order.
func (s Series) Fetch(ctx context.Context, from, to time.Time) ([]indicator.Point, error) {
	windows := splitWindows(from, to, maxWindowYears)
	results := make([][]indicator.Point, len(windows))

	g, ctx := errgroup.WithContext(ctx)
	for i, w := range windows {
		g.Go(func() error {
			points, err := s.fetchWindow(ctx, w)
			results[i] = points
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return slices.Concat(results...), nil
}

func (s Series) fetchWindow(ctx context.Context, w window) ([]indicator.Point, error) {
	url := fmt.Sprintf("%s/bcdata.sgs.%d/dados?formato=json&dataInicial=%s&dataFinal=%s",
		s.client.baseURL, s.code, w.from.Format(sgsDate), w.to.Format(sgsDate))

	body, err := s.client.http.Get(ctx, url)
	var statusErr *upstream.StatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		return nil, nil // SGS answers 404 when a window has no values
	}
	if err != nil {
		return nil, fmt.Errorf("sgs series %d: %w", s.code, err)
	}
	points, err := parse(body)
	if err != nil {
		return nil, fmt.Errorf("sgs series %d: %w", s.code, err)
	}
	return points, nil
}

type window struct {
	from, to time.Time
}

// splitWindows splits from..to (inclusive) into consecutive windows of at
// most the given number of years.
func splitWindows(from, to time.Time, years int) []window {
	var windows []window
	for start := from; !start.After(to); {
		end := start.AddDate(years, 0, -1)
		if end.After(to) {
			end = to
		}
		windows = append(windows, window{from: start, to: end})
		start = end.AddDate(0, 0, 1)
	}
	return windows
}

// parse decodes an SGS response: [{"data":"dd/MM/yyyy","valor":"1.23"}].
func parse(body []byte) ([]indicator.Point, error) {
	var rows []struct {
		Date  string `json:"data"`
		Value string `json:"valor"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	points := make([]indicator.Point, 0, len(rows))
	for _, row := range rows {
		if row.Value == "" {
			continue // a date with no observation
		}
		date, err := time.Parse(sgsDate, row.Date)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", row.Date, err)
		}
		value, err := strconv.ParseFloat(row.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("parse value %q: %w", row.Value, err)
		}
		points = append(points, indicator.Point{Date: date.Format(time.DateOnly), Value: value})
	}
	return points, nil
}
