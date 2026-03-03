package templatectl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type cliUI struct {
	stdout  io.Writer
	stderr  io.Writer
	colors  bool
	spinner bool
	tty     bool
	mu      sync.Mutex
}

func newCLIUI(stdout, stderr io.Writer, noColor, noSpinner bool) *cliUI {
	useColors := !noColor && terminalWriter(stdout) && strings.TrimSpace(os.Getenv("NO_COLOR")) == ""
	useSpinner := !noSpinner && terminalWriter(stdout)
	useTTY := terminalWriter(stdout) && terminalWriter(os.Stdin)

	return &cliUI{
		stdout:  stdout,
		stderr:  stderr,
		colors:  useColors,
		spinner: useSpinner,
		tty:     useTTY,
	}
}

func (u *cliUI) info(format string, args ...any) {
	u.println(u.stdout, colorCyan, "INFO", format, args...)
}

func (u *cliUI) success(format string, args ...any) {
	u.println(u.stdout, colorGreen, "OK", format, args...)
}

func (u *cliUI) warn(format string, args ...any) {
	u.println(u.stderr, colorYellow, "WARN", format, args...)
}

func (u *cliUI) failure(format string, args ...any) {
	u.println(u.stderr, colorRed, "ERR", format, args...)
}

func (u *cliUI) print(format string, args ...any) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Fprintf(u.stdout, format, args...)
}

func (u *cliUI) runStep(label string, fn func() error) error {
	if !u.spinner {
		u.info("%s", label)
		err := fn()
		if err != nil {
			u.failure("%s failed", label)
			return err
		}
		u.success("%s", label)
		return nil
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		frames := []string{"|", "/", "-", "\\"}
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()

		index := 0
		for {
			select {
			case <-stop:
				u.clearSpinnerLine(label)
				close(done)
				return
			case <-ticker.C:
				frame := frames[index%len(frames)]
				index++
				u.renderSpinnerLine(frame, label)
			}
		}
	}()

	err := fn()
	close(stop)
	<-done

	if err != nil {
		u.failure("%s failed", label)
		return err
	}

	u.success("%s", label)
	return nil
}

func (u *cliUI) println(target io.Writer, colorCode, level, format string, args ...any) {
	u.mu.Lock()
	defer u.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	prefix := fmt.Sprintf("[%s]", level)
	if u.colors {
		prefix = fmt.Sprintf("%s[%s]%s", colorCode, level, colorReset)
	}

	fmt.Fprintf(target, "%s %s\n", prefix, message)
}

func (u *cliUI) renderSpinnerLine(frame, label string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	fmt.Fprintf(u.stdout, "\r[%s] %s", frame, label)
}

func (u *cliUI) clearSpinnerLine(label string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	erase := strings.Repeat(" ", len(label)+8)
	fmt.Fprintf(u.stdout, "\r%s\r", erase)
}

func (u *cliUI) confirmYesNo(question string) (bool, error) {
	if !u.tty {
		return true, nil
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		u.mu.Lock()
		prefix := "[CONFIRM]"
		if u.colors {
			prefix = fmt.Sprintf("%s[CONFIRM]%s", colorMagenta, colorReset)
		}
		fmt.Fprintf(u.stdout, "%s %s ", prefix, question)
		u.mu.Unlock()

		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return false, err
		}

		answer := strings.ToLower(strings.TrimSpace(line))
		switch answer {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		default:
			u.warn(`please answer with "y" or "n"`)
		}
	}
}

func terminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}

const (
	colorReset   = "\x1b[0m"
	colorBold    = "\x1b[1m"
	colorRed     = "\x1b[31m"
	colorGreen   = "\x1b[32m"
	colorYellow  = "\x1b[33m"
	colorBlue    = "\x1b[34m"
	colorMagenta = "\x1b[35m"
	colorCyan    = "\x1b[36m"
	colorGray    = "\x1b[90m"
)
