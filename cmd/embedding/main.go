// Command embedding runs the HTTP server of the embedding service. It stops
// gracefully on SIGTERM/SIGINT.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/AugustoOjd/riftplay_embedding_service/internal/config"
)

// shutdownTimeout bounds how long in-flight requests get to finish after
// SIGTERM. Keep it below Railway's draining window.
const shutdownTimeout = 25 * time.Second

func routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func main() {
	level := new(slog.LevelVar)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	if err := run(logger, level); err != nil {
		logger.Error("embedding stopped", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, level *slog.LevelVar) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	level.Set(cfg.LogLevel)

	mux := http.NewServeMux()
	routes(mux)

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
