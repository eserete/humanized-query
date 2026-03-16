package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eduardoserete/humanized-query/internal/masking"
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
	path  string
	rules []masking.Rule
}

// New returns a Logger writing to path.
// rules are applied to mask SQL before logging. Pass nil for no masking.
func New(path string, rules []masking.Rule) *Logger {
	return &Logger{path: path, rules: rules}
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

	sql := e.SQL
	if len(l.rules) > 0 {
		sql = masking.Apply(sql, l.rules)
	}

	line := fmt.Sprintf("%s db=%q status=%q", time.Now().UTC().Format(time.RFC3339), e.DB, e.Status)
	if e.Status == "ok" {
		line += fmt.Sprintf(" rows=%d duration_ms=%d", e.RowCount, e.DurationMs)
	}
	if e.Error != "" {
		line += fmt.Sprintf(" error=%q", e.Error)
	}
	line += fmt.Sprintf(" sql=%q\n", sql)

	if _, err = f.WriteString(line); err != nil {
		return fmt.Errorf("audit: write failed: %w", err)
	}
	return nil
}
