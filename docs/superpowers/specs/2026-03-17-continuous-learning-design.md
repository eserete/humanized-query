# Spec: Continuous Learning & Performance — Agent Instructions

**Date:** 2026-03-17  
**Status:** Approved by user

---

## Problem

The agent currently has two inefficiencies:

1. **Redundant initialization on every query.** `hq db list` and reads of `glossary.md`/`mapping.json` run at the start of every query, even when the agent already has that context from earlier in the session.

2. **Passive knowledge updates.** Step 9 ("Update knowledge assets") is described as conditional and optional. In practice the agent rarely writes to the knowledge files because the triggers are undefined.

---

## Goals

- The agent learns automatically from every successful query and persists that knowledge.
- Initialization (db discovery + knowledge file reads) runs once per session, not once per query.
- No new CLI tools, no user friction, no blocking of responses.

---

## What Changes

### 1. Session-scoped initialization (performance)

Add a **Session Init** rule to the workflow:

> On the **first query of a session**, run `hq db list` and read `glossary.md` + `mapping.json`. Cache the results in the conversation context. On all subsequent queries in the same session, skip these steps and use the cached context.

This eliminates 3 tool calls per follow-up query.

### 2. Automatic knowledge update after every successful query

Replace the current vague Step 9 with four explicit, mandatory triggers:

| Trigger | What is saved | Where |
|---|---|---|
| `hq schema` was called for a table | Table name, columns list, and any FK relationships discovered | `mapping.json` |
| A JOIN was used in the generated query | Relationship between the two tables (source field, target field, type, confidence) | `mapping.json` |
| A business term was resolved to a DB concept | Term definition + resolved table/filter pattern | `glossary.md` |
| A query executed without error or correction | Query pattern with a plain-language description | `glossary.md` |

**Write rules:**
- Never overwrite an entry where `confirmed_by_user: true`.
- New entries default to `confirmed_by_user: false`.
- If an entry already exists and is not confirmed, update it silently.
- Writing happens **after presenting the result to the user** — never before, never blocking the response.

### 3. Glossary entry formats

**Business term / filter resolved:**
```markdown
## <term>
Definition: <plain-language description>
Query pattern: `<SQL fragment>`
Confirmed by user: false
```

**Reusable query pattern:**
```markdown
## <description>
Query pattern: `<full or partial SQL>`
Confirmed by user: false
```

### 4. mapping.json entry format (no change to existing format)

Tables discovered via `hq schema` are written using the existing format:

```json
{
  "description": "<table description inferred from name and columns>",
  "columns": ["col1", "col2", "..."],
  "relationships": [
    {
      "target_table": "<table>",
      "source_field": "<field>",
      "target_field": "id",
      "type": "explicit",
      "confidence": "high",
      "confirmed_by_user": false
    }
  ]
}
```

---

## What Does NOT Change

- Query confirmation policy (user must confirm before every execution).
- Security constraints (no write SQL, no config modification).
- Correction loop.
- Result presentation rules.
- Language detection rule.

---

## Scope of Changes

Only `AGENTS.md` is modified. No changes to Go source code, CLI, or knowledge file schemas.

---

## Success Criteria

- After a session with 3+ queries, `glossary.md` contains at least one new entry per resolved business term.
- After a session with 3+ queries, `mapping.json` contains entries for every table consulted via `hq schema`.
- Follow-up queries in the same session do not re-run `hq db list` or re-read the knowledge files.
