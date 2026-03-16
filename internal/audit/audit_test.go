package audit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eduardoserete/humanized-query/internal/audit"
	"github.com/eduardoserete/humanized-query/internal/masking"
)

func TestLog_writesEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	a := audit.New(path, nil)
	err := a.Log(audit.Entry{
		DB:         "postgres_main",
		Status:     "ok",
		RowCount:   10,
		DurationMs: 45,
		SQL:        "SELECT id FROM users LIMIT 10",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	line := string(data)
	if !strings.Contains(line, `db="postgres_main"`) {
		t.Errorf("expected db field in log, got: %s", line)
	}
	if !strings.Contains(line, `status="ok"`) {
		t.Errorf("expected status field in log, got: %s", line)
	}
	if !strings.Contains(line, `sql="SELECT id FROM users LIMIT 10"`) {
		t.Errorf("expected sql field in log, got: %s", line)
	}
}

func TestLog_appendsEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	a := audit.New(path, nil)

	if err := a.Log(audit.Entry{DB: "db1", Status: "ok", SQL: "SELECT 1"}); err != nil {
		t.Fatalf("first log failed: %v", err)
	}
	if err := a.Log(audit.Entry{DB: "db1", Status: "rejected", SQL: "DELETE FROM x"}); err != nil {
		t.Fatalf("second log failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestLog_rejectedEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	a := audit.New(path, nil)

	err := a.Log(audit.Entry{
		DB:     "mydb",
		Status: "rejected",
		Error:  "forbidden_statement",
		SQL:    "UPDATE users SET x=1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `error="forbidden_statement"`) {
		t.Errorf("expected error field in log, got: %s", string(data))
	}
	if strings.Contains(string(data), "rows=") {
		t.Errorf("rejected entry should not contain rows field, got: %s", string(data))
	}
}

func TestLog_masksSQLEmail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	rules := masking.BuiltinRules()
	logger := audit.New(path, rules)

	err := logger.Log(audit.Entry{
		DB:     "prod",
		Status: "ok",
		SQL:    "SELECT * FROM users WHERE email = 'joao@empresa.com'",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := os.ReadFile(path)
	content := string(b)
	if strings.Contains(content, "joao@empresa.com") {
		t.Errorf("email should be masked in audit log, got: %s", content)
	}
	if !strings.Contains(content, "***@***.***") {
		t.Errorf("expected masked email in audit log, got: %s", content)
	}
}

func TestLog_noRules_logsAsIs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	logger := audit.New(path, nil)
	err := logger.Log(audit.Entry{
		DB:     "prod",
		Status: "ok",
		SQL:    "SELECT id FROM users",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "SELECT id FROM users") {
		t.Errorf("SQL should be logged as-is when no rules, got: %s", string(b))
	}
}
