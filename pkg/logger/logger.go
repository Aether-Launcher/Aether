package logger

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogEntry describes a single log item
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Target    string `json:"target"`
	Message   string `json:"message"`
}

var (
	mu         sync.Mutex
	entries    []LogEntry
	maxEntries = 1000
	appCtx     context.Context
	emitter    func(context.Context, string, ...interface{})
	origStdout *os.File
	origStderr *os.File
	pipeReader *os.File
	pipeWriter *os.File
	hookOnce   sync.Once
)

var tagRegex = regexp.MustCompile(`^\[([^\]]+)\]\s*(.*)$`)

func init() {
	InitStdoutHook()
}

// InitStdoutHook redirects os.Stdout and os.Stderr so all fmt.Print/log output
// is captured into the log buffer and sent to Wails Dev Logs.
func InitStdoutHook() {
	hookOnce.Do(func() {
		origStdout = os.Stdout
		origStderr = os.Stderr

		r, w, err := os.Pipe()
		if err != nil {
			return
		}
		pipeReader = r
		pipeWriter = w
		os.Stdout = w
		os.Stderr = w

		go func() {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				// Echo to the real OS stdout so CLI/terminal continues to show output
				if origStdout != nil {
					origStdout.WriteString(line + "\n")
				}
				ingestLine(line)
			}
		}()
	})
}

func ingestLine(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	level := "INFO"
	target := "System"
	message := trimmed

	// Check if line matches [Target] Message
	matches := tagRegex.FindStringSubmatch(trimmed)
	if len(matches) == 3 {
		target = matches[1]
		message = matches[2]
	}

	lower := strings.ToLower(trimmed)
	targetLower := strings.ToLower(target)

	if strings.HasPrefix(targetLower, "sandbox") {
		level = "SANDBOX"
	} else if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.Contains(lower, "panic:") || strings.Contains(lower, "exception") {
		level = "ERROR"
	} else if strings.Contains(lower, "warn") {
		level = "WARN"
	} else if strings.Contains(lower, "debug") {
		level = "DEBUG"
	}

	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05.000"),
		Level:     level,
		Target:    target,
		Message:   message,
	}

	recordEntry(entry)
}

func recordEntry(entry LogEntry) {
	mu.Lock()
	entries = append(entries, entry)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	ctx := appCtx
	emit := emitter
	mu.Unlock()

	if ctx != nil && emit != nil {
		emit(ctx, "log:event", entry)
	}
}

// SetEmitter registers the Wails runtime EventsEmit callback and context
func SetEmitter(ctx context.Context, emit func(context.Context, string, ...interface{})) {
	mu.Lock()
	defer mu.Unlock()
	appCtx = ctx
	emitter = emit
}

// Log records a new log entry, buffers it, and emits a Wails event if connected
func Log(level, target, message string) {
	entry := LogEntry{
		Timestamp: time.Now().Format("15:04:05.000"),
		Level:     level,
		Target:    target,
		Message:   message,
	}

	recordEntry(entry)

	out := origStdout
	if out == nil {
		out = os.Stdout
	}
	out.WriteString(fmt.Sprintf("[%s] [%s] [%s] %s\n", entry.Timestamp, entry.Level, entry.Target, entry.Message))
}

// Info logs an informational message
func Info(target, message string) {
	Log("INFO", target, message)
}

// Warn logs a warning message
func Warn(target, message string) {
	Log("WARN", target, message)
}

// Error logs an error message
func Error(target, message string) {
	Log("ERROR", target, message)
}

// Sandbox logs an extension sandbox message
func Sandbox(target, message string) {
	Log("SANDBOX", target, message)
}

// Debug logs a debug message
func Debug(target, message string) {
	Log("DEBUG", target, message)
}

// GetEntries returns a copy of all buffered log entries
func GetEntries() []LogEntry {
	mu.Lock()
	defer mu.Unlock()
	copied := make([]LogEntry, len(entries))
	copy(copied, entries)
	return copied
}

// Clear flushes all buffered entries
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	entries = make([]LogEntry, 0)
}
