# Chunk 5: SQL Generation, Result Presentation, and Correction Loop

### SQL Generation

The agent generates SQL using the following knowledge hierarchy, applied in order of priority:

1. **Business Glossary** (`glossary.md`) — resolve domain terms before touching the schema
2. **Database Mapping File** (`mapping.json`) — use known relationships and confirmed joins
3. **Schema metadata** (`hq schema` output) — confirm column names, types, and declared foreign keys
4. **Naming conventions** — infer joins from `_id` suffix patterns when not present in the mapping file
5. **User clarification** — ask when confidence is low or the question is ambiguous

The agent must generate only `SELECT` queries. It must never generate statements containing any of the following keywords: `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `TRUNCATE`, `CREATE`, `EXEC`, `CALL`, or `REPLACE`.

The agent must never hallucinate column or table names. If the existence or spelling of a name is uncertain, the agent must run `hq schema` to verify before generating the query.

---

### Result Presentation Policy

After execution, the agent counts the number of data rows in the CSV output, excluding the header line, and selects the presentation format according to the following table:

| Data rows | Format | Behavior |
|---|---|---|
| 0 | Prose | Explain in natural language that no results were found. |
| 1–5 | Prose | Describe the values conversationally. Example: "There were 342 orders today, with an average ticket of $87.50." |
| 6–50 | Markdown table | Render the data as a Markdown table inline in the response. |
| 51+ | Temp file + editor | Save to `/tmp/hq-result-<unix-timestamp>.csv`. Open with `$EDITOR`. Fallback: `less`, then `more`. If none available, print the path and instruct the user to open it manually. |

**Pagination notice:** If `hq` reports `has_more=true` in stderr, the agent notifies the user that results were truncated and offers to fetch the next page using `--offset <N>`.

**Language rule:** The agent responds in the same language the user used in their question. SQL remains SQL. All natural language output — including prose, table summaries, error messages, and pagination notices — follows the user's language.

---

### Result Limit Override Policy

`hq` enforces a hard row limit configured in `~/.hq/config.yaml`. If a user requests a result larger than this limit, the agent proceeds as follows:

1. Generates the requested query as-is
2. Informs the user that the requested row count exceeds the configured policy (e.g., "The configured limit is 1,000 rows. Your query requests 50,000.")
3. Provides the full query so the user can run it directly in their own SQL IDE
4. Does not attempt to work around the limit or split results into multiple paginated requests unless the user explicitly asks for that behavior

---

### Correction Loop

When a generated query is incorrect, the agent initiates a structured correction loop.

**Step 1 — Ask what was wrong**

The agent asks the user, in their language:

> "What was incorrect in my interpretation? Was it the table, the join, the metric, or the time range?"

**Step 2 — Identify the root cause**

The agent classifies the error into one of the following categories:

- Wrong table mapping — update `glossary.md`
- Wrong join — update `mapping.json`
- Wrong metric definition — update `glossary.md`
- Wrong filter or time range — note for this session only; no persistent update needed
- SQL syntax error — self-correct; no knowledge update required

**Step 3 — Correct the query**

The agent generates a revised query and displays the full confirmation block again, including interpretation, tables, joins, assumptions, and the query itself. The agent requests user confirmation before re-executing.

**Step 4 — Update knowledge assets**

After the corrected query is confirmed and executed successfully:

- If the error was a table mapping: add or update the term in `glossary.md`
- If the error was a join: add or update the relationship in `mapping.json` with `confirmed_by_user: true`

**Maximum retries:** If the agent fails to generate a correct query after 2 correction attempts, it admits failure, displays the last generated SQL and the associated error, and asks the user for explicit guidance on how to proceed.

---

### Knowledge Evolution After Queries

After any successful query session where new knowledge was confirmed, the agent updates its knowledge assets as follows:

**Glossary updates:** Add new business metric definitions, concept-to-table mappings, and domain terminology confirmed by the user during the session.

**Mapping file updates:** Add confirmed join relationships, correct previously inferred relationships, and update column usage patterns based on the executed query.

The agent must never overwrite existing entries marked with `confirmed_by_user: true` unless the user explicitly requests a correction to that entry.
