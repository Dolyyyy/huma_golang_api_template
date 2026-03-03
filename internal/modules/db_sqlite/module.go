package db_sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"

	"github.com/Dolyyyy/huma_golang_api_template/internal/modulekit"
)

type pingResponse struct {
	OK     bool   `json:"ok"`
	Driver string `json:"driver"`
	Detail string `json:"detail"`
	Error  string `json:"error,omitempty"`
}

func init() {
	modulekit.Register(modulekit.Definition{
		ID:          "db-sqlite",
		Name:        "SQLite connector",
		Description: "Add SQLite connector and /db/sqlite/ping endpoint.",
		Routes:      registerRoutes,
	})
}

func registerRoutes(router chi.Router) {
	router.Get("/db/sqlite/ping", pingHandler)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), connectTimeout())
	defer cancel()

	db, err := openDatabase()
	if err != nil {
		respondJSON(w, http.StatusServiceUnavailable, pingResponse{
			OK:     false,
			Driver: "sqlite",
			Detail: "SQLite connection failed.",
			Error:  err.Error(),
		})
		return
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, pingResponse{
			OK:     false,
			Driver: "sqlite",
			Detail: "SQLite ping failed.",
			Error:  err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, pingResponse{
		OK:     true,
		Driver: "sqlite",
		Detail: "SQLite ping successful.",
	})
}

func openDatabase() (*sql.DB, error) {
	dbPath := envOrDefault("SQLITE_PATH", "data/app.db")
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create sqlite directory %q: %w", dbDir, err)
		}
	}

	timeoutMs := envOrDefault("SQLITE_BUSY_TIMEOUT_MS", "5000")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(%s)", filepath.ToSlash(dbPath), timeoutMs)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	return db, nil
}

func connectTimeout() time.Duration {
	value := strings.TrimSpace(os.Getenv("SQLITE_CONNECT_TIMEOUT_SEC"))
	if value == "" {
		return 5 * time.Second
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 5 * time.Second
	}

	return time.Duration(seconds) * time.Second
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func respondJSON(w http.ResponseWriter, statusCode int, payload pingResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
