// cmd/hq/commands/db_test.go
package commands

import (
	"testing"
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
