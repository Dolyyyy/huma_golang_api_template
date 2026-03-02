package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dolyyyy/golang_api_template/internal/config"
	"github.com/Dolyyyy/golang_api_template/internal/logger"
	"github.com/Dolyyyy/golang_api_template/internal/server"
)

func main() {
	cfg := config.Load()
	bootstrapLogger := logger.NewConsoleFallback()
	appLogger, err := logger.New(cfg.Logging)
	if err != nil {
		bootstrapLogger.Critical("failed to initialize logger", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := appLogger.Close(); err != nil {
			appLogger.Error("failed to close logger", "error", err)
		}
	}()

	srv := server.New(cfg)

	errCh := make(chan error, 1)
	go func() {
		appLogger.Success("API is up and ready to listen")

		startupURLs := server.DiscoverStartupURLs(cfg.Address())
		for _, startupURL := range startupURLs {
			appLogger.Info(startupURL.Label, "url", startupURL.URL)
		}

		if len(startupURLs) > 0 {
			appLogger.Info("API docs", "url", startupURLs[0].URL+"/docs")
		}
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		appLogger.Critical("server failed", "error", err)
		return
	case sig := <-stopCh:
		appLogger.Warning("received signal, shutting down", "signal", sig.String())
	}

	// Keep shutdown bounded so process termination never blocks forever.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Critical("graceful shutdown failed", "error", err)
		return
	}

	appLogger.Info("server stopped")
}
