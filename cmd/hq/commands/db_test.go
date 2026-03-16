// cmd/hq/commands/db_test.go
package commands

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/eduardoserete/humanized-query/internal/executor"
)

func TestMaskDSN_postgres(t *testing.T) {
	dsn := "postgres://myuser:secret@localhost:5432/mydb"
	masked := maskDSN(dsn)
	if masked != "postgres://myuser:***@localhost:5432/mydb" {
		t.Errorf("unexpected: %s", masked)
	}
}

func TestMaskDSN_mysql(t *testing.T) {
	dsn := "myuser:secret@tcp(localhost:3306)/mydb"
	masked := maskDSN(dsn)
	if masked != "myuser:***@tcp(localhost:3306)/mydb" {
		t.Errorf("unexpected: %s", masked)
	}
}

func TestMaskDSN_noPassword(t *testing.T) {
	dsn := "postgres://myuser@localhost:5432/mydb"
	masked := maskDSN(dsn)
	if masked != dsn {
		t.Errorf("unexpected change: %s", masked)
	}
}

func TestCheckReadOnlyAndWarn_writesWarning_whenNotReadOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Postgres adapter: simulate CREATE privilege present → not read-only → warning expected
	rows := sqlmock.NewRows([]string{"has_database_privilege"}).AddRow(true)
	mock.ExpectQuery(`has_database_privilege`).WillReturnRows(rows)

	a, _ := executor.New("postgres")
	var buf bytes.Buffer
	checkReadOnlyAndWarn(a, db, &buf)

	got := buf.String()
	want := "# warning: database user has write permissions"
	if !strings.Contains(got, want) {
		t.Errorf("expected warning in stderr output, got: %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnlyAndWarn_noWarning_whenReadOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Postgres adapter: simulate no CREATE privilege → read-only → no warning
	rows := sqlmock.NewRows([]string{"has_database_privilege"}).AddRow(false)
	mock.ExpectQuery(`has_database_privilege`).WillReturnRows(rows)

	a, _ := executor.New("postgres")
	var buf bytes.Buffer
	checkReadOnlyAndWarn(a, db, &buf)

	if buf.Len() != 0 {
		t.Errorf("expected no warning for read-only user, got: %q", buf.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestCheckReadOnlyAndWarn_noWarning_whenCheckFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Simulate privilege query error — warning should be suppressed (best-effort)
	mock.ExpectQuery(`has_database_privilege`).WillReturnError(sql.ErrConnDone)

	a, _ := executor.New("postgres")
	var buf bytes.Buffer
	checkReadOnlyAndWarn(a, db, &buf)

	if buf.Len() != 0 {
		t.Errorf("expected no warning when check errors, got: %q", buf.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}
