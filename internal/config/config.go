package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultPort = "8888"

const (
	defaultLogLevel         = "INFO"
	defaultLogFilePath      = "logs/api.log"
	defaultLogMaxSizeMB     = 10
	defaultLogMaxBackups    = 5
	defaultLogMaxAgeDays    = 30
	defaultLogCompress      = true
	defaultLogToFile        = true
	defaultLogConsoleColors = true
)

const (
	DocsRendererSwaggerUI = "swagger-ui"
	DocsRendererStoplight = "stoplight"
	DocsRendererScalar    = "scalar"
)

const defaultDocsRenderer = DocsRendererSwaggerUI

// Config holds runtime options loaded from environment variables.
type Config struct {
	Port    string
	Logging LoggingConfig
	Docs    DocsConfig
}

// LoggingConfig controls console/file output and rotation behavior.
type LoggingConfig struct {
	Level         string
	ToFile        bool
	FilePath      string
	MaxSizeMB     int
	MaxBackups    int
	MaxAgeDays    int
	Compress      bool
	ConsoleColors bool
}

// DocsConfig controls the OpenAPI docs UI renderer.
type DocsConfig struct {
	Renderer string
}

// Load reads environment variables and applies safe defaults.
func Load() Config {
	// `.env` is optional. Missing file is ignored.
	_ = godotenv.Load()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	logLevel := strings.ToUpper(strings.TrimSpace(getStringEnv("LOG_LEVEL", defaultLogLevel)))
	docsRenderer := normalizeDocsRenderer(getStringEnv("DOCS_RENDERER", defaultDocsRenderer))
	if docsRenderer == "" {
		docsRenderer = defaultDocsRenderer
	}

	return Config{
		Port: port,
		Logging: LoggingConfig{
			Level:         logLevel,
			ToFile:        getBoolEnv("LOG_TO_FILE", defaultLogToFile),
			FilePath:      strings.TrimSpace(getStringEnv("LOG_FILE_PATH", defaultLogFilePath)),
			MaxSizeMB:     getIntEnv("LOG_MAX_SIZE_MB", defaultLogMaxSizeMB),
			MaxBackups:    getIntEnv("LOG_MAX_BACKUPS", defaultLogMaxBackups),
			MaxAgeDays:    getIntEnv("LOG_MAX_AGE_DAYS", defaultLogMaxAgeDays),
			Compress:      getBoolEnv("LOG_COMPRESS", defaultLogCompress),
			ConsoleColors: getBoolEnv("LOG_CONSOLE_COLORS", defaultLogConsoleColors),
		},
		Docs: DocsConfig{
			Renderer: docsRenderer,
		},
	}
}

// Address returns a valid net/http listen address such as ":8888".
func (c Config) Address() string {
	if strings.HasPrefix(c.Port, ":") {
		return c.Port
	}

	return ":" + c.Port
}

// Validate enforces config invariants and fails fast on invalid settings.
func (c Config) Validate() error {
	if c.Docs.Renderer != "" && !isValidDocsRenderer(c.Docs.Renderer) {
		return fmt.Errorf("invalid DOCS_RENDERER %q (allowed: %s, %s, %s)", c.Docs.Renderer, DocsRendererSwaggerUI, DocsRendererStoplight, DocsRendererScalar)
	}

	return nil
}

func getStringEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func getIntEnv(key string, fallback int) int {
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

func getBoolEnv(key string, fallback bool) bool {
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

func normalizeDocsRenderer(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case DocsRendererSwaggerUI, "swagger", "swaggerui":
		return DocsRendererSwaggerUI
	case DocsRendererStoplight, "stoplight-elements", "elements":
		return DocsRendererStoplight
	case DocsRendererScalar:
		return DocsRendererScalar
	default:
		return ""
	}
}

func isValidDocsRenderer(value string) bool {
	switch value {
	case DocsRendererSwaggerUI, DocsRendererStoplight, DocsRendererScalar:
		return true
	default:
		return false
	}
}
