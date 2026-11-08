// Command server runs the Painel Econômico API and serves the embedded
// front-end.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"painel-economico/internal/api"
	"painel-economico/internal/bcb"
	"painel-economico/internal/cache"
	"painel-economico/internal/config"
	"painel-economico/internal/ibge"
	"painel-economico/internal/indicator"
	"painel-economico/internal/upstream"
	"painel-economico/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	client := upstream.New(upstream.Options{Timeout: cfg.UpstreamTimeout, MaxAttempts: cfg.MaxAttempts, Logger: log})
	svc := indicator.NewService(
		catalog(bcb.New(bcb.DefaultBaseURL, client), ibge.New(ibge.DefaultBaseURL, client)),
		cache.NewMemory[indicator.Snapshot](cfg.CacheTTL, time.Now),
		log,
	)
	static, err := web.Dist()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewHandler(svc, static, log),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      time.Minute, // a cold refresh can take a while
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", "http://localhost:"+cfg.Port, "cache_ttl", cfg.CacheTTL.String())

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
