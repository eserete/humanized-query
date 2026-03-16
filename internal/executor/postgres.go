// internal/executor/postgres.go
package executor

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type postgresAdapter struct{}

func (a *postgresAdapter) Open(dsn string) (*sql.DB, error) {
	// Append read-only session parameter
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn = dsn + sep + "options=-c+default_transaction_read_only%3Don"
	return sql.Open("postgres", dsn)
}

var limitRe = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)

func (a *postgresAdapter) HasLimit(query string) (bool, int, error) {
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

func (a *postgresAdapter) InjectLimit(query string, maxRows, offset int) string {
	query = strings.TrimRight(strings.TrimSpace(query), ";")
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", query, maxRows, offset)
}
