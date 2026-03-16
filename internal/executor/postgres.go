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
