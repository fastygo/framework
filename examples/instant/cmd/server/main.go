// examples/instant serves one prebuilt HTML document from memory.
//
// Routes:
//
//	/           — instant article
//	/index.html — same article
//	/healthz    — plain liveness response
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/fastygo/example-instant/internal/site"
	"github.com/fastygo/framework/pkg/app"
	instantstore "github.com/fastygo/framework/pkg/web/instant"
)

const (
	defaultInstantMaxPages = 1
	defaultInstantMaxBytes = 64 * 1024
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	handler, err := site.NewHandler(instantstore.Options{
		MaxPages: envInt("APP_INSTANT_MAX_PAGES", defaultInstantMaxPages),
		MaxBytes: envInt("APP_INSTANT_MAX_BYTES", defaultInstantMaxBytes),
	})
	if err != nil {
		slog.Error("failed to build instant store", "error", err)
		os.Exit(1)
	}

	stats := handler.StoreStats()
	slog.Info("instant store ready",
		"pages", stats.Pages,
		"bytes", stats.Bytes,
		"max_pages", stats.MaxPages,
		"max_bytes", stats.MaxBytes,
	)

	server := &http.Server{
		Addr:              cfg.AppBind,
		Handler:           handler,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    cfg.HTTPMaxHeaderBytes,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("instant server listening", "addr", cfg.AppBind)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown failed", "error", err)
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped with error", "error", err)
			os.Exit(1)
		}
	}
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
