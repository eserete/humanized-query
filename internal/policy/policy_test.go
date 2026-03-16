package policy_test

import (
	"github.com/eduardoserete/humanized-query/internal/policy"
	"testing"
)

func TestCheck_allowsSelect(t *testing.T) {
	err := policy.Check("SELECT id, name FROM users LIMIT 10", nil)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheck_blocksForbiddenTokens(t *testing.T) {
	cases := []string{
		"UPDATE users SET name='x'",
		"DELETE FROM users",
		"INSERT INTO users VALUES (1)",
		"DROP TABLE users",
		"ALTER TABLE users ADD COLUMN x INT",
		"TRUNCATE users",
		"CREATE TABLE foo (id INT)",
		"EXEC sp_something",
		"CALL my_proc()",
		"select * from users; DELETE FROM users",
	}
	for _, sql := range cases {
		err := policy.Check(sql, nil)
		if err == nil {
			t.Errorf("expected error for query: %s", sql)
		}
	}
}

func TestCheck_allowsColumnWithForbiddenSubstring(t *testing.T) {
	cases := []string{
		"SELECT update_count FROM stats",
		"SELECT drop_rate FROM metrics",
		"SELECT created_at FROM users WHERE status = 'deleted'",
	}
	for _, sql := range cases {
		err := policy.Check(sql, nil)
		if err != nil {
			t.Errorf("expected no error for query %q, got: %v", sql, err)
		}
	}
}

func TestCheck_schemaRestriction(t *testing.T) {
	allowed := []string{"public"}
	err := policy.Check("SELECT * FROM public.users", allowed)
	if err != nil {
		t.Errorf("expected no error: %v", err)
	}
	err = policy.Check("SELECT * FROM secret.users", allowed)
	if err == nil {
		t.Error("expected error for disallowed schema")
	}
}

func TestCheck_schemaRestriction_empty(t *testing.T) {
	err := policy.Check("SELECT * FROM any_schema.users", nil)
	if err != nil {
		t.Errorf("expected no error when no schema restriction: %v", err)
	}
}
