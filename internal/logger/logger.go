package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	resetColor    = "\033[0m"
	infoColor     = "\033[36m"
	successColor  = "\033[32m"
	warningColor  = "\033[33m"
	errorColor    = "\033[31m"
	criticalColor = "\033[97;41;1m"
)

type level int

const (
	levelInfo level = iota
	levelSuccess
	levelWarning
	levelError
	levelCritical
)

type Logger struct {
	minLevel      level
	console       io.Writer
	file          io.Writer
	rotatingFile  *lumberjack.Logger
	consoleColors bool
	mu            sync.Mutex
}

// AccessLogEntry represents one HTTP request/response exchange for access logging.
type AccessLogEntry struct {
	Method   string
	Target   string
	Proto    string
	Status   int
	Bytes    int
	Duration time.Duration
	RemoteIP string
}

// New builds a logger with colored console output and optional rotating file logs.
func New(cfg config.LoggingConfig) (*Logger, error) {
	logger := &Logger{
		minLevel:      parseLevel(cfg.Level),
		console:       os.Stdout,
		consoleColors: cfg.ConsoleColors,
	}

	if cfg.ToFile {
		if err := ensureLogDirectory(cfg.FilePath); err != nil {
			return nil, err
		}

		rotator := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}

		logger.file = rotator
		logger.rotatingFile = rotator
	}

	return logger, nil
}

// NewConsoleFallback provides a safe console-only logger for bootstrap errors.
func NewConsoleFallback() *Logger {
	return &Logger{
		minLevel:      levelInfo,
		console:       os.Stdout,
		consoleColors: true,
	}
}

func (l *Logger) Info(message string, fields ...any) {
	l.log(levelInfo, message, fields...)
}

func (l *Logger) Success(message string, fields ...any) {
	l.log(levelSuccess, message, fields...)
}

func (l *Logger) Warning(message string, fields ...any) {
	l.log(levelWarning, message, fields...)
}

func (l *Logger) Error(message string, fields ...any) {
	l.log(levelError, message, fields...)
}

func (l *Logger) Critical(message string, fields ...any) {
	l.log(levelCritical, message, fields...)
}

// Access writes one HTTP access log line with colored status on console output.
func (l *Logger) Access(entry AccessLogEntry) {
	if levelInfo.priority() < l.minLevel.priority() {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	durationText := entry.Duration.Truncate(time.Microsecond).String()
	statusText := strconv.Itoa(entry.Status)

	consoleStatus := statusText
	if l.consoleColors {
		consoleStatus = accessStatusColor(entry.Status) + statusText + resetColor
	}

	plainLine := fmt.Sprintf(
		"%s [HTTP] ip=%s method=%s target=%q proto=%s status=%s bytes=%d duration=%s\n",
		timestamp,
		entry.RemoteIP,
		entry.Method,
		entry.Target,
		entry.Proto,
		statusText,
		entry.Bytes,
		durationText,
	)

	consoleLine := fmt.Sprintf(
		"%s [HTTP] ip=%s method=%s target=%q proto=%s status=%s bytes=%d duration=%s\n",
		timestamp,
		entry.RemoteIP,
		entry.Method,
		entry.Target,
		entry.Proto,
		consoleStatus,
		entry.Bytes,
		durationText,
	)

	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = io.WriteString(l.console, consoleLine)
	if l.file != nil {
		_, _ = io.WriteString(l.file, plainLine)
	}
}

// Close flushes/closes the rotating file sink if file logging is enabled.
func (l *Logger) Close() error {
	if l.rotatingFile == nil {
		return nil
	}

	return l.rotatingFile.Close()
}

func (l *Logger) log(currentLevel level, message string, fields ...any) {
	if currentLevel.priority() < l.minLevel.priority() {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	formattedFields := formatFields(fields...)

	plainLine := fmt.Sprintf("%s [%s] %s%s\n", timestamp, currentLevel.String(), message, formattedFields)
	consoleLevel := "[" + currentLevel.String() + "]"
	if l.consoleColors {
		consoleLevel = currentLevel.color() + consoleLevel + resetColor
	}

	consoleLine := fmt.Sprintf("%s %s %s%s\n", timestamp, consoleLevel, message, formattedFields)

	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = io.WriteString(l.console, consoleLine)
	if l.file != nil {
		_, _ = io.WriteString(l.file, plainLine)
	}
}

func parseLevel(raw string) level {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SUCCESS":
		return levelSuccess
	case "WARNING", "WARN":
		return levelWarning
	case "ERROR":
		return levelError
	case "CRITICAL":
		return levelCritical
	default:
		return levelInfo
	}
}

func (l level) String() string {
	switch l {
	case levelSuccess:
		return "SUCCESS"
	case levelWarning:
		return "WARNING"
	case levelError:
		return "ERROR"
	case levelCritical:
		return "CRITICAL"
	default:
		return "INFO"
	}
}

func (l level) priority() int {
	switch l {
	case levelSuccess:
		return 15
	case levelWarning:
		return 20
	case levelError:
		return 30
	case levelCritical:
		return 40
	default:
		return 10
	}
}

func (l level) color() string {
	switch l {
	case levelSuccess:
		return successColor
	case levelWarning:
		return warningColor
	case levelError:
		return errorColor
	case levelCritical:
		return criticalColor
	default:
		return infoColor
	}
}

func accessStatusColor(status int) string {
	switch {
	case status >= 500:
		return errorColor
	case status >= 400:
		return warningColor
	case status >= 300:
		return infoColor
	case status >= 200:
		return successColor
	default:
		return criticalColor
	}
}

func ensureLogDirectory(filePath string) error {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return fmt.Errorf("LOG_FILE_PATH cannot be empty when LOG_TO_FILE=true")
	}

	directory := filepath.Dir(trimmed)
	if directory == "." || directory == "" {
		return nil
	}

	return os.MkdirAll(directory, 0o755)
}

func formatFields(fields ...any) string {
	if len(fields) == 0 {
		return ""
	}

	if len(fields)%2 != 0 {
		fields = append(fields, "<missing>")
	}

	var b strings.Builder
	for i := 0; i < len(fields); i += 2 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprintf("%v=%v", fields[i], fields[i+1]))
	}

	return b.String()
}
