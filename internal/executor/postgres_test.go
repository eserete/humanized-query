// internal/executor/postgres_test.go
package executor_test

import (
	"github.com/eduardoserete/humanized-query/internal/executor"
	"testing"
)

func TestPostgresAdapter_InjectLimit_noLimit(t *testing.T) {
	a, _ := executor.New("postgres")
	q := a.InjectLimit("SELECT id FROM users", 100, 0)
	expected := "SELECT id FROM users LIMIT 100 OFFSET 0"
	if q != expected {
		t.Errorf("got %q, want %q", q, expected)
	}
}

func TestPostgresAdapter_InjectLimit_withOffset(t *testing.T) {
	a, _ := executor.New("postgres")
	q := a.InjectLimit("SELECT id FROM users", 100, 200)
	expected := "SELECT id FROM users LIMIT 100 OFFSET 200"
	if q != expected {
		t.Errorf("got %q, want %q", q, expected)
	}
}

func TestPostgresAdapter_HasLimit_true(t *testing.T) {
	a, _ := executor.New("postgres")
	has, n, err := a.HasLimit("SELECT id FROM users LIMIT 50")
	if err != nil || !has || n != 50 {
		t.Errorf("expected has=true n=50, got has=%v n=%d err=%v", has, n, err)
	}
}

func TestPostgresAdapter_HasLimit_false(t *testing.T) {
	a, _ := executor.New("postgres")
	has, _, err := a.HasLimit("SELECT id FROM users")
	if err != nil || has {
		t.Errorf("expected has=false, got has=%v err=%v", has, err)
	}
}

func TestPostgresAdapter_InjectLimit_trailingSemicolon(t *testing.T) {
	a, _ := executor.New("postgres")
	q := a.InjectLimit("SELECT id FROM users;", 100, 0)
	expected := "SELECT id FROM users LIMIT 100 OFFSET 0"
	if q != expected {
		t.Errorf("got %q, want %q", q, expected)
	}
}
