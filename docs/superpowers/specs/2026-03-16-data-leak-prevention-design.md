# Design Spec: Data Leak Prevention

**Date:** 2026-03-16
**Status:** Approved
**Scope:** `hq` CLI — 6 security layers to prevent sensitive data exposure

---

## Problem

The `hq` CLI executes SQL queries against company databases containing sensitive PII (CPF, CNPJ, phone numbers), financial data, and client information. Several vectors exist where this data can leak:

1. Query results printed to terminal or saved to `/tmp` CSV files
2. SQL statements logged to `audit.log` (may contain literal values in WHERE clauses)
3. Knowledge files (`glossary.md`, `mapping.json`) stored without access restrictions
4. Database credentials stored in plaintext in `~/.hq/config.yaml`
5. Database connection opened with potentially over-privileged credentials
6. Malicious data in the database manipulating the LLM via prompt injection

---

## Goals

- Mask PII in all output paths (terminal, `/tmp` CSV, audit log)
- Never store credentials in plaintext
- Enforce least-privilege file permissions on all knowledge and log files
- Warn when database user has write permissions
- Detect and neutralize prompt injection payloads in query results

## Non-Goals

- Encryption of data in transit (handled by TLS at the DB driver level)
- Access control to the `hq` binary itself
- Central audit server or remote log shipping

---

## Architecture Overview

```
Database
    ↓
executor.StreamCSV  (reads rows)
    ↓
masking.Apply       (Layer 1: PII masking per cell)
    ↓
sanitize.Apply      (Layer 6: prompt injection sanitization per cell)
    ↓
csv.Writer          (writes to stdout)
    ↓
Terminal / /tmp file
```

All other layers (2–5) operate at setup/write time, not in the hot path.

---

## Layer 1: Output Masking (CSV / Terminal)

### Location
- `internal/masking/masking.go` — engine
- `internal/masking/patterns.go` — built-in patterns
- `internal/executor/executor.go` — integration point in `StreamCSV`

### Behavior

`StreamCSV` receives a `[]masking.Rule` slice. After scanning each row and before writing to the CSV writer, every cell value is passed through `masking.Apply(value, rules)`.

`masking.Apply` iterates rules in order and applies `regexp.ReplaceAllString` with the rule's replacement string.

### Built-in Patterns

| Name | Pattern | Replacement |
|---|---|---|
| `cpf` | `\d{3}\.?\d{3}\.?\d{3}-?\d{2}` | `***.***.***-**` |
| `cnpj` | `\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}` | `**.***.***/****-**` |
| `phone_br` | `(\+55\s?)?(\(?\d{2}\)?\s?)?\d{4,5}[-\s]?\d{4}` | `(**) *****-****` |
| `email` | `[^\s@]+@[^\s@]+\.[^\s@]+` | `***@***.***` |
| `credit_card` | `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b` | `**** **** **** ****` |
| `ipv4` | `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b` | `***.***.***.***` |

Built-in patterns are compiled once at program startup via `init()` and are always active.

### Custom Patterns (config.yaml)

```yaml
masking:
  rules:
    - name: internal_token
      regex: 'tok_[a-zA-Z0-9]{32}'
      replacement: 'tok_***'
```

Custom rules are appended after built-in rules. Invalid regex in a custom rule causes `hq` to exit with `config_error` on startup.

### Types

```go
// internal/masking/masking.go

type Rule struct {
    Name        string
    Re          *regexp.Regexp
    Replacement string
}

// Apply masks value against all rules in order.
func Apply(value string, rules []Rule) string

// BuiltinRules returns the compiled built-in rules.
func BuiltinRules() []Rule
```

### StreamCSV signature change

```go
// Before
func StreamCSV(ctx context.Context, db *sql.DB, query string, includeHeader bool, w io.Writer) (*Result, error)

// After
func StreamCSV(ctx context.Context, db *sql.DB, query string, includeHeader bool, w io.Writer, rules []masking.Rule) (*Result, error)
```

---

## Layer 2: Audit Log Masking

### Location
- `internal/audit/audit.go` — `Logger.Log` masks SQL before writing

### Behavior

Before writing the `sql=` field to `audit.log`, the SQL string is passed through `masking.Apply(sql, rules)` using the same rule set as Layer 1.

The `Logger` struct gains a `Rules []masking.Rule` field, set at construction time.

```go
// Before
func New(path string) *Logger

// After
func New(path string, rules []masking.Rule) *Logger
```

### Example

Input SQL: `SELECT * FROM users WHERE email = 'joao@empresa.com'`
Logged SQL: `SELECT * FROM users WHERE email = '***@***.***'`

---

## Layer 3: File Permissions (0600)

### Location
- `internal/audit/audit.go` — `Logger.Log` (already `0600`, no change needed)
- `internal/knowledge/knowledge.go` (new) — shared write helper with enforced `0600`

### Behavior

Any file written by `hq` under `~/.hq/` uses permissions `0600` (owner read/write only). This covers:

- `~/.hq/audit.log`
- `~/.hq/knowledge/glossary.md`
- `~/.hq/knowledge/mapping.json`
- `~/.hq/cache/table_usage.json`

A shared helper `knowledge.WriteFile(path string, data []byte) error` is introduced to centralise this enforcement. All existing write paths are migrated to use it.

Known write paths to migrate:
1. `internal/audit/audit.go` — `Logger.Log` (already `0600`, verify only)
2. `internal/cache/cache.go` — `Increment` writes `table_usage.json`
3. Agent-layer writes to `glossary.md` and `mapping.json` via whatever file write call is currently used

---

## Layer 4: DSN via Environment Variable

### Location
- `internal/config/config.go` — `Load` function

### Behavior

`config.Load` expands `${VAR}` and `$VAR` references in DSN strings using `os.Expand` before returning the config. No other fields are expanded (principle of least surprise).

```yaml
# ~/.hq/config.yaml
databases:
  prod:
    dsn: "${HQ_PROD_DSN}"
    dialect: postgres
```

If the referenced variable is not set, `hq` exits with:
```
{"error": "config_error", "detail": "DSN for database \"prod\" references unset env var HQ_PROD_DSN"}
```

Plaintext DSNs continue to work — this is additive, not breaking.

---

## Layer 5: Read-Only User Verification

### Location
- `internal/executor/adapter.go` — new `CheckReadOnly(db *sql.DB) (bool, error)` method on `Adapter` interface
- `internal/executor/postgres.go` and `internal/executor/mariadb.go` — implementations
- `cmd/hq/commands/query.go` — called after `adapter.Open`, warning emitted to stderr

### Behavior

After opening the connection and before executing the user query, `hq` runs a dialect-specific privilege check:

**Postgres:**
```sql
SELECT has_database_privilege(current_user, current_database(), 'CREATE')
```

**MySQL/MariaDB:**
```sql
SELECT COUNT(*) FROM information_schema.USER_PRIVILEGES
WHERE GRANTEE = CONCAT("'", SUBSTRING_INDEX(CURRENT_USER(), '@', 1), "'@'", SUBSTRING_INDEX(CURRENT_USER(), '@', -1), "'")
AND PRIVILEGE_TYPE IN ('INSERT','UPDATE','DELETE','DROP','ALTER','CREATE')
```

> Note: The `GRANTEE` construction via `CONCAT`/`SUBSTRING_INDEX` may behave unexpectedly for users with `%` as host wildcard. Tests must cover this edge case.

If write privileges are detected, `hq` emits a warning to stderr and continues:
```
# warning: database user has write permissions — a read-only user is strongly recommended
```

This is a warning, not a hard block. Blocking is an infrastructure decision outside `hq`'s scope.

---

## Layer 6: Prompt Injection Sanitization

### Location
- `internal/sanitize/sanitize.go` — engine
- `internal/executor/executor.go` — applied after masking, before csv.Write

### Behavior

After `masking.Apply`, each cell is passed through `sanitize.Apply(value)`.

**What is sanitized:**

1. **Control characters:** bytes `\x00`–`\x1f` (excluding `\t`, `\n`, `\r`) are stripped.
2. **SQL comment tokens:** occurrences of `--`, `/*`, `*/` are replaced with `[SQL-COMMENT]`.
3. **Prompt injection phrases:** a fixed, intentionally minimal list of case-insensitive trigger phrases (English-only; extensible in future iterations):
   - `ignore previous instructions`
   - `ignore all instructions`
   - `disregard previous`
   - `you are now`
   - `new instructions:`
   - `system prompt:`

   Cells containing any of these phrases are fully replaced with `[REDACTED:injection-risk]` and a warning is emitted to stderr:
   ```
   # warning: possible prompt injection detected in query results — cell redacted
   ```

**Design choice:** SQL comment tokens are sanitized in-place (partial replacement). Injection phrases trigger full cell redaction to avoid any partial leakage of the payload.

---

## Configuration Schema

Full `~/.hq/config.yaml` with all new fields:

```yaml
databases:
  prod:
    dsn: "${HQ_PROD_DSN}"
    dialect: postgres

execution:
  max_rows: 1000
  timeout_seconds: 30
  allowed_schemas:
    - public

masking:
  rules:
    - name: internal_token
      regex: 'tok_[a-zA-Z0-9]{32}'
      replacement: 'tok_***'

knowledge:
  cache_top_n: 10
```

The `masking` key is optional. If absent, only built-in rules apply.

---

## File Structure Changes

```
internal/
├── masking/
│   ├── masking.go        ← NEW: Apply(), BuiltinRules(), Rule type
│   ├── masking_test.go   ← NEW
│   └── patterns.go       ← NEW: compiled built-in regexp patterns
├── sanitize/
│   ├── sanitize.go       ← NEW: Apply(), control char + injection detection
│   └── sanitize_test.go  ← NEW
├── audit/
│   └── audit.go          ← MODIFIED: Logger gains Rules field, masks SQL
├── config/
│   └── config.go         ← MODIFIED: MaskingConfig added, DSN env expansion
├── executor/
│   ├── adapter.go        ← MODIFIED: CheckReadOnly added to Adapter interface
│   ├── executor.go       ← MODIFIED: StreamCSV gains rules param, calls masking+sanitize
│   ├── postgres.go       ← MODIFIED: CheckReadOnly implementation
│   └── mariadb.go        ← MODIFIED: CheckReadOnly implementation
└── knowledge/            ← NEW package (optional, for 0600 write helper)
    └── knowledge.go
```

---

## Testing Requirements

Each new package requires unit tests covering:

**masking:**
- Each built-in pattern matches and masks correctly
- Custom rules are applied after built-in rules
- Value with no PII passes through unchanged
- Empty string is handled safely

**sanitize:**
- Control characters are stripped
- SQL comment tokens are replaced
- Each injection phrase triggers full redaction + stderr warning
- Clean value passes through unchanged

**audit (modified):**
- SQL containing e-mail is masked in log output
- SQL with no PII is logged as-is

**config (modified):**
- `${VAR}` in DSN is expanded when env var is set
- Unset env var returns descriptive error
- Config without `masking` key loads with built-in rules only

**executor (modified):**
- `StreamCSV` with masking rules masks output
- `StreamCSV` with injection payload in results redacts cell

---

## Rollout Notes

- All changes are backward-compatible. No existing config files need to be updated.
- `StreamCSV` signature change requires updating the single call site in `cmd/hq/commands/query.go`.
- The `CheckReadOnly` warning can be disabled in future via config if it causes noise in certain environments.
