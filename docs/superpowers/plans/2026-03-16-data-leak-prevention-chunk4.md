# Chunk 4: DSN env expansion + MaskingConfig

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `MaskingConfig` struct to `config.go`, expand `${VAR}` in DSN strings on load, and wire custom masking rules into `query.go`.

**Architecture:** Two tasks. Task 4.1 modifies `internal/config/config.go` only: adds `MaskingRuleConfig`/`MaskingConfig` types and DSN env expansion via `os.Expand`. Task 4.2 modifies `cmd/hq/commands/query.go` only: reads `cfg.Masking` custom rules, compiles them, appends to builtin rules, and passes the full slice to `StreamCSV` and `audit.New`.

**Tech Stack:** Go stdlib (`os`, `regexp`), `gopkg.in/yaml.v3`, existing `masking` package.

---

### Task 4.1: Add MaskingConfig and DSN env expansion to config.go

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests in `config_test.go`**

Add to the end of `internal/config/config_test.go`:

```go
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
```

Also add `"strings"` to the import block in `config_test.go`.

- [ ] **Step 2: Run tests — confirm new tests fail**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/config/... 2>&1 | head -20
```

Expected: compile error — `cfg.Masking` undefined and `strings` import error.

- [ ] **Step 3: Update `config.go` — add MaskingConfig + DSN expansion**

Replace the full content of `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaskingRuleConfig holds a single custom masking rule from config.yaml.
type MaskingRuleConfig struct {
	Name        string `yaml:"name"`
	Regex       string `yaml:"regex"`
	Replacement string `yaml:"replacement"`
}

// MaskingConfig holds custom masking rules from config.yaml.
type MaskingConfig struct {
	Rules []MaskingRuleConfig `yaml:"rules"`
}

type DBConfig struct {
	DSN     string `yaml:"dsn"`
	Dialect string `yaml:"dialect"`
}

type ExecutionConfig struct {
	MaxRows        int      `yaml:"max_rows"`
	TimeoutSeconds int      `yaml:"timeout_seconds"`
	AllowedSchemas []string `yaml:"allowed_schemas"`
}

type KnowledgeConfig struct {
	CacheTopN int `yaml:"cache_top_n"`
}

type Config struct {
	Databases map[string]DBConfig `yaml:"databases"`
	Execution ExecutionConfig     `yaml:"execution"`
	Knowledge KnowledgeConfig     `yaml:"knowledge"`
	Masking   *MaskingConfig      `yaml:"masking"`
}

// Load reads and parses the config file at path.
// DSN values containing ${VAR} or $VAR references are expanded via os.Expand.
// Returns an error if any referenced env var is unset.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read %s: %w", path, err)
	}
	cfg := &Config{
		Execution: ExecutionConfig{
			MaxRows:        1000,
			TimeoutSeconds: 30,
		},
		Knowledge: KnowledgeConfig{
			CacheTopN: 10,
		},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: invalid yaml: %w", err)
	}
	if err := expandDSNs(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// expandDSNs expands environment variable references in all DSN fields.
// Returns an error if any referenced variable is not set.
func expandDSNs(cfg *Config) error {
	expanded := make(map[string]DBConfig, len(cfg.Databases))
	for name, db := range cfg.Databases {
		var expandErr string
		dsn := os.Expand(db.DSN, func(varName string) string {
			val, ok := os.LookupEnv(varName)
			if !ok {
				expandErr = varName
				return ""
			}
			return val
		})
		if expandErr != "" {
			return fmt.Errorf("config: DSN for database %q references unset env var %s", name, expandErr)
		}
		expanded[name] = DBConfig{DSN: dsn, Dialect: db.Dialect}
	}
	cfg.Databases = expanded
	return nil
}

func (c *Config) DB(name string) (DBConfig, error) {
	db, ok := c.Databases[name]
	if !ok {
		return DBConfig{}, fmt.Errorf("db_not_found: no database configured with name %q", name)
	}
	return db, nil
}
```

- [ ] **Step 4: Run tests — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/config/... -v 2>&1 | tail -20
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add MaskingConfig and DSN env expansion to config"
```

---

### Task 4.2: Wire custom masking rules into query.go

**Files:**
- Modify: `cmd/hq/commands/query.go`

This task wires `cfg.Masking` custom rules (now available from Task 4.1) into the rules slice passed to `StreamCSV` and `audit.New`. Currently both calls use `masking.BuiltinRules()` directly — this task replaces those with a computed `rules` slice that includes custom rules.

There are no new unit tests for this task: the custom rule compilation is covered by `TestLoad_maskingConfig_customRules` in config, and the masking behaviour for custom rules is covered by `TestApply_customRuleAfterBuiltin` in the masking package. Integration correctness is verified by running the full test suite.

- [ ] **Step 1: Replace `masking.BuiltinRules()` calls in `query.go` with computed rules slice**

In `cmd/hq/commands/query.go`, add a helper function at the bottom of the file:

```go
// buildMaskingRules returns builtin rules + any custom rules from config.
// Returns an error if a custom rule has an invalid regex.
func buildMaskingRules(cfg *config.Config) ([]masking.Rule, error) {
	rules := masking.BuiltinRules()
	if cfg.Masking == nil {
		return rules, nil
	}
	for _, r := range cfg.Masking.Rules {
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			return nil, fmt.Errorf("masking rule %q has invalid regex: %w", r.Name, err)
		}
		rules = append(rules, masking.Rule{Name: r.Name, Re: re, Replacement: r.Replacement})
	}
	return rules, nil
}
```

- [ ] **Step 2: Use `buildMaskingRules` in the `RunE` function**

In the `RunE` function, after loading `cfg` and before `executor.StreamCSV`, insert:

```go
rules, err := buildMaskingRules(cfg)
if err != nil {
    return writeError("config_error", err.Error())
}
```

Find the line:
```go
result, err := executor.StreamCSV(ctx, db, finalSQL, includeHeader, os.Stdout, masking.BuiltinRules())
```

Replace with:
```go
result, err := executor.StreamCSV(ctx, db, finalSQL, includeHeader, os.Stdout, rules)
```

- [ ] **Step 3: Use `buildMaskingRules` in `logAudit`**

The `logAudit` function currently calls `masking.BuiltinRules()` directly. Update it to accept the rules slice as a parameter and use it:

Find:
```go
func logAudit(cfg *config.Config, dbName, status, errCode, sql string, rows int, durationMs int64) {
	dir, err := hqDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "# audit write failed: %v\n", err)
		return
	}
	path := filepath.Join(dir, "audit.log")
	logger := audit.New(path, masking.BuiltinRules())
```

Replace with:
```go
func logAudit(cfg *config.Config, dbName, status, errCode, sql string, rows int, durationMs int64, rules []masking.Rule) {
	dir, err := hqDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "# audit write failed: %v\n", err)
		return
	}
	path := filepath.Join(dir, "audit.log")
	logger := audit.New(path, rules)
```

Then update all call sites of `logAudit` in `query.go` to pass `rules` as the last argument. There are 4 call sites, all in `RunE`:

- Before `buildMaskingRules` can be called (policy check and limit exceeded), `rules` is not yet built. For these early-exit paths, pass `masking.BuiltinRules()` directly.
- After `buildMaskingRules` succeeds, pass `rules`.

Specifically:

```go
// Before buildMaskingRules (policy rejection):
logAudit(cfg, dbName, "rejected", err.Error(), sql, 0, 0, masking.BuiltinRules())

// Before buildMaskingRules (limit exceeded):
logAudit(cfg, dbName, "rejected", "limit_exceeded", sql, 0, 0, masking.BuiltinRules())

// After buildMaskingRules (timeout):
logAudit(cfg, dbName, "rejected", "timeout", sql, 0, 0, rules)

// After buildMaskingRules (success):
logAudit(cfg, dbName, "ok", "", sql, result.RowCount, durationMs, rules)
```

- [ ] **Step 4: Run all tests — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./... 2>&1 | tail -15
```

Expected: all tests PASS across all packages.

- [ ] **Step 5: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add cmd/hq/commands/query.go
git commit -m "feat: wire custom masking rules from config into StreamCSV and audit log"
```
