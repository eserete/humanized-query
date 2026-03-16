package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eduardoserete/humanized-query/internal/config"
)

func TestLoad_validConfig(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  mydb:
    dsn: "postgres://user:pass@host:5432/db"
    dialect: postgres
execution:
  max_rows: 500
  timeout_seconds: 10
  allowed_schemas: []
knowledge:
  cache_top_n: 5
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Databases["mydb"].DSN != "postgres://user:pass@host:5432/db" {
		t.Errorf("unexpected DSN: %s", cfg.Databases["mydb"].DSN)
	}
	if cfg.Execution.MaxRows != 500 {
		t.Errorf("unexpected max_rows: %d", cfg.Execution.MaxRows)
	}
	if cfg.Knowledge.CacheTopN != 5 {
		t.Errorf("unexpected cache_top_n: %d", cfg.Knowledge.CacheTopN)
	}
}

func TestLoad_missingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_defaults(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  mydb:
    dsn: "postgres://user:pass@host:5432/db"
    dialect: postgres
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Execution.MaxRows != 1000 {
		t.Errorf("expected default max_rows=1000, got %d", cfg.Execution.MaxRows)
	}
	if cfg.Execution.TimeoutSeconds != 30 {
		t.Errorf("expected default timeout=30, got %d", cfg.Execution.TimeoutSeconds)
	}
	if cfg.Knowledge.CacheTopN != 10 {
		t.Errorf("expected default cache_top_n=10, got %d", cfg.Knowledge.CacheTopN)
	}
}

func TestDBConfig_notFound(t *testing.T) {
	dir := t.TempDir()
	content := `databases: {}`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	_, err = cfg.DB("missing")
	if err == nil {
		t.Fatal("expected error for missing db")
	}
}
