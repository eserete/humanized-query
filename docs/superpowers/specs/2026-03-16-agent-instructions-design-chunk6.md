# Chunk 6: Error Handling, Edge Cases, Security, and Non-Goals

### Error Handling

**Errors from `hq` (returned as JSON on stderr):**

| `error` value | Agent behavior |
|---|---|
| `forbidden_statement` | Explain to the user that the operation is not allowed (the system is read-only). Do not attempt to rephrase or work around the restriction. |
| `limit_exceeded` | Warn that the query requested more rows than the configured maximum. Provide the generated query for the user to run in their own IDE. Do not retry with a lower limit without explicit user request. |
| `policy_violation` | Explain which schema or table is not permitted by the current configuration. |
| Connection error | Report the technical error. Suggest verifying the database configuration in `~/.hq/config.yaml`. |
| Timeout | Report that the query timed out. Suggest adding filters to reduce execution time. |
| SQL syntax error | Correct the query internally and retry. Maximum 2 retries. After 2 failures, show the last SQL and error, and ask the user for guidance. |

**Pagination (`has_more=true` in stderr metadata):**

If `hq` reports `has_more=true`, the agent notifies the user that results were truncated to the configured row limit and offers to fetch the next page:

> "Results were truncated at N rows. Would you like me to fetch the next page?"

If yes, re-run with `hq query --db <name> --sql "..." --header --offset <next_offset>`.

---

### Edge Cases

- **Ambiguous question:** If the question could map to multiple tables or interpretations, ask one focused clarifying question before generating SQL. Do not guess.
- **Multiple databases, none specified:** Run `hq db list`, display the available databases, and ask which one to use. If only one is configured, use it automatically without asking.
- **No relevant tables found in knowledge assets:** Load the full `hq schema --db <name>` output and use naming conventions and context to infer relevance. If still uncertain, ask the user.
- **Empty or missing knowledge files:** Proceed without them. They are optional. The agent falls back to schema introspection alone.
- **SQL syntax error after 2 retries:** Admit failure. Show the last generated SQL and the error message. Ask the user for clarification before trying again.
- **`$EDITOR` not set:** Try `less`, then `more`. If neither is available, print the absolute path of the temp CSV file and instruct the user to open it manually.
- **User declines confirmation:** Do not execute. Ask what needs to change, revise the query, and show the confirmation block again.

---

### Security Constraints

The agent must respect and reinforce `hq`'s safety model:

- Never attempt to work around `hq`'s policy checks (forbidden statements, schema allowlist, row limits)
- Never construct queries containing: `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `TRUNCATE`, `CREATE`, `EXEC`, `CALL`, `REPLACE`
- Never modify `~/.hq/config.yaml` on behalf of the user during a query interaction
- Never modify knowledge files (`glossary.md`, `mapping.json`) without a clear user-initiated reason (correction or new confirmation)
- Never execute a query without explicit user confirmation — even if the query looks obviously safe

The LLM is never the security boundary. `hq` is.

---

### Language Rule

The agent detects the language of the user's question and responds entirely in that language — including:

- Prose result descriptions
- Error messages and explanations
- Pagination and truncation notices
- Clarifying questions
- Correction loop prompts
- Confirmation prompts

SQL queries remain in SQL (language-neutral). Column and table names are never translated.

---

### Non-Goals

- **No embedded LLM** — the AI agent running this AGENTS.md is already the LLM. No API keys or model configuration needed.
- **No new Go code** — the deliverable is a documentation file, not a code change.
- **No MCP server** — direct CLI invocation via `hq` subcommands is sufficient.
- **No write operations** — `hq` enforces read-only at the database connection level. This is not configurable by the agent.
- **No DynamoDB support in v1** — planned for future iterations.
- **No automatic query execution** — the agent must never skip the confirmation step, even for trivially simple queries.
