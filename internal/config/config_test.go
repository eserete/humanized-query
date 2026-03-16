package config_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoad_dsnEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  prod:
    dsn: "${HQ_PROD_DSN_TEST}"
    dialect: postgres
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	t.Setenv("HQ_PROD_DSN_TEST", "postgres://ro:secret@host:5432/db")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Databases["prod"].DSN
	if got != "postgres://ro:secret@host:5432/db" {
		t.Errorf("expected expanded DSN, got %q", got)
	}
}

func TestLoad_dsnEnvExpansion_unsetVar_returnsError(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  prod:
    dsn: "${HQ_PROD_DSN_UNSET_XYZ}"
    dialect: postgres
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Ensure the variable is not set
	os.Unsetenv("HQ_PROD_DSN_UNSET_XYZ")

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for unset env var in DSN")
	}
	if !strings.Contains(err.Error(), "HQ_PROD_DSN_UNSET_XYZ") {
		t.Errorf("error should mention the unset variable name, got: %v", err)
	}
}

func TestLoad_dsnPlaintext_unchanged(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  prod:
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
	got := cfg.Databases["prod"].DSN
	want := "postgres://user:pass@host:5432/db"
	if got != want {
		t.Errorf("plaintext DSN should be unchanged, got %q", got)
	}
}

func TestLoad_maskingConfig_customRules(t *testing.T) {
	dir := t.TempDir()
	content := `
databases:
  mydb:
    dsn: "postgres://user:pass@host:5432/db"
    dialect: postgres
masking:
  rules:
    - name: internal_token
      regex: 'tok_[a-zA-Z0-9]{8}'
      replacement: 'tok_***'
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Masking == nil {
		t.Fatal("expected non-nil Masking config")
	}
	if len(cfg.Masking.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Masking.Rules))
	}
	r := cfg.Masking.Rules[0]
	if r.Name != "internal_token" {
		t.Errorf("expected name=internal_token, got %q", r.Name)
	}
	if r.Regex != `tok_[a-zA-Z0-9]{8}` {
		t.Errorf("unexpected regex: %q", r.Regex)
	}
	if r.Replacement != "tok_***" {
		t.Errorf("unexpected replacement: %q", r.Replacement)
	}
}

func TestLoad_maskingConfig_absent_nilMasking(t *testing.T) {
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
	if cfg.Masking != nil && len(cfg.Masking.Rules) != 0 {
		t.Errorf("expected nil or empty Masking when key absent, got %+v", cfg.Masking)
	}
}
