# Chunk 2: Roles, Responsibilities, and Supported Databases

### Agent Role and Responsibilities

The agent performs all cognitive tasks. It is the LLM layer (Claude Code, OpenCode, Copilot, or any compatible AI agent). It never executes queries directly.

Responsibilities:

- Interpret user requests in natural language
- Apply the business glossary to resolve domain terms
- Analyze schema metadata from `hq schema`
- Infer relationships between tables when not explicitly declared
- Generate database-specific SQL queries
- Show the user: what it understood, which tables and joins are used, what assumptions were made, and the generated query
- Request explicit user confirmation before any execution
- Interpret returned results
- Format and present the final output
- Update knowledge assets (glossary and mapping file) when corrections or new mappings are confirmed

The agent is never the security boundary.

### hq Execution Service Role and Responsibilities

`hq` is the trust boundary. It is a Go CLI binary that treats all incoming queries as untrusted input, regardless of source.

Responsibilities:

- Receive user-approved SQL queries via CLI flags
- Validate the database dialect and connection
- Enforce security constraints (forbidden statements: `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `TRUNCATE`, `CREATE`, `EXEC`, `CALL`, `REPLACE`)
- Enforce maximum result row limits (configurable via `~/.hq/config.yaml`)
- Enforce execution timeout
- Prevent destructive operations at the connection level (read-only session)
- Execute queries safely and stream results as CSV to stdout
- Return structured errors as JSON to stderr
- Write all executions to an append-only audit log

The `hq` service must enforce safety rules regardless of what the LLM generates. The Go service is the enforcement boundary.

### Separation of Concerns

| Concern | Owner |
|---|---|
| Natural language understanding | Agent |
| SQL generation | Agent |
| Business glossary resolution | Agent |
| Schema introspection | Agent (via `hq schema`) |
| Query explanation | Agent |
| User confirmation gate | Agent |
| Result interpretation | Agent |
| Forbidden statement enforcement | hq |
| Read-only connection enforcement | hq |
| Row limit enforcement | hq |
| Timeout enforcement | hq |
| Audit logging | hq |
| Schema allowlist enforcement | hq |

### Supported Databases

Current support:

- **PostgreSQL** — full support, read-only via session characteristics
- **MariaDB / MySQL** — full support, read-only via session transaction mode

Planned (not in current scope):

- **DynamoDB** — the agent would generate a structured execution plan; `hq` would convert it into DynamoDB API operations. This is out of scope for v1.
