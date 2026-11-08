package ibge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"painel-economico/internal/indicator"
	"painel-economico/internal/upstream"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    []indicator.Point
		wantErr string
	}{
		{
			name: "moving-quarter periods map to the last month",
			body: `[{"NC":"1","MC":"2","V":"6.5","D1C":"1","D2C":"4099","D3C":"202501"},
			        {"NC":"1","MC":"2","V":"5.2","D1C":"1","D2C":"4099","D3C":"202609"}]`,
			want: []indicator.Point{{Date: "2025-01-01", Value: 6.5}, {Date: "2026-09-01", Value: 5.2}},
		},
		{
			name: "skips SIDRA missing-value symbols",
			body: `[{"V":"...","D3C":"202501"},{"V":"-","D3C":"202502"},{"V":"X","D3C":"202503"},{"V":"7.0","D3C":"202504"}]`,
			want: []indicator.Point{{Date: "2025-04-01", Value: 7}},
		},
		{name: "empty range", body: `[]`, want: []indicator.Point{}},
		{name: "plain-text error body", body: `Parâmetro V (Variável) com código 9999 inexistente`, wantErr: "decode response"},
		{name: "bad period", body: `[{"V":"7.0","D3C":"2025T1"}]`, wantErr: "parse period"},
		{name: "bad value", body: `[{"V":"sete","D3C":"202501"}]`, wantErr: "parse value"},
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

func TestVariableFetch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"V":"5.3","D3C":"202608"}]`))
	}))
	defer srv.Close()
	client := New(srv.URL, upstream.New(upstream.Options{MaxAttempts: 1}))

	from := time.Date(2012, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.November, 8, 15, 0, 0, 0, time.UTC)
	points, err := client.Variable(6381, 4099).Fetch(context.Background(), from, to)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if want := "/t/6381/n1/all/v/4099/p/201203-202611/f/c/h/n"; gotPath != want {
		t.Errorf("requested %q, want %q", gotPath, want)
	}
	if want := []indicator.Point{{Date: "2026-08-01", Value: 5.3}}; !reflect.DeepEqual(points, want) {
		t.Errorf("Fetch() = %v, want %v", points, want)
	}
}

func TestVariableFetchUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Tabela inexistente", http.StatusBadRequest)
	}))
	defer srv.Close()
	client := New(srv.URL, upstream.New(upstream.Options{MaxAttempts: 1}))

	_, err := client.Variable(1, 1).Fetch(context.Background(), time.Now(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "sidra table 1") {
		t.Fatalf("Fetch() error = %v, want a wrapped upstream error", err)
	}
}
