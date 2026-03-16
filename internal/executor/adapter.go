// internal/executor/adapter.go
package executor

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Adapter abstracts dialect-specific behavior.
type Adapter interface {
	// Open returns a read-only *sql.DB for this dialect.
	Open(dsn string) (*sql.DB, error)
	// InjectLimit returns the query with LIMIT/OFFSET appended if not already present.
	InjectLimit(query string, maxRows, offset int) string
	// HasLimit returns true if the query already contains a LIMIT clause.
	HasLimit(query string) (bool, int, error)
}

// New returns the adapter for the given dialect.
// Returns error for unsupported dialects.
func New(dialect string) (Adapter, error) {
	switch dialect {
	case "postgres":
		return &postgresAdapter{}, nil
	case "mariadb", "mysql":
		return &mariadbAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %q", dialect)
	}
}

var limitRe = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)

// hasLimit is a shared implementation used by both adapters.
func hasLimit(query string) (bool, int, error) {
	m := limitRe.FindStringSubmatch(query)
	if m == nil {
		return false, 0, nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return false, 0, fmt.Errorf("invalid LIMIT value: %w", err)
	}
	return true, n, nil
}

// injectLimit is a shared implementation used by both adapters.
func injectLimit(query string, maxRows, offset int) string {
	query = strings.TrimRight(strings.TrimSpace(query), ";")
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", query, maxRows, offset)
}
