// internal/executor/adapter.go
package executor

import (
	"database/sql"
	"fmt"
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
