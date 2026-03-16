// internal/executor/postgres.go
package executor

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type postgresAdapter struct{}

func (a *postgresAdapter) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// Enforce read-only at session level — works for both URL and key=value DSN formats.
	if _, err := db.Exec("SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY"); err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres: failed to set read-only: %w", err)
	}
	return db, nil
}

func (a *postgresAdapter) HasLimit(query string) (bool, int, error) {
	return hasLimit(query)
}

func (a *postgresAdapter) InjectLimit(query string, maxRows, offset int) string {
	return injectLimit(query, maxRows, offset)
}

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
