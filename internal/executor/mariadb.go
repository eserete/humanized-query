// internal/executor/mariadb.go
package executor

import (
	"database/sql"

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
