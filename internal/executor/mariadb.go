// internal/executor/mariadb.go
package executor

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type mariadbAdapter struct{}

func (a *mariadbAdapter) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// go-sql-driver does not expose a DSN-level read-only flag;
	// enforce via post-connect session command.
	if err := setMariaDBReadOnly(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func setMariaDBReadOnly(db *sql.DB) error {
	_, err := db.Exec("SET SESSION TRANSACTION READ ONLY")
	return err
}

var mariaLimitRe = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)`)

func (a *mariadbAdapter) HasLimit(query string) (bool, int, error) {
	m := mariaLimitRe.FindStringSubmatch(query)
	if m == nil {
		return false, 0, nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return false, 0, fmt.Errorf("invalid LIMIT value: %w", err)
	}
	return true, n, nil
}

func (a *mariadbAdapter) InjectLimit(query string, maxRows, offset int) string {
	query = strings.TrimRight(strings.TrimSpace(query), ";")
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", query, maxRows, offset)
}
