package logger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAccessColorsStatusInConsoleAndNotInFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        int
		expectedColor string
	}{
		{
			name:          "success",
			status:        200,
			expectedColor: successColor,
		},
		{
			name:          "client error",
			status:        404,
			expectedColor: warningColor,
		},
		{
			name:          "server error",
			status:        500,
			expectedColor: errorColor,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var console bytes.Buffer
			var file bytes.Buffer
			log := &Logger{
				minLevel:      levelInfo,
				console:       &console,
				file:          &file,
				consoleColors: true,
			}

			log.Access(AccessLogEntry{
				Method:   "GET",
				Target:   "/api/test",
				Proto:    "HTTP/1.1",
				Status:   tc.status,
				Bytes:    42,
				Duration: 1234 * time.Microsecond,
				RemoteIP: "127.0.0.1",
			})

			coloredStatus := tc.expectedColor + fmt.Sprintf("%d", tc.status) + resetColor
			consoleOutput := console.String()
			if !strings.Contains(consoleOutput, coloredStatus) {
				t.Fatalf("expected colored status in console log, got:\n%s", consoleOutput)
			}

			fileOutput := file.String()
			if strings.Contains(fileOutput, "\x1b[") {
				t.Fatalf("expected no ANSI code in file log, got:\n%s", fileOutput)
			}
			if !strings.Contains(fileOutput, fmt.Sprintf("status=%d", tc.status)) {
				t.Fatalf("expected plain status in file log, got:\n%s", fileOutput)
			}
		})
	}
}

func TestAccessNoColorWhenConsoleColorsDisabled(t *testing.T) {
	t.Parallel()

	var console bytes.Buffer
	log := &Logger{
		minLevel:      levelInfo,
		console:       &console,
		consoleColors: false,
	}

	log.Access(AccessLogEntry{
		Method:   "GET",
		Target:   "/api/test",
		Proto:    "HTTP/1.1",
		Status:   200,
		Bytes:    0,
		Duration: 0,
		RemoteIP: "127.0.0.1",
	})

	output := console.String()
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("expected no ANSI code when console colors are disabled, got:\n%s", output)
	}
	if !strings.Contains(output, "status=200") {
		t.Fatalf("expected status in output, got:\n%s", output)
	}
}

func TestAccessDurationUsesMilliseconds(t *testing.T) {
	t.Parallel()

	var console bytes.Buffer
	log := &Logger{
		minLevel:      levelInfo,
		console:       &console,
		consoleColors: false,
	}

	log.Access(AccessLogEntry{
		Method:   "GET",
		Target:   "/api/test",
		Proto:    "HTTP/1.1",
		Status:   200,
		Bytes:    1,
		Duration: 5678900 * time.Nanosecond,
		RemoteIP: "127.0.0.1",
	})

	output := console.String()
	if !strings.Contains(output, "duration=5.678900ms") {
		t.Fatalf("expected millisecond duration in output, got:\n%s", output)
	}
}
