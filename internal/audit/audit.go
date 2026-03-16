package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single audit log record.
type Entry struct {
	DB         string
	Status     string // "ok" or "rejected"
	RowCount   int
	DurationMs int64
	Error      string
	SQL        string
}

// Logger writes audit entries to a file.
type Logger struct {
	path string
}

// New returns a Logger writing to path.
func New(path string) *Logger {
	return &Logger{path: path}
}

// Log appends an entry to the audit log.
// Non-fatal: returns error but callers should log to stderr and continue.
func (l *Logger) Log(e Entry) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0700); err != nil {
		return fmt.Errorf("audit: mkdir failed: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit: open failed: %w", err)
	}
	defer f.Close()

	line := fmt.Sprintf("%s db=%s status=%s", time.Now().UTC().Format(time.RFC3339), e.DB, e.Status)
	if e.Status == "ok" {
		line += fmt.Sprintf(" rows=%d duration_ms=%d", e.RowCount, e.DurationMs)
	}
	if e.Error != "" {
		line += fmt.Sprintf(" error=%s", e.Error)
	}
	line += fmt.Sprintf(" sql=%q\n", e.SQL)

	_, err = f.WriteString(line)
	return err
}
