// internal/executor/mariadb_test.go
package executor_test

import (
	"github.com/eduardoserete/humanized-query/internal/executor"
	"testing"
)

func TestMariadbAdapter_InjectLimit(t *testing.T) {
	a, _ := executor.New("mariadb")
	q := a.InjectLimit("SELECT id FROM users", 100, 0)
	expected := "SELECT id FROM users LIMIT 100 OFFSET 0"
	if q != expected {
		t.Errorf("got %q, want %q", q, expected)
	}
}

func TestMariadbAdapter_HasLimit_true(t *testing.T) {
	a, _ := executor.New("mariadb")
	has, n, err := a.HasLimit("SELECT id FROM users LIMIT 25")
	if err != nil || !has || n != 25 {
		t.Errorf("expected has=true n=25, got has=%v n=%d err=%v", has, n, err)
	}
}

func TestMariadbAdapter_HasLimit_false(t *testing.T) {
	a, _ := executor.New("mariadb")
	has, _, err := a.HasLimit("SELECT id FROM users")
	if err != nil || has {
		t.Errorf("expected has=false, got has=%v err=%v", has, err)
	}
}

func TestNewAdapter_unsupported(t *testing.T) {
	_, err := executor.New("dynamodb")
	if err == nil {
		t.Error("expected error for unsupported dialect")
	}
}
