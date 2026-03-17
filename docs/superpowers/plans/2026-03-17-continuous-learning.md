# Continuous Learning & Performance Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `AGENTS.md` so the agent initializes once per session (not per query) and automatically persists knowledge after every successful query.

**Architecture:** Pure prompt engineering — only `AGENTS.md` is modified. No Go code changes. Two changes: (1) session-scoped init guard replaces per-query Steps 1–2, (2) Step 9 becomes a mandatory, rule-driven knowledge update with four explicit triggers.

**Tech Stack:** Markdown (AGENTS.md). No code, no dependencies.

---

## Chunk 1: Session-scoped initialization

### Task 1: Replace per-query Steps 1–2 with a session init guard

**Files:**
- Modify: `AGENTS.md` — Execution Workflow section, Steps 1 and 2

- [ ] **Step 1: Read the current Steps 1 and 2 in AGENTS.md**

Open `AGENTS.md` and locate the `## Execution Workflow` section. Read Steps 1 and 2 verbatim. Confirm they currently say "run `hq db list`" and "Read glossary.md and mapping.json" with no session guard.

- [ ] **Step 2: Replace Steps 1 and 2 with the session-init block**

Replace the current Steps 1 and 2 with the following text (preserve surrounding formatting):

```markdown
**Step 1 — Session init (run once per session, skip if already done)**

On the **first query of a session**, run all of the following. On subsequent queries in the same session, skip this step entirely and use the context already loaded.

```bash
hq db list
```
- If one database is configured: use it automatically.
- If multiple databases exist and the user has not specified one: list them and ask which to use.

Read `~/.hq/knowledge/glossary.md` (if it exists) and `~/.hq/knowledge/mapping.json` (if it exists). Apply the glossary to resolve business terms. Use `mapping.json` to pre-populate known table relationships.

**Step 2 — Identify relevant tables**
```

Note: old Step 2 ("Load knowledge assets") is absorbed into Step 1. Old Step 3 ("Identify relevant tables") becomes new Step 2. Renumber all subsequent steps accordingly (old Step 4 → new Step 3, etc.).

- [ ] **Step 3: Verify renumbering is correct**

After editing, confirm the workflow steps read: Step 1 (session init), Step 2 (identify tables), Step 3 (introspect schema), Step 4 (generate SQL), Step 5 (show confirmation), Step 6 (execute), Step 7 (present result), Step 8 (update knowledge assets).

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md
git commit -m "feat(agents): session-scoped init — skip db list and knowledge reads after first query"
```

---

## Chunk 2: Automatic knowledge update rules

### Task 2: Replace Step 9 with four explicit, mandatory triggers

**Files:**
- Modify: `AGENTS.md` — Execution Workflow section, last step (currently Step 9)

- [ ] **Step 1: Locate the current Step 9 in AGENTS.md**

Find the text: "**Step 9 — Update knowledge assets**" (or equivalent after renumbering from Chunk 1 it will be Step 8). Read it. Confirm it currently says something vague like "If any new table mapping or relationship was confirmed during this session, update glossary.md and/or mapping.json immediately."

- [ ] **Step 2: Replace with the four-trigger rule block**

Replace the content of this step with:

```markdown
**Step 8 — Update knowledge assets (mandatory after every successful query)**

After presenting the result to the user, apply all of the following rules. Do this silently — do not announce it to the user. Never block the response to do this; write knowledge files after the response is complete.

**Write rules (apply to all triggers below):**
- Never overwrite an entry where `confirmed_by_user: true`.
- New entries default to `confirmed_by_user: false`.
- If an entry already exists and is not user-confirmed, update it silently.

**Trigger 1 — `hq schema` was called for a table**
Write the table to `mapping.json` under the current database key. Include: `description` (inferred from table name and columns), `columns` (full list from schema output), and `relationships` (from FK fields in schema output, using `type: "explicit"` and `confidence: "high"`).

**Trigger 2 — A JOIN was used in the generated query**
Write the relationship to `mapping.json` in the source table's `relationships` array. Fields: `target_table`, `source_field`, `target_field`, `type` (`"explicit"` if from FK, `"inferred"` otherwise), `confidence` (`"high"` if from `_id` convention or FK, `"medium"` otherwise), `confirmed_by_user: false`.

**Trigger 3 — A business term was resolved to a database concept**
Write to `glossary.md` using this format:
```
## <term as the user wrote it>
Definition: <plain-language description>
Query pattern: `<SQL fragment that resolves this term>`
Confirmed by user: false
```
Examples of terms: "usuários da california", "oficinas ativas", "último mês".

**Trigger 4 — A query executed without error or correction**
Write a reusable pattern to `glossary.md` only if the query is generalizable (i.e., it uses joins across multiple tables, or filters on a business concept). Skip trivial or highly specific queries (e.g., `SELECT * FROM orders WHERE id = 123`). Use this format:
```
## <plain-language description of what the query does>
Query pattern: `<full or partial SQL with placeholders where appropriate>`
Confirmed by user: false
```
```

- [ ] **Step 3: Verify the section is self-contained and unambiguous**

Read the updated Step 8. Confirm:
- Each trigger has a clear condition ("when X happens") and a clear action ("write Y to Z").
- The write rules are stated once at the top and apply to all triggers.
- There is no vague language like "if relevant" or "when appropriate" (except the discretion note in Trigger 4).

- [ ] **Step 4: Commit**

```bash
git add AGENTS.md
git commit -m "feat(agents): mandatory four-trigger knowledge update after every successful query"
```

---

## Chunk 3: Backfill current session knowledge

### Task 3: Update knowledge files with what was learned in the current session

**Files:**
- Modify: `~/.hq/knowledge/glossary.md`
- Modify: `~/.hq/knowledge/mapping.json`

This task applies the new rules retroactively to what was discovered in the current session (the conversation in which this plan was created).

- [ ] **Step 1: Add tables discovered via `hq schema` to mapping.json**

The following tables were introspected via `hq schema` in this session and are not yet fully recorded in `mapping.json`:

- `mechanic_address` — columns: id, state, shop_id, type, address1, city, zipcode, address2, latitude, longitude, updated_at, country, default_address, time_zone_id. FKs: shop_id → mechanic_shops.id, state → usstates.id
- `usstates` — columns: id, abbr, name, salestax, taxable_shipping, country, two_letter_code. No FKs.
- `mechanic_shops` — columns: id, name, phone, ein, stripe_token, tax_free_status, reseller_state_id, reseller_permit_number, pickup_radius, epicor_buyer_id, catalog, test_stripe_token, allow_to_modify_suppliers, sales_rep_id, updated_at, cellphone, skip_address_validation, manager_email_cc, other_business_name, tires_pricing_id, logo, show_tire_retail_price, created_at, website, skip_onboard, tires_catalog, demo, tires_preferred_brands, new_labor, parts_preferred_brands, sms_part_type_as_part_name, advertising_segment, ex_shop, new_onboard, confirm_shipping, mandatory_po_number, time_zone_updated, custom_preferred_tire_brands. FKs: sales_rep_id → user.id, reseller_state_id → usstates.id
- `user_profile` — already in mapping.json; verify columns match schema output (id, user_id, first_name, last_name, phone, updated_at)

Add or update each of these in `mapping.json` under `databases.meu_db.tables`.

- [ ] **Step 2: Add business terms resolved in this session to glossary.md**

Add the following entries to `~/.hq/knowledge/glossary.md`:

```markdown
## usuários da california
Definition: Usuários cujas oficinas têm endereço registrado no estado CA (Califórnia).
Query pattern: `JOIN mechanic_shop_members msm ON msm.user_id = u.id JOIN mechanic_address ma ON ma.shop_id = msm.shop_id JOIN usstates s ON s.id = ma.state WHERE s.abbr = 'CA'`
Confirmed by user: false

## endereço de oficina / estado
Definition: O estado de uma oficina é armazenado em mechanic_address.state como FK para usstates.id. Usar usstates.abbr para filtrar por sigla (ex: 'CA').
Query pattern: `JOIN mechanic_address ma ON ma.shop_id = <shop_id_field> JOIN usstates s ON s.id = ma.state`
Confirmed by user: false

## usuários ativos (últimos 90 dias)
Definition: Usuários com last_login nos últimos 90 dias. Nota: a tabela user só registra o último login, não o histórico.
Query pattern: `SELECT ... FROM user u WHERE u.last_login >= NOW() - INTERVAL 90 DAY`
Confirmed by user: false
```

- [ ] **Step 3: Add reusable query patterns to glossary.md**

```markdown
## contar usuários por estado
Query pattern: `SELECT COUNT(DISTINCT u.id) AS total FROM user u JOIN mechanic_shop_members msm ON msm.user_id = u.id JOIN mechanic_address ma ON ma.shop_id = msm.shop_id JOIN usstates s ON s.id = ma.state WHERE s.abbr = ?`
Confirmed by user: false

## usuários com login recente com nome e email
Query pattern: `SELECT u.id, u.username, u.email, up.first_name, up.last_name, u.last_login FROM user u LEFT JOIN user_profile up ON up.user_id = u.id WHERE u.last_login >= NOW() - INTERVAL ? DAY ORDER BY u.last_login DESC`
Confirmed by user: false
```

- [ ] **Step 4: Commit**

```bash
git commit -m "chore(knowledge): backfill session knowledge — mechanic_address, usstates, CA filter patterns"
```

Note: the knowledge files (`~/.hq/knowledge/`) are outside the repo and are updated in place on disk. Only commit AGENTS.md separately if it was changed in this step (it won't be if Chunks 1 and 2 were already committed).
