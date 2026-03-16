// internal/executor/mariadb.go
package executor

import (
	"database/sql"
	"fmt"

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

func (a *mariadbAdapter) HasLimit(query string) (bool, int, error) {
	return hasLimit(query)
}

func (a *mariadbAdapter) InjectLimit(query string, maxRows, offset int) string {
	return injectLimit(query, maxRows, offset)
}

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
