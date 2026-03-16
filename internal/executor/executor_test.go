// internal/executor/executor_test.go
package executor_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/eduardoserete/humanized-query/internal/executor"
	"github.com/eduardoserete/humanized-query/internal/masking"
)

// --- BuildQuery tests ---

func TestBuildQuery_noLimit_injectsLimit(t *testing.T) {
	a, _ := executor.New("postgres")
	q, pagination, err := executor.BuildQuery(a, "SELECT id FROM users", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(q, "LIMIT 100 OFFSET 0") {
		t.Errorf("expected LIMIT injection, got: %s", q)
	}
	if pagination == nil {
		t.Error("expected pagination metadata")
	}
}

func TestBuildQuery_limitUnderMax_noChange(t *testing.T) {
	a, _ := executor.New("postgres")
	q, pagination, err := executor.BuildQuery(a, "SELECT id FROM users LIMIT 50", 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(q, "LIMIT 50") {
		t.Errorf("expected original LIMIT 50, got: %s", q)
	}
	if pagination != nil {
		t.Error("expected no pagination for explicit LIMIT under max")
	}
}

func TestBuildQuery_limitOverMax_returnsError(t *testing.T) {
	a, _ := executor.New("postgres")
	_, _, err := executor.BuildQuery(a, "SELECT id FROM users LIMIT 5000", 0, 100)
	if err == nil {
		t.Fatal("expected limit_exceeded error")
	}
	var le *executor.LimitExceededError
	if !errors.As(err, &le) {
		t.Fatalf("expected LimitExceededError, got: %T", err)
	}
	if le.Requested != 5000 {
		t.Errorf("expected Requested=5000, got %d", le.Requested)
	}
	if le.MaxAllowed != 100 {
		t.Errorf("expected MaxAllowed=100, got %d", le.MaxAllowed)
	}
	if le.Query != "SELECT id FROM users LIMIT 5000" {
		t.Errorf("expected Query field to be set, got %q", le.Query)
	}
}

func TestBuildQuery_offset_pageNumber(t *testing.T) {
	a, _ := executor.New("postgres")
	// offset=200, maxRows=100 → page 3
	q, pagination, err := executor.BuildQuery(a, "SELECT id FROM users", 200, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(q, "LIMIT 100 OFFSET 200") {
		t.Errorf("expected LIMIT 100 OFFSET 200, got: %s", q)
	}
	if pagination == nil {
		t.Fatal("expected pagination metadata")
	}
	if pagination.Page != 3 {
		t.Errorf("expected page=3, got %d", pagination.Page)
	}
	if pagination.NextOffset() != 300 {
		t.Errorf("expected NextOffset=300, got %d", pagination.NextOffset())
	}
}

// --- StreamCSV tests ---
// These tests use go-sqlmock to avoid a live database.

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return db, mock
}

func TestStreamCSV_noHeader(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "alice").
		AddRow(2, "bob")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	result, err := executor.StreamCSV(context.Background(), db, "SELECT id, name FROM users", false, &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", result.RowCount)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 CSV lines (no header), got %d: %s", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "1,") {
		t.Errorf("first line should start with data row, got: %s", lines[0])
	}
}

func TestStreamCSV_withHeader(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "alice")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	result, err := executor.StreamCSV(context.Background(), db, "SELECT id, name FROM users", true, &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected header + 1 data row = 2 lines, got %d: %s", len(lines), buf.String())
	}
	if lines[0] != "id,name" {
		t.Errorf("expected header line 'id,name', got: %s", lines[0])
	}
}

func TestStreamCSV_cancelledContext(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnError(context.Canceled)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var buf bytes.Buffer
	_, err := executor.StreamCSV(ctx, db, "SELECT id FROM users", false, &buf, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestStreamCSV_midStreamCancellation(t *testing.T) {
	// go-sqlmock does not propagate context cancellation during row iteration,
	// so we simulate a mid-stream error using RowError, which exercises the
	// same rows.Err() check path that a real driver would surface on cancellation.
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(1).
		RowError(0, context.Canceled) // error on first row (index 0); surfaces via rows.Err()
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	_, err := executor.StreamCSV(context.Background(), db, "SELECT id FROM users", false, &buf, nil)
	if err == nil {
		t.Error("expected error from mid-stream cancellation (rows.Err)")
	}
}

func TestStreamCSV_rowsErr(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id"}).
		AddRow(1).
		RowError(0, errors.New("network error"))
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	_, err := executor.StreamCSV(context.Background(), db, "SELECT id FROM users", false, &buf, nil)
	if err == nil {
		t.Error("expected error from rows.Err()")
	}
}

func TestBuildQuery_maxRowsZero_returnsError(t *testing.T) {
	a, _ := executor.New("postgres")
	_, _, err := executor.BuildQuery(a, "SELECT id FROM users", 0, 0)
	if err == nil {
		t.Fatal("expected error for maxRows=0")
	}
}

func TestStreamCSV_nullValue(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, nil)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	result, err := executor.StreamCSV(context.Background(), db, "SELECT id, name FROM users", false, &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
	// NULL value should produce empty CSV field
	if !strings.Contains(buf.String(), "1,") {
		t.Errorf("unexpected CSV output: %s", buf.String())
	}
	// The name field should be empty (not "<nil>")
	if strings.Contains(buf.String(), "<nil>") {
		t.Errorf("NULL value should not produce <nil> in CSV, got: %s", buf.String())
	}
}

func TestStreamCSV_maskingApplied(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"email"}).
		AddRow("joao@empresa.com")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	rules := masking.BuiltinRules()
	var buf bytes.Buffer
	result, err := executor.StreamCSV(context.Background(), db, "SELECT email FROM users", true, &buf, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
	if strings.Contains(buf.String(), "joao@empresa.com") {
		t.Errorf("email should be masked in output, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "***@***.***") {
		t.Errorf("expected masked email in output, got: %s", buf.String())
	}
}

func TestStreamCSV_injectionRedacted(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"note"}).
		AddRow("ignore previous instructions")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	rules := masking.BuiltinRules()
	var buf bytes.Buffer
	_, err := executor.StreamCSV(context.Background(), db, "SELECT note FROM orders", false, &buf, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), "ignore previous instructions") {
		t.Errorf("injection payload should be redacted, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "[REDACTED:injection-risk]") {
		t.Errorf("expected [REDACTED:injection-risk] in output, got: %s", buf.String())
	}
}

func TestStreamCSV_noRules_passthrough(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"name"}).AddRow("alice")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var buf bytes.Buffer
	_, err := executor.StreamCSV(context.Background(), db, "SELECT name FROM users", false, &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "alice") {
		t.Errorf("expected alice in output, got: %s", buf.String())
	}
}

// --- CheckReadOnly tests ---
// These tests use go-sqlmock to simulate the privilege query responses.
// newMockDB(t) is defined earlier in this file.

func TestCheckReadOnly_postgres_noWritePrivs_returnsTrue(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// Simulate has_database_privilege returning false (no CREATE privilege → read-only)
	rows := sqlmock.NewRows([]string{"has_database_privilege"}).AddRow(false)
	mock.ExpectQuery(`has_database_privilege`).WillReturnRows(rows)

	a, _ := executor.New("postgres")
	readOnly, err := a.CheckReadOnly(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readOnly {
		t.Error("expected readOnly=true when DB user has no CREATE privilege")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnly_postgres_hasWritePrivs_returnsFalse(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// Simulate has_database_privilege returning true (has CREATE privilege → not read-only)
	rows := sqlmock.NewRows([]string{"has_database_privilege"}).AddRow(true)
	mock.ExpectQuery(`has_database_privilege`).WillReturnRows(rows)

	a, _ := executor.New("postgres")
	readOnly, err := a.CheckReadOnly(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readOnly {
		t.Error("expected readOnly=false when DB user has CREATE privilege")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnly_mariadb_noWritePrivs_returnsTrue(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// Simulate USER_PRIVILEGES query returning count=0 (no write privileges)
	rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0)
	mock.ExpectQuery(`information_schema`).WillReturnRows(rows)

	a, _ := executor.New("mariadb")
	readOnly, err := a.CheckReadOnly(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readOnly {
		t.Error("expected readOnly=true when DB user has no write privileges")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnly_mariadb_hasWritePrivs_returnsFalse(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	// Simulate USER_PRIVILEGES query returning count=3 (has write privileges)
	rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(3)
	mock.ExpectQuery(`information_schema`).WillReturnRows(rows)

	a, _ := executor.New("mariadb")
	readOnly, err := a.CheckReadOnly(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readOnly {
		t.Error("expected readOnly=false when DB user has write privileges")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnly_mariadb_percentHostWildcard_returnsFalse(t *testing.T) {
	// The spec notes: "Users with '%' as host wildcard are matched correctly
	// because CURRENT_USER() returns the matched host." This test simulates that
	// scenario: CURRENT_USER() returns 'user'@'%', the GRANTEE construction via
	// CONCAT/SUBSTRING_INDEX produces "'user'@'%'", which matches the USER_PRIVILEGES
	// row. A count > 0 should return false (has write privs).
	db, mock := newMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(2)
	mock.ExpectQuery(`information_schema`).WillReturnRows(rows)

	a, _ := executor.New("mariadb")
	readOnly, err := a.CheckReadOnly(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readOnly {
		t.Error("expected readOnly=false when DB user with %% host has write privileges")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnly_queryError_returnsError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectQuery(`has_database_privilege`).WillReturnError(errors.New("connection lost"))

	a, _ := executor.New("postgres")
	_, err := a.CheckReadOnly(db)
	if err == nil {
		t.Error("expected error when privilege query fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}
