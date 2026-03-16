# Chunk 4: Execution Workflow, Query Confirmation, and Relationship Inference

### Execution Workflow

The agent follows this sequence for every query request:

**Step 1 — Discover databases**

```
hq db list
```

- If one database is available: use it automatically.
- If multiple databases are available and none was specified by the user: list them and ask which to use.

---

**Step 2 — Load knowledge assets**

```
Read ~/.hq/knowledge/glossary.md   (if exists)
Read ~/.hq/knowledge/mapping.json  (if exists)
```

- Apply glossary to resolve business terms present in the user's question.
- Use `mapping.json` to pre-populate known table relationships.

---

**Step 3 — Identify relevant tables**

- Use glossary and `mapping.json` to determine candidate tables.
- Use `~/.hq/cache/table_usage.json` to prioritize frequently queried tables.
- If candidate tables remain unclear after the above, ask the user one clarifying question before proceeding.

---

**Step 4 — Introspect schema**

```
hq schema --db <name> --table <table_name>   (repeated per relevant table)
```

- Do NOT load the full database schema. Load only tables that are plausibly relevant to the query.
- Use the returned schema to confirm column names, data types, primary keys, and foreign keys.

---

**Step 5 — Generate SQL**

- Apply the knowledge hierarchy in order: glossary → mapping → schema → naming conventions → user clarification.
- Generate a safe, correct `SELECT` query.
- Check for non-formal relationships before finalizing the query (see Relationship Inference section below).

---

**Step 6 — Show interpretation and request confirmation**

- Display the full confirmation block to the user (see Query Confirmation Policy below).
- Wait for explicit confirmation before proceeding.
- Do NOT execute the query if the user declines or requests changes.

---

**Step 7 — Execute**

```
hq query --db <name> --sql "..." --header [--offset N]
```

- Capture stdout (CSV rows) and stderr (metadata and errors).

---

**Step 8 — Count rows and present result**

- Count data rows in the CSV output, excluding the header line.
- Present results in the appropriate format (see Chunk 5 for result presentation rules).

---

**Step 9 — Update knowledge assets (if applicable)**

- If any new table mapping or relationship was confirmed during this session, update `glossary.md` and `mapping.json` accordingly.

---

### Query Confirmation Policy

Before executing any query, the agent must display all of the following sections and wait for explicit user confirmation.

**Confirmation block format:**

```
What I understood
-----------------
[Plain language description of the user's intent. Example: "You want the 10 users
with the highest number of logins during the current calendar month."]

Tables used
-----------
- login_events (aliased as le)
- users (aliased as u)

Join used
---------
- users.id = login_events.user_id

Assumptions
-----------
- "current month" means date_trunc('month', CURRENT_DATE)
- "logins" maps to login_events WHERE event_type = 'login'

Generated query
---------------
SELECT
  u.id,
  u.name,
  COUNT(*) AS login_count
FROM login_events le
JOIN users u ON u.id = le.user_id
WHERE date_trunc('month', le.created_at) = date_trunc('month', CURRENT_DATE)
GROUP BY u.id, u.name
ORDER BY login_count DESC
LIMIT 10;

Shall I execute this query? [yes/no]
```

**Confirmation rules:**

- The query executes only after explicit user confirmation.
- Accepted affirmatives include: `yes`, `sim`, `ok`, `y`, and equivalent affirmatives in the user's language.
- If the user declines:
  - Ask what needs to change.
  - Revise the query and display the full confirmation block again.
  - Do NOT execute until confirmed.

---

### Relationship Inference

Not all database relationships are declared as primary or foreign keys. The agent must infer relationships when they are not explicit in the schema.

**Inference signals, in priority order:**

1. Explicit FK declared in schema (`hq schema` output) — `type: explicit`, `confidence: high`
2. Field name ends in `_id` matching a known table name — `type: inferred`, `confidence: high`
3. Field name is semantically similar to a known table name — `type: inferred`, `confidence: medium`
4. Previous confirmed mapping present in `mapping.json` — use directly, trust fully

**Examples of high-confidence inference:**

```
company_id   → companies.id
user_id      → users.id
account_id   → accounts.id  (resolved via glossary to mechanic_shop_companies.id)
```

**Low-confidence threshold:**

If the inferred confidence level is `"low"`, the agent must ask the user before using the inferred join. Example prompt:

> "I'm not sure how `order_ref` relates to other tables. Does it reference `orders.id` or `order_lines.ref`?"

**After user confirmation:**

- Use the confirmed relationship in the current query.
- Write it to `mapping.json` with `confirmed_by_user: true`.
- Never ask the same question again in future queries.
