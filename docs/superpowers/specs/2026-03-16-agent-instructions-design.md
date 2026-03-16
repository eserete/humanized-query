# Design: Agent Instructions for humanized-query

**Date:** 2026-03-16
**Status:** Approved

---

## Overview

`hq` is a safe, read-only SQL execution CLI designed to be called as a subprocess by AI agents. It has no LLM, no natural-language processing, and no REPL. The AI agent (Claude Code, OpenCode, Copilot, etc.) is the LLM — it owns semantic reasoning and SQL generation. `hq` owns safety enforcement and execution.

This design specifies a single `AGENTS.md` file at the repository root that instructs any compatible AI agent how to behave as a conversational data analyst using `hq`.

---

## Deliverable

A single file:

```
humanized-query/
└── AGENTS.md
```

No new code. No new binaries. No additional dependencies.

---

## Agent Role

The agent acts as a conversational data analyst. When the user asks a question in natural language, the agent:

1. Translates the question into SQL
2. Executes it safely via `hq`
3. Presents the result in the format most appropriate for the size of the response

---

## Execution Workflow

```
1. hq db list
   → Discover available databases.
   → If only one database is configured, use it automatically without asking.
   → If multiple databases exist and the user has not specified one, list them and ask.

2. hq schema --db <name> [--table <name>]
   → Introspect the relevant tables for the question.
   → Do not load the full schema. Load only tables that are plausibly relevant.
   → Use ~/.hq/cache/table_usage.json (top-N most queried tables) as a hint for relevance.

3. Read context files (if they exist):
   → ~/.hq/knowledge/glossary.md  — business terms mapped to columns/tables
   → ~/.hq/knowledge/mapping.md   — additional domain concept mappings

4. Generate SQL
   → Use the schema + knowledge files to produce a correct, safe SELECT query.
   → Always show the generated SQL to the user before executing.

5. hq query --db <name> --sql "..." --header
   → Execute the query. Capture stdout (CSV) and stderr (metadata/errors).

6. Count data rows in the CSV output (excluding the header line).

7. Present the result in the appropriate format (see Output Formatting).
```

---

## Output Formatting

The agent counts the number of data rows returned (CSV lines minus the header) and selects the output format:

| Data rows | Format | Behavior |
|---|---|---|
| 0 | Prose | Explain that no results were found for the question. |
| 1–5 | Prose | Describe the values in natural language. Example: "There were 342 orders today, with an average ticket of $87.50." |
| 6–50 | Markdown table | Render the data as a Markdown table inline in the response. |
| 51+ | Temp file + editor | Save to `/tmp/hq-result-<unix-timestamp>.csv`. Open with `$EDITOR`. If `$EDITOR` is not set, try `less`, then `more`. If neither is available, print the file path and instruct the user to open it manually. |

---

## Language Rule

The agent detects the language of the user's question and responds **entirely in that language** — including prose descriptions, error messages, pagination notices, and correction suggestions.

SQL queries remain in SQL (language-neutral).

---

## Context Optimization

To minimize token usage when building the SQL prompt:

- Load schema only for tables plausibly relevant to the question (use `--table` flag when possible).
- Use `~/.hq/cache/table_usage.json` to prioritize frequently queried tables.
- Read `~/.hq/knowledge/glossary.md` and `~/.hq/knowledge/mapping.md` to resolve business terms before schema lookup.

---

## Error Handling

**Errors from `hq` (JSON objects on stderr):**

| `error` value | Agent behavior |
|---|---|
| `forbidden_statement` | Explain the operation is not allowed (read-only). Do not attempt to work around this. |
| `limit_exceeded` | Warn that the query requested more rows than the configured maximum. Suggest adding filters to narrow the result. |
| `policy_violation` | Explain which schema or table is not permitted. |
| Connection error / timeout | Report the technical error. Suggest checking the database configuration in `~/.hq/config.yaml`. |
| SQL syntax error | Correct the query and retry. Maximum 2 retries before admitting failure. |

**Pagination (`has_more=true` in stderr):**

If `hq` reports `has_more=true`, the agent notifies the user that the results were truncated and offers to fetch the next page using `--offset`.

---

## Edge Cases

- **Ambiguous question:** If the question could map to multiple tables or interpretations, ask one clarifying question before executing.
- **Multiple databases, none specified:** List available databases (from `hq db list`) and ask the user which one to use.
- **SQL syntax error after 2 retries:** Admit failure, show the last generated SQL and the error, and ask the user for clarification.
- **Empty knowledge files:** Proceed without them. They are optional context.
- **`$EDITOR` not set, `less`/`more` unavailable:** Print the path of the temp file and tell the user to open it manually.

---

## Security Constraints

- The agent must never attempt to work around `hq`'s safety checks.
- The agent must never construct queries containing `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `TRUNCATE`, `CREATE`, `EXEC`, `CALL`, or `REPLACE`.
- The agent must never modify `~/.hq/config.yaml` or any knowledge/cache files on behalf of the user unless explicitly asked to do so in a separate, non-query interaction.

---

## Non-Goals

- No embedded LLM or API key configuration — the agent running `AGENTS.md` is already the LLM.
- No new Go code — this is purely an instruction document.
- No MCP server — direct CLI invocation is sufficient.
- No support for write operations — `hq` enforces read-only at the connection level.
