package config

import (
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

// Config holds runtime options loaded from environment variables.
type Config struct {
	Port    string
	Logging LoggingConfig
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

// Load reads environment variables and applies safe defaults.
func Load() Config {
	// `.env` is optional. Missing file is ignored.
	_ = godotenv.Load()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	logLevel := strings.ToUpper(strings.TrimSpace(getStringEnv("LOG_LEVEL", defaultLogLevel)))

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
	}
}

// Address returns a valid net/http listen address such as ":8888".
func (c Config) Address() string {
	if strings.HasPrefix(c.Port, ":") {
		return c.Port
	}

	return ":" + c.Port
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
