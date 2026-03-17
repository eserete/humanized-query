# hq — humanized-query

A safe, read-only SQL execution CLI for PostgreSQL and MariaDB, designed to be called as a subprocess by external AI agents.

---

## What is hq?

`hq` is the execution layer between an AI agent and a database. The agent is responsible for understanding natural language, generating SQL, and interpreting results. `hq` is responsible for executing queries safely, enforcing security policies, and auditing every execution.

The agent is never the security boundary. `hq` is.

---

## Installation

### Download binary (recommended)

Download the pre-compiled binary for your platform from the [latest release](https://github.com/eduardoserete/humanized-query/releases/latest).

**macOS (Apple Silicon — arm64):**
```bash
curl -fL https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-darwin-arm64.tar.gz | tar -xz -C /tmp
chmod +x /tmp/hq
sudo mv /tmp/hq /usr/local/bin/hq
```

**macOS (Intel — amd64):**
```bash
curl -fL https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-darwin-amd64.tar.gz | tar -xz -C /tmp
chmod +x /tmp/hq
sudo mv /tmp/hq /usr/local/bin/hq
```

**Linux (amd64):**
```bash
curl -fL https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-linux-amd64.tar.gz | tar -xz -C /tmp
chmod +x /tmp/hq
sudo mv /tmp/hq /usr/local/bin/hq
```

If you don't have `sudo` access, move the binary to any directory in your `$PATH`, e.g. `mv /tmp/hq ~/.local/bin/hq`.

Verify:
```bash
hq --help
```

### Install from source

Requires Go 1.22+.

```bash
go install github.com/eduardoserete/humanized-query/cmd/hq@latest
```

---

## Configuration

Create `~/.hq/config.yaml`:

```yaml
databases:
  postgres_main:
    dsn: "postgres://user:pass@host:5432/mydb"
    dialect: postgres

  mariadb_main:
    dsn: "user:pass@tcp(host:3306)/mydb"
    dialect: mariadb

execution:
  max_rows: 1000         # default: 1000
  timeout_seconds: 30   # default: 30
  allowed_schemas: []   # empty = no restriction

knowledge:
  cache_top_n: 10       # most-used tables to prioritize in agent prompt
```

---

## Commands

### `hq query`

Execute a read-only SQL query. Streams results as CSV to stdout.

```bash
hq query --db postgres_main --sql "SELECT id, name FROM users LIMIT 10"
hq query --db postgres_main --sql "SELECT id, name FROM users LIMIT 10" --header
```

**stdout (with `--header`):**
```
id,name
1,Ana
2,Carlos
```

**stderr on success:**
```
# rows=2 duration_ms=45
```

**stderr on error:**
```json
{"error": "forbidden_statement", "detail": "UPDATE is not allowed"}
{"error": "limit_exceeded", "requested": 5000, "max_allowed": 1000}
{"error": "timeout", "detail": "query exceeded 30s limit"}
```

Use `--offset N` to paginate when `has_more=true` is returned on stderr.

---

### `hq schema`

Introspect the database schema and return structured JSON.

```bash
hq schema --db postgres_main
hq schema --db postgres_main --table users
```

**stdout:**
```json
{
  "database": "postgres_main",
  "dialect": "postgres",
  "tables": {
    "users": {
      "columns": [
        {"name": "id", "type": "integer", "nullable": false},
        {"name": "company_id", "type": "integer", "nullable": true}
      ],
      "primary_key": ["id"],
      "foreign_keys": [
        {"column": "company_id", "references_table": "companies", "references_column": "id"}
      ]
    }
  }
}
```

---

### `hq db list`

List configured database connections. Passwords are always masked.

```bash
hq db list
```

**stdout:**
```json
{
  "databases": [
    {"name": "postgres_main", "dialect": "postgres", "dsn": "postgres://user:***@host:5432/mydb"}
  ]
}
```

---

## Security

- **Read-only by design:** enforced at two independent layers — database connection level (`default_transaction_read_only=on` for PostgreSQL, `SET SESSION TRANSACTION READ ONLY` for MariaDB) and lexical token check before execution.
- **Blocked statements:** `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `TRUNCATE`, `CREATE`, `EXEC`, `CALL`, `REPLACE`.
- **Row limit:** configurable via `execution.max_rows`. Queries without `LIMIT` get one injected automatically. Queries explicitly requesting more than the limit are rejected.
- **Timeout:** configurable via `execution.timeout_seconds`. Cancelled via `context.WithTimeout`.
- **Schema restriction:** optional allowlist via `execution.allowed_schemas`.
- **Audit log:** every execution (including rejections) is appended to `~/.hq/audit.log`. Cannot be disabled.

---

## Integration with AI Agents

`hq` is the execution layer. An AI agent — any LLM-based assistant that can run shell commands — is the reasoning layer. The agent understands natural language, generates SQL, calls `hq`, and interprets the results. `hq` enforces safety.

### Quick setup

Paste the contents of [`AGENTS.md`](./AGENTS.md) as the system prompt (or "custom instructions") of whatever AI agent you are using. That's it — one prompt, no plugins, no additional configuration.

The `AGENTS.md` file contains a complete instruction set that teaches the agent:
- How to discover configured databases (`hq db list`)
- How to introspect schemas (`hq schema`)
- How to generate and confirm SQL before executing
- How to paginate and present results
- How to handle errors returned by `hq`

This works with any agent that supports a system prompt: OpenCode, Claude, ChatGPT custom instructions, Cursor rules, Copilot instructions, or any other LLM interface.

### Knowledge files (optional, built over time)

The agent reads two files from `~/.hq/knowledge/` at the start of each session:

```
~/.hq/knowledge/
├── glossary.md     ← business terms mapped to database concepts
└── mapping.json    ← table relationships, inferred joins, confirmed mappings
```

These files are written by the agent — not by you. Whenever you correct a query or confirm a table mapping, the agent updates them. Over time, the agent learns the language of your business and stops asking the same clarifying questions.

You do not need to create these files. The agent will create and update them automatically.

### How it works

```
User asks a question in natural language
  └── Agent reads glossary.md + mapping.json
  └── Agent calls: hq schema --db <name> --table <table>
  └── Agent generates SQL and shows it for confirmation
  └── User confirms
  └── Agent calls: hq query --db <name> --sql "..." --header
  └── hq streams CSV to stdout, metadata to stderr
  └── Agent interprets and responds in natural language
```

### Table usage cache

After each successful query, `hq` appends an entry to `~/.hq/cache/table_usage.json`. The agent uses this to prioritize the most frequently queried tables when deciding which schemas to load — keeping its context lean and relevant.

---

## Development

**Build:**
```bash
go build -o hq ./cmd/hq
```

**Test:**
```bash
go test ./...
```

**Package structure:**
```
cmd/hq/
  main.go              # wires subcommands, exits
  commands/            # query.go, schema.go, db.go

internal/
  config/              # reads ~/.hq/config.yaml
  policy/              # lexical token check
  executor/            # adapter interface, PostgreSQL, MariaDB, CSV streaming
  schema/              # schema introspection (PostgreSQL + MariaDB)
  audit/               # append-only audit.log writer
  cache/               # table_usage.json reader/writer
```

---

## License

MIT
