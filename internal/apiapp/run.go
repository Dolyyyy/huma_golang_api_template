package apiapp

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	"github.com/Dolyyyy/huma_golang_api_template/internal/logger"
	"github.com/Dolyyyy/huma_golang_api_template/internal/modulekit"
	"github.com/Dolyyyy/huma_golang_api_template/internal/server"
)

// Run boots and serves the API until shutdown and returns a process exit code.
func Run() int {
	cfg := config.Load()
	bootstrapLogger := logger.NewConsoleFallback()
	if err := cfg.Validate(); err != nil {
		bootstrapLogger.Critical("invalid configuration", "error", err)
		return 1
	}
	if err := modulekit.ValidateAll(); err != nil {
		bootstrapLogger.Critical("invalid module configuration", "error", err)
		return 1
	}

	appLogger, err := logger.New(cfg.Logging)
	if err != nil {
		bootstrapLogger.Critical("failed to initialize logger", "error", err)
		return 1
	}
	defer func() {
		if closeErr := appLogger.Close(); closeErr != nil {
			appLogger.Error("failed to close logger", "error", closeErr)
		}
	}()

	srv := server.New(cfg)

	errCh := make(chan error, 1)
	go func() {
		appLogger.Success("API is up and ready to listen")
		logEnabledModules(appLogger)

		startupURLs := server.DiscoverStartupURLs(cfg.Address())
		for _, startupURL := range startupURLs {
			appLogger.Info(startupURL.Label, "url", startupURL.URL)
		}

		if len(startupURLs) > 0 {
			appLogger.Info("API docs", "url", startupURLs[0].URL+"/docs")
		}

		if listenErr := srv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			errCh <- listenErr
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	select {
	case runErr := <-errCh:
		appLogger.Critical("server failed", "error", runErr)
		return 1
	case sig := <-stopCh:
		appLogger.Warning("received signal, shutting down", "signal", sig.String())
	}

	// Keep shutdown bounded so process termination never blocks forever.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Critical("graceful shutdown failed", "error", err)
		return 1
	}

	appLogger.Info("server stopped")
	return 0
}

func logEnabledModules(appLogger *logger.Logger) {
	enabled := modulekit.IDs()
	if len(enabled) == 0 {
		appLogger.Info("No optional modules enabled")
		return
	}

	appLogger.Info("Optional modules enabled", "modules", strings.Join(enabled, ","))
}
