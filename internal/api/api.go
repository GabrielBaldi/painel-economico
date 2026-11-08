// Package api is the HTTP layer: JSON endpoints, the embedded front-end and
// the middleware around them.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"painel-economico/internal/indicator"
)

// Service is what the handlers need from the indicator service.
type Service interface {
	Catalog() []indicator.Indicator
	Series(ctx context.Context, id string, from, to time.Time) (indicator.Result, error)
}

type handler struct {
	svc Service
	log *slog.Logger
}

// NewHandler returns the root handler: the API under /api, a health check at
// /healthz and the static front-end (from static) everywhere else.
func NewHandler(svc Service, static fs.FS, log *slog.Logger) http.Handler {
	h := &handler{svc: svc, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/indicators", h.catalog)
	mux.HandleFunc("GET /api/indicators/{id}", h.series)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Rota não encontrada.")
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /", staticFiles(static))

	return logRequests(log, securityHeaders(mux))
}

func (h *handler) catalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"indicators": h.svc.Catalog()})
}

func (h *handler) series(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, to, msg := parseRange(query.Get("from"), query.Get("to"), time.Now().In(indicator.Brasilia))
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	res, err := h.svc.Series(r.Context(), r.PathValue("id"), from, to)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, res)
	case errors.Is(err, indicator.ErrNotFound):
		writeError(w, http.StatusNotFound, "Indicador não encontrado. Veja a lista em /api/indicators.")
	case r.Context().Err() != nil:
		// The client went away; nobody is left to read a response.
	default:
		h.log.Error("series unavailable", "indicator", r.PathValue("id"), "error", err)
		writeError(w, http.StatusBadGateway, "A fonte dos dados não respondeu. Tente novamente em alguns minutos.")
	}
}

// parseRange validates the optional from/to query parameters (YYYY-MM-DD).
// Missing to means today; missing from means one year before to. On failure
// it returns a message for the client.
func parseRange(fromParam, toParam string, today time.Time) (from, to time.Time, msg string) {
	to = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if toParam != "" {
		var err error
		if to, err = time.Parse(time.DateOnly, toParam); err != nil {
			return from, to, `Parâmetro "to" inválido: use o formato AAAA-MM-DD.`
		}
	}
	from = to.AddDate(-1, 0, 0)
	if fromParam != "" {
		var err error
		if from, err = time.Parse(time.DateOnly, fromParam); err != nil {
			return from, to, `Parâmetro "from" inválido: use o formato AAAA-MM-DD.`
		}
	}
	if from.After(to) {
		return from, to, `O parâmetro "from" deve ser anterior ou igual a "to".`
	}
	return from, to, ""
}

// staticFiles serves the built front-end. Vite fingerprints everything under
// /assets, so those files can be cached forever; index.html must not be.
func staticFiles(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the status is already sent; nothing useful to do on error
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
