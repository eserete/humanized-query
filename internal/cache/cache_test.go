package cache_test

import (
	"path/filepath"
	"testing"

	"github.com/eduardoserete/humanized-query/internal/cache"
)

func TestIncrement_createsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "table_usage.json")
	c := cache.New(path)

	err := c.Increment("mydb", []string{"users", "companies"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage, err := c.TopN("mydb", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage["users"] != 1 || usage["companies"] != 1 {
		t.Errorf("unexpected counts: %v", usage)
	}
}

func TestIncrement_accumulates(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, "table_usage.json"))

	c.Increment("mydb", []string{"users"})
	c.Increment("mydb", []string{"users"})
	c.Increment("mydb", []string{"companies"})

	usage, _ := c.TopN("mydb", 10)
	if usage["users"] != 2 {
		t.Errorf("expected users=2, got %d", usage["users"])
	}
	if usage["companies"] != 1 {
		t.Errorf("expected companies=1, got %d", usage["companies"])
	}
}

func TestTopN_returnsTopN(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, "table_usage.json"))

	c.Increment("mydb", []string{"a", "b", "c", "d"})
	c.Increment("mydb", []string{"a", "b", "c"})
	c.Increment("mydb", []string{"a", "b"})
	c.Increment("mydb", []string{"a"})

	usage, _ := c.TopN("mydb", 2)
	if len(usage) != 2 {
		t.Errorf("expected 2 results, got %d", len(usage))
	}
	if usage["a"] != 4 {
		t.Errorf("expected a=4, got %d", usage["a"])
	}
}

func TestTopN_missingDB_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, "table_usage.json"))
	usage, err := c.TopN("nonexistent", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(usage) != 0 {
		t.Errorf("expected empty, got %v", usage)
	}
}
