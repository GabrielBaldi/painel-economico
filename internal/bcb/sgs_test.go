package bcb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"painel-economico/internal/indicator"
	"painel-economico/internal/upstream"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSplitWindows(t *testing.T) {
	tests := []struct {
		name     string
		from, to string
		want     [][2]string
	}{
		{name: "single day", from: "2026-01-05", to: "2026-01-05", want: [][2]string{{"2026-01-05", "2026-01-05"}}},
		{name: "fits in one window", from: "2020-01-01", to: "2026-11-08", want: [][2]string{{"2020-01-01", "2026-11-08"}}},
		{name: "exactly one window", from: "2000-01-01", to: "2009-12-31", want: [][2]string{{"2000-01-01", "2009-12-31"}}},
		{name: "one day over", from: "2000-01-01", to: "2010-01-01", want: [][2]string{
			{"2000-01-01", "2009-12-31"},
			{"2010-01-01", "2010-01-01"},
		}},
		{name: "full history", from: "2000-01-01", to: "2026-11-08", want: [][2]string{
			{"2000-01-01", "2009-12-31"},
			{"2010-01-01", "2019-12-31"},
			{"2020-01-01", "2026-11-08"},
		}},
		{name: "empty when from is after to", from: "2026-02-01", to: "2026-01-01", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got [][2]string
			for _, w := range splitWindows(day(tt.from), day(tt.to), maxWindowYears) {
				got = append(got, [2]string{w.from.Format(time.DateOnly), w.to.Format(time.DateOnly)})
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitWindows(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    []indicator.Point
		wantErr string
	}{
		{
			name: "daily values",
			body: `[{"data":"02/01/2025","valor":"6.1923"},{"data":"03/01/2025","valor":"6.1647"}]`,
			want: []indicator.Point{{Date: "2025-01-02", Value: 6.1923}, {Date: "2025-01-03", Value: 6.1647}},
		},
		{
			name: "monthly values, including a negative one",
			body: `[{"data":"01/07/2025","valor":"0.26"},{"data":"01/08/2025","valor":"-0.11"}]`,
			want: []indicator.Point{{Date: "2025-07-01", Value: 0.26}, {Date: "2025-08-01", Value: -0.11}},
		},
		{
			name: "skips dates without a value",
			body: `[{"data":"01/07/2025","valor":""},{"data":"01/08/2025","valor":"1"}]`,
			want: []indicator.Point{{Date: "2025-08-01", Value: 1}},
		},
		{name: "empty array", body: `[]`, want: []indicator.Point{}},
		{name: "not json", body: `<html>erro</html>`, wantErr: "decode response"},
		{name: "bad date", body: `[{"data":"2025-01-02","valor":"1"}]`, wantErr: "parse date"},
		{name: "bad value", body: `[{"data":"02/01/2025","valor":"1,5"}]`, wantErr: "parse value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse([]byte(tt.body))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parse() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

// fakeSGS answers like SGS: one point on the first and last day of each
// requested window, or 404 for windows listed in empty.
type fakeSGS struct {
	mu       sync.Mutex
	requests []string
	empty    map[string]bool // dataInicial values that have no data
	fail     bool
}

func (f *fakeSGS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("dataInicial"), q.Get("dataFinal")
	f.mu.Lock()
	f.requests = append(f.requests, r.URL.Path+" "+from+"-"+to)
	f.mu.Unlock()

	switch {
	case f.fail:
		w.WriteHeader(http.StatusBadGateway)
	case f.empty[from]:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"erro":{"statusCode":404,"detail":"Value(s) not found"}}`))
	default:
		_, _ = fmt.Fprintf(w, `[{"data":%q,"valor":"1.5"},{"data":%q,"valor":"2.5"}]`, from, to)
	}
}

func TestSeriesFetch(t *testing.T) {
	tests := []struct {
		name         string
		fake         *fakeSGS
		wantDates    []string
		wantRequests int // 0: not checked
		wantErr      bool
	}{
		{
			name: "joins the windows in date order",
			fake: &fakeSGS{},
			wantDates: []string{
				"2000-01-01", "2009-12-31",
				"2010-01-01", "2019-12-31",
				"2020-01-01", "2026-11-08",
			},
			wantRequests: 3,
		},
		{
			name:         "treats a 404 window as empty",
			fake:         &fakeSGS{empty: map[string]bool{"01/01/2010": true}},
			wantDates:    []string{"2000-01-01", "2009-12-31", "2020-01-01", "2026-11-08"},
			wantRequests: 3,
		},
		{
			// The first failure cancels the other windows, so how many
			// requests reach the server is not deterministic.
			name:    "fails when a window fails",
			fake:    &fakeSGS{fail: true},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.fake)
			client := New(srv.URL, upstream.New(upstream.Options{MaxAttempts: 1}))

			points, err := client.Series(1).Fetch(context.Background(), day("2000-01-01"), day("2026-11-08"))
			srv.Close() // waits for in-flight handlers before we read fake.requests

			if tt.wantRequests > 0 && len(tt.fake.requests) != tt.wantRequests {
				t.Errorf("made %d requests, want %d: %v", len(tt.fake.requests), tt.wantRequests, tt.fake.requests)
			}
			for _, req := range tt.fake.requests {
				if !strings.HasPrefix(req, "/bcdata.sgs.1/dados ") {
					t.Errorf("unexpected request path %q", req)
				}
			}
			if tt.wantErr {
				if err == nil {
					t.Fatal("Fetch() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}
			var dates []string
			for _, p := range points {
				dates = append(dates, p.Date)
			}
			if !reflect.DeepEqual(dates, tt.wantDates) {
				t.Errorf("Fetch() dates = %v, want %v", dates, tt.wantDates)
			}
		})
	}
}
