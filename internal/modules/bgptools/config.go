package bgptools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type moduleConfig struct {
	TableURL       string
	ASNsURL        string
	CacheDir       string
	TableMaxAge    time.Duration
	ASNsMaxAge     time.Duration
	UpdateInterval time.Duration
	HTTPTimeout    time.Duration
	UserAgent      string
	Preload        bool
	MaxBatch       int
}

func loadConfig() (moduleConfig, error) {
	tableMaxAge, err := durationEnv("BGPTOOLS_TABLE_MAX_AGE", "30m")
	if err != nil {
		return moduleConfig{}, err
	}

	asnsMaxAge, err := durationEnv("BGPTOOLS_ASNS_MAX_AGE", "24h")
	if err != nil {
		return moduleConfig{}, err
	}

	updateInterval, err := durationEnv("BGPTOOLS_UPDATE_INTERVAL", "1h")
	if err != nil {
		return moduleConfig{}, err
	}

	httpTimeout, err := durationEnv("BGPTOOLS_HTTP_TIMEOUT", "5m")
	if err != nil {
		return moduleConfig{}, err
	}

	maxBatch, err := intEnv("BGPTOOLS_MAX_BATCH", 256)
	if err != nil {
		return moduleConfig{}, err
	}

	cfg := moduleConfig{
		TableURL:       stringEnv("BGPTOOLS_TABLE_URL", "https://bgp.tools/table.jsonl"),
		ASNsURL:        stringEnv("BGPTOOLS_ASNS_URL", "https://bgp.tools/asns.csv"),
		CacheDir:       stringEnv("BGPTOOLS_CACHE_DIR", "cache/bgptools"),
		TableMaxAge:    tableMaxAge,
		ASNsMaxAge:     asnsMaxAge,
		UpdateInterval: updateInterval,
		HTTPTimeout:    httpTimeout,
		UserAgent:      stringEnv("BGPTOOLS_USER_AGENT", "huma-golang-api-template/bgptools"),
		Preload:        boolEnv("BGPTOOLS_PRELOAD", false),
		MaxBatch:       maxBatch,
	}

	if err := cfg.validate(); err != nil {
		return moduleConfig{}, err
	}

	return cfg, nil
}

func (c moduleConfig) validate() error {
	if strings.TrimSpace(c.TableURL) == "" {
		return fmt.Errorf("BGPTOOLS_TABLE_URL cannot be empty")
	}
	if strings.TrimSpace(c.ASNsURL) == "" {
		return fmt.Errorf("BGPTOOLS_ASNS_URL cannot be empty")
	}
	if strings.TrimSpace(c.CacheDir) == "" {
		return fmt.Errorf("BGPTOOLS_CACHE_DIR cannot be empty")
	}
	if c.TableMaxAge <= 0 {
		return fmt.Errorf("BGPTOOLS_TABLE_MAX_AGE must be > 0")
	}
	if c.ASNsMaxAge <= 0 {
		return fmt.Errorf("BGPTOOLS_ASNS_MAX_AGE must be > 0")
	}
	if c.UpdateInterval <= 0 {
		return fmt.Errorf("BGPTOOLS_UPDATE_INTERVAL must be > 0")
	}
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("BGPTOOLS_HTTP_TIMEOUT must be > 0")
	}
	if c.MaxBatch <= 0 {
		return fmt.Errorf("BGPTOOLS_MAX_BATCH must be > 0")
	}

	return nil
}

func (c moduleConfig) tablePath() string {
	return filepath.Join(c.CacheDir, "table.jsonl")
}

func (c moduleConfig) asnsPath() string {
	return filepath.Join(c.CacheDir, "asns.csv")
}

func stringEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func intEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}

	return parsed, nil
}

func durationEnv(key, fallback string) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a valid positive duration (example: 30m)", key)
	}

	return parsed, nil
}
