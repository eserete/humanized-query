package schema_test

import (
	"testing"

	"github.com/eduardoserete/humanized-query/internal/schema"
)

func TestNewIntrospector_postgres(t *testing.T) {
	i, err := schema.New("postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i == nil {
		t.Error("expected non-nil introspector")
	}
}

func TestNewIntrospector_mariadb(t *testing.T) {
	i, err := schema.New("mariadb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i == nil {
		t.Error("expected non-nil introspector")
	}
}

func TestNewIntrospector_mysql(t *testing.T) {
	i, err := schema.New("mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i == nil {
		t.Error("expected non-nil introspector")
	}
}

func TestNewIntrospector_unsupported(t *testing.T) {
	_, err := schema.New("dynamodb")
	if err == nil {
		t.Error("expected error for unsupported dialect")
	}
}
