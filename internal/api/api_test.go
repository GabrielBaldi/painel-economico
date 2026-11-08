package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"painel-economico/internal/indicator"
)

var selic = indicator.Indicator{ID: "selic", Name: "Selic", Unit: "% a.a.", Frequency: indicator.Daily}

// fakeService records the range it was asked for and returns canned results.
type fakeService struct {
	gotFrom, gotTo time.Time
	stale          bool
	err            error
}

func (f *fakeService) Catalog() []indicator.Indicator { return []indicator.Indicator{selic} }

func (f *fakeService) Series(ctx context.Context, id string, from, to time.Time) (indicator.Result, error) {
	f.gotFrom, f.gotTo = from, to
	if f.err != nil {
		return indicator.Result{}, f.err
	}
	if id != "selic" {
		return indicator.Result{}, indicator.ErrNotFound
	}
	return indicator.Result{
		Indicator: selic,
		Points:    []indicator.Point{{Date: "2026-01-02", Value: 15}},
		Stale:     f.stale,
		FetchedAt: time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC),
	}, nil
}

func newTestHandler(svc Service) http.Handler {
	static := fstest.MapFS{
		"index.html":        {Data: []byte("<!doctype html><title>Painel</title>")},
		"assets/app-123.js": {Data: []byte("console.log(1)")},
	}
	return NewHandler(svc, static, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestSeriesEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		svc        *fakeService
		wantStatus int
		wantBody   string // substring of the JSON body
		wantFrom   string // range passed to the service, when checked
		wantTo     string
	}{
		{
			name:       "explicit range",
			target:     "/api/indicators/selic?from=2026-01-01&to=2026-02-01",
			svc:        &fakeService{},
			wantStatus: http.StatusOK,
			wantBody:   `"points":[{"date":"2026-01-02","value":15}],"stale":false,"fetchedAt":"2026-11-08T10:00:00Z"`,
			wantFrom:   "2026-01-01",
			wantTo:     "2026-02-01",
		},
		{
			name:       "only from given",
			target:     "/api/indicators/selic?from=2025-06-30",
			svc:        &fakeService{},
			wantStatus: http.StatusOK,
			wantBody:   `"indicator":{"id":"selic"`,
			wantFrom:   "2025-06-30",
		},
		{
			name:       "stale data is flagged",
			target:     "/api/indicators/selic",
			svc:        &fakeService{stale: true},
			wantStatus: http.StatusOK,
			wantBody:   `"stale":true`,
		},
		{
			name:       "malformed from",
			target:     "/api/indicators/selic?from=01/01/2026",
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   `Parâmetro \"from\" inválido`,
		},
		{
			name:       "impossible date",
			target:     "/api/indicators/selic?to=2026-02-30",
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   `Parâmetro \"to\" inválido`,
		},
		{
			name:       "inverted range",
			target:     "/api/indicators/selic?from=2026-02-01&to=2026-01-01",
			svc:        &fakeService{},
			wantStatus: http.StatusBadRequest,
			wantBody:   `deve ser anterior`,
		},
		{
			name:       "unknown indicator",
			target:     "/api/indicators/bitcoin",
			svc:        &fakeService{},
			wantStatus: http.StatusNotFound,
			wantBody:   `"error":"Indicador não encontrado`,
		},
		{
			name:       "upstream down and nothing cached",
			target:     "/api/indicators/selic",
			svc:        &fakeService{err: errors.New("fetch selic: timeout")},
			wantStatus: http.StatusBadGateway,
			wantBody:   `"error":"A fonte dos dados não respondeu`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(newTestHandler(tt.svc), http.MethodGet, tt.target)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want JSON", ct)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want it to contain %s", rec.Body.String(), tt.wantBody)
			}
			if tt.wantFrom != "" && tt.svc.gotFrom.Format(time.DateOnly) != tt.wantFrom {
				t.Errorf("service got from = %s, want %s", tt.svc.gotFrom.Format(time.DateOnly), tt.wantFrom)
			}
			if tt.wantTo != "" && tt.svc.gotTo.Format(time.DateOnly) != tt.wantTo {
				t.Errorf("service got to = %s, want %s", tt.svc.gotTo.Format(time.DateOnly), tt.wantTo)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	today := time.Date(2026, time.November, 8, 22, 30, 0, 0, indicator.Brasilia)
	tests := []struct {
		name, from, to    string
		wantFrom, wantTo  string
		wantErrorContains string
	}{
		{name: "defaults to the last year", wantFrom: "2025-11-08", wantTo: "2026-11-08"},
		{name: "from defaults to a year before to", to: "2024-02-29", wantFrom: "2023-03-01", wantTo: "2024-02-29"},
		{name: "same day", from: "2026-01-01", to: "2026-01-01", wantFrom: "2026-01-01", wantTo: "2026-01-01"},
		{name: "datetime is rejected", from: "2026-01-01T00:00:00Z", wantErrorContains: `"from"`},
		{name: "from after default to", from: "2027-01-01", wantErrorContains: "anterior"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, msg := parseRange(tt.from, tt.to, today)
			if tt.wantErrorContains != "" {
				if !strings.Contains(msg, tt.wantErrorContains) {
					t.Fatalf("parseRange() message = %q, want it to contain %q", msg, tt.wantErrorContains)
				}
				return
			}
			if msg != "" {
				t.Fatalf("parseRange() message = %q, want none", msg)
			}
			if got := from.Format(time.DateOnly); got != tt.wantFrom {
				t.Errorf("from = %s, want %s", got, tt.wantFrom)
			}
			if got := to.Format(time.DateOnly); got != tt.wantTo {
				t.Errorf("to = %s, want %s", got, tt.wantTo)
			}
		})
	}
}

func TestCatalogEndpoint(t *testing.T) {
	rec := serve(newTestHandler(&fakeService{}), http.MethodGet, "/api/indicators")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Indicators []indicator.Indicator `json:"indicators"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Indicators) != 1 || body.Indicators[0] != selic {
		t.Errorf("indicators = %+v, want [selic]", body.Indicators)
	}
}

func TestRouting(t *testing.T) {
	tests := []struct {
		name, method, target string
		wantStatus           int
		wantBody             string
		wantCacheControl     string
	}{
		{name: "health check", method: http.MethodGet, target: "/healthz", wantStatus: http.StatusOK, wantBody: `{"status":"ok"}`},
		{name: "unknown API route is JSON", method: http.MethodGet, target: "/api/nope", wantStatus: http.StatusNotFound, wantBody: `"error"`},
		{name: "wrong method", method: http.MethodPost, target: "/api/indicators", wantStatus: http.StatusMethodNotAllowed},
		{name: "front-end entry point", method: http.MethodGet, target: "/", wantStatus: http.StatusOK, wantBody: "<title>Painel</title>", wantCacheControl: "no-cache"},
		{name: "fingerprinted asset", method: http.MethodGet, target: "/assets/app-123.js", wantStatus: http.StatusOK, wantCacheControl: "public, max-age=31536000, immutable"},
		{name: "missing file", method: http.MethodGet, target: "/nada.txt", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(newTestHandler(&fakeService{}), tt.method, tt.target)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantCacheControl != "" && rec.Header().Get("Cache-Control") != tt.wantCacheControl {
				t.Errorf("Cache-Control = %q, want %q", rec.Header().Get("Cache-Control"), tt.wantCacheControl)
			}
			if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("security headers missing")
			}
		})
	}
}
