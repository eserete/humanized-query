# Chunk 5: CheckReadOnly — read-only user verification

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `CheckReadOnly(db *sql.DB) (bool, error)` to the `Adapter` interface and implement it for Postgres and MariaDB; call it from `query.go` after `adapter.Open` and emit a warning to stderr if write privileges are detected.

**Architecture:** Three tasks. Task 5.1 adds `CheckReadOnly` to the `Adapter` interface in `adapter.go`, implements it in `postgres.go` and `mariadb.go` using dialect-specific privilege queries, and tests both implementations with `go-sqlmock`. Task 5.2 adds the call site in `query.go`: after `adapter.Open`, call `CheckReadOnly` and emit the warning if `!isReadOnly`. Task 5.3 adds a white-box call-site test in `cmd/hq/commands/db_test.go` that verifies the warning string is emitted to stderr when a mock adapter reports write privileges.

**Tech Stack:** Go stdlib (`database/sql`), `github.com/DATA-DOG/go-sqlmock`, `github.com/lib/pq` (postgres driver, already in go.mod), `github.com/go-sql-driver/mysql` (mariadb driver, already in go.mod).

---

### Task 5.1: Add CheckReadOnly to Adapter interface + implementations

**Files:**
- Modify: `internal/executor/adapter.go`
- Modify: `internal/executor/postgres.go`
- Modify: `internal/executor/mariadb.go`
- Modify: `internal/executor/executor_test.go`

- [ ] **Step 1: Write failing tests in `executor_test.go`**

Add to the end of `internal/executor/executor_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — confirm new tests fail**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/executor/... 2>&1 | head -15
```

Expected: compile error — `a.CheckReadOnly undefined` (method doesn't exist yet on interface).

- [ ] **Step 3: Add `CheckReadOnly` to the `Adapter` interface in `adapter.go`**

In `internal/executor/adapter.go`, replace:

```go
// Adapter abstracts dialect-specific behavior.
type Adapter interface {
	// Open returns a read-only *sql.DB for this dialect.
	Open(dsn string) (*sql.DB, error)
	// InjectLimit returns the query with LIMIT/OFFSET appended if not already present.
	InjectLimit(query string, maxRows, offset int) string
	// HasLimit returns true if the query already contains a LIMIT clause.
	HasLimit(query string) (bool, int, error)
}
```

With:

```go
// Adapter abstracts dialect-specific behavior.
type Adapter interface {
	// Open returns a read-only *sql.DB for this dialect.
	Open(dsn string) (*sql.DB, error)
	// InjectLimit returns the query with LIMIT/OFFSET appended if not already present.
	InjectLimit(query string, maxRows, offset int) string
	// HasLimit returns true if the query already contains a LIMIT clause.
	HasLimit(query string) (bool, int, error)
	// CheckReadOnly queries the database server to determine whether the current
	// session user has any write privileges. Returns true if the user is
	// effectively read-only (no write privs detected), false if write privs are
	// present. Returns an error only if the privilege query itself fails.
	CheckReadOnly(db *sql.DB) (bool, error)
}
```

- [ ] **Step 4: Implement `CheckReadOnly` in `postgres.go`**

In `internal/executor/postgres.go`, add the following method after the `InjectLimit` method:

```go
// CheckReadOnly returns true if the current DB user does NOT have CREATE privilege
// on the current database (proxy for write access). Returns false if CREATE is granted.
func (a *postgresAdapter) CheckReadOnly(db *sql.DB) (bool, error) {
	var hasCreate bool
	err := db.QueryRow(
		`SELECT has_database_privilege(current_user, current_database(), 'CREATE')`,
	).Scan(&hasCreate)
	if err != nil {
		return false, fmt.Errorf("postgres: privilege check failed: %w", err)
	}
	return !hasCreate, nil
}
```

No import changes needed — `fmt` is already imported in `postgres.go`.

- [ ] **Step 5: Implement `CheckReadOnly` in `mariadb.go`**

First, add `"fmt"` to the import block in `internal/executor/mariadb.go`.

Replace:

```go
import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)
```

With:

```go
import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)
```

Then add the following method after the `InjectLimit` method:

```go
// CheckReadOnly returns true if the current DB user has no write privileges
// (INSERT, UPDATE, DELETE, DROP, ALTER, CREATE) in information_schema.USER_PRIVILEGES.
// Returns false if one or more write privileges are detected.
//
// The GRANTEE construction using CONCAT/SUBSTRING_INDEX matches the exact format
// stored in USER_PRIVILEGES (e.g. 'user'@'host' or 'user'@'%'). CURRENT_USER()
// returns the host as actually matched by the server, so '%' wildcard users are
// handled correctly.
func (a *mariadbAdapter) CheckReadOnly(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.USER_PRIVILEGES
		WHERE GRANTEE = CONCAT("'", SUBSTRING_INDEX(CURRENT_USER(), '@', 1), "'@'",
		                       SUBSTRING_INDEX(CURRENT_USER(), '@', -1), "'")
		AND PRIVILEGE_TYPE IN ('INSERT','UPDATE','DELETE','DROP','ALTER','CREATE')
	`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("mariadb: privilege check failed: %w", err)
	}
	return count == 0, nil
}
```

- [ ] **Step 6: Run tests — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/executor/... -v 2>&1 | tail -35
```

Expected: all tests PASS including the 6 new CheckReadOnly tests (2 postgres, 3 mariadb, 1 error).

- [ ] **Step 7: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add internal/executor/adapter.go internal/executor/postgres.go internal/executor/mariadb.go internal/executor/executor_test.go
git commit -m "feat: add CheckReadOnly to Adapter interface with postgres and mariadb implementations"
```

---

### Task 5.2: Call CheckReadOnly in query.go and emit warning

**Files:**
- Modify: `cmd/hq/commands/query.go`

There are no new unit tests in this task — the `CheckReadOnly` method is fully tested in Task 5.1, and the call-site warning path is tested in Task 5.3. The post-implementation full test suite run in Step 2 ensures no regressions.

- [ ] **Step 1: Insert CheckReadOnly call after `adapter.Open` in `query.go`**

In `cmd/hq/commands/query.go`, find:

```go
			// Layer 1: read-only connection
			db, err := adapter.Open(dbCfg.DSN)
			if err != nil {
				return writeError("connection_error", err.Error())
			}
			defer db.Close()
```

Replace with:

```go
			// Layer 1: read-only connection
			db, err := adapter.Open(dbCfg.DSN)
			if err != nil {
				return writeError("connection_error", err.Error())
			}
			defer db.Close()

			// Layer 5: warn if DB user has write privileges
			if isReadOnly, roErr := adapter.CheckReadOnly(db); roErr == nil && !isReadOnly {
				fmt.Fprintf(os.Stderr, "# warning: database user has write permissions — a read-only user is strongly recommended\n")
			}
```

- [ ] **Step 2: Run full test suite — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./... 2>&1 | tail -15
```

Expected: all tests PASS across all packages.

- [ ] **Step 3: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add cmd/hq/commands/query.go
git commit -m "feat: call CheckReadOnly after Open and emit write-privilege warning to stderr"
```

---

### Task 5.3: Call-site warning test in commands package

**Files:**
- Modify: `cmd/hq/commands/db_test.go`

This task adds a white-box test (package `commands`) that directly calls `checkReadOnlyAndWarn` — a small extracted helper — to verify the warning string is written to stderr when `isReadOnly=false`. To make this testable without a live DB, Task 5.3 first extracts the warning logic into an unexported helper, then tests it.

- [ ] **Step 1: Extract warning logic into a helper function in `query.go`**

In `cmd/hq/commands/query.go`, add this helper at the bottom of the file:

```go
// checkReadOnlyAndWarn calls adapter.CheckReadOnly and writes a warning to w
// if the database user has write privileges. Errors from CheckReadOnly are silently
// ignored (the warning is best-effort; a privilege check failure should not block queries).
func checkReadOnlyAndWarn(adapter executor.Adapter, db *sql.DB, w io.Writer) {
	if isReadOnly, err := adapter.CheckReadOnly(db); err == nil && !isReadOnly {
		fmt.Fprintf(w, "# warning: database user has write permissions — a read-only user is strongly recommended\n")
	}
}
```

Also add `"database/sql"` and `"io"` to the import block if not already present. Check the existing imports: `os` is already there (which satisfies `io.Writer` via `os.Stderr`), but `io` and `database/sql` need to be added if absent. The current imports are:

```go
import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    "github.com/eduardoserete/humanized-query/internal/audit"
    "github.com/eduardoserete/humanized-query/internal/cache"
    "github.com/eduardoserete/humanized-query/internal/config"
    "github.com/eduardoserete/humanized-query/internal/executor"
    "github.com/eduardoserete/humanized-query/internal/masking"
    "github.com/eduardoserete/humanized-query/internal/policy"
    "github.com/spf13/cobra"
)
```

Add `"database/sql"` and `"io"` to the stdlib block:

```go
import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"

    "github.com/eduardoserete/humanized-query/internal/audit"
    "github.com/eduardoserete/humanized-query/internal/cache"
    "github.com/eduardoserete/humanized-query/internal/config"
    "github.com/eduardoserete/humanized-query/internal/executor"
    "github.com/eduardoserete/humanized-query/internal/masking"
    "github.com/eduardoserete/humanized-query/internal/policy"
    "github.com/spf13/cobra"
)
```

Then update the inline call in `RunE` to use the helper. Replace:

```go
			// Layer 5: warn if DB user has write privileges
			if isReadOnly, roErr := adapter.CheckReadOnly(db); roErr == nil && !isReadOnly {
				fmt.Fprintf(os.Stderr, "# warning: database user has write permissions — a read-only user is strongly recommended\n")
			}
```

With:

```go
			// Layer 5: warn if DB user has write privileges
			checkReadOnlyAndWarn(adapter, db, os.Stderr)
```

- [ ] **Step 2: Write failing test in `db_test.go`**

The `commands` package tests use `package commands` (white-box). Add the following to the end of `cmd/hq/commands/db_test.go`:

```go
import (
    // existing imports stay; add:
    "bytes"
    "database/sql"
    "strings"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/eduardoserete/humanized-query/internal/executor"
)
```

**Note:** `db_test.go` currently imports only `"testing"`. Replace the import block entirely:

```go
import (
    "bytes"
    "database/sql"
    "strings"
    "testing"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/eduardoserete/humanized-query/internal/executor"
)
```

Then add the test function:

```go
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
```

- [ ] **Step 3: Run tests — confirm new tests fail**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./cmd/hq/commands/... 2>&1 | head -15
```

Expected: compile error — `checkReadOnlyAndWarn undefined` (function not yet extracted from Task 5.2).

**Note:** If Task 5.2 was already completed before Task 5.3, the tests should fail at runtime (FAIL), not at compile time. Either outcome is acceptable — what matters is that they fail before implementation.

- [ ] **Step 4: Run tests — confirm they pass (after Task 5.2 Step 1 is done)**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./cmd/hq/commands/... -v 2>&1 | tail -20
```

Expected: all tests PASS including the 3 new `TestCheckReadOnlyAndWarn_*` tests.

- [ ] **Step 5: Run full test suite — confirm no regressions**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./... 2>&1 | tail -15
```

Expected: all tests PASS across all packages.

- [ ] **Step 6: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add cmd/hq/commands/query.go cmd/hq/commands/db_test.go
git commit -m "test: add call-site tests for checkReadOnlyAndWarn helper"
```
