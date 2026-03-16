# Chunk 3: Knowledge Assets

### Overview of Knowledge Assets

The system maintains two persistent knowledge sources stored in `~/.hq/knowledge/`:

```
~/.hq/knowledge/
├── glossary.md      ← Markdown: semantic meaning of business terms
└── mapping.json     ← JSON: database structure, relationships, confirmed mappings
```

Both files are optional. The agent works without them but improves with use. Both evolve through interaction — corrections and confirmations update them automatically.

### Business Glossary (`glossary.md`)

Format: **Markdown** (human-readable and easily editable manually)

Purpose: maps business language to database concepts. Allows the agent to resolve domain-specific terms that don't match column or table names literally.

Content includes:
- Meaning of domain concepts
- Metric definitions
- Aliases used by users
- Confirmed table mappings

Example `glossary.md`:

```markdown
# Business Glossary

## accounts
Maps to table: `mechanic_shop_companies`
Confirmed by user: yes

## active users
Definition: users who have logged in at least once in the last 30 days.
Query pattern: `SELECT user_id FROM login_events WHERE created_at >= NOW() - INTERVAL '30 days' GROUP BY user_id`

## monthly logins
Metric: COUNT(*) from login_events WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE)

## ticket average
Metric: AVG(total_amount) FROM orders WHERE status = 'completed'
```

The glossary is updated whenever:
- The user confirms a new term-to-table mapping during ambiguity resolution
- The user corrects a generated query and identifies the source of the error as a terminology misunderstanding

### Database Mapping File (`mapping.json`)

Format: **JSON** (structured, machine-readable, updatable programmatically by the agent)

Purpose: stores database structure knowledge including table relationships, inferred joins, and user-confirmed mappings. Separate from the glossary because it captures structural database knowledge, not semantic business language.

Example `mapping.json`:

```json
{
  "databases": {
    "postgres_main": {
      "dialect": "postgres",
      "tables": {
        "users": {
          "description": "application users",
          "columns": ["id", "name", "email", "company_id", "created_at"],
          "relationships": [
            {
              "target_table": "companies",
              "source_field": "company_id",
              "target_field": "id",
              "type": "explicit",
              "confidence": "high",
              "confirmed_by_user": true
            }
          ]
        },
        "login_events": {
          "description": "user login audit trail",
          "columns": ["id", "user_id", "event_type", "created_at"],
          "relationships": [
            {
              "target_table": "users",
              "source_field": "user_id",
              "target_field": "id",
              "type": "inferred",
              "confidence": "high",
              "confirmed_by_user": false
            }
          ]
        }
      }
    }
  }
}
```

Field definitions:
- `type`: `"explicit"` (declared FK in schema) or `"inferred"` (detected by agent via naming conventions)
- `confidence`: `"high"`, `"medium"`, or `"low"`
- `confirmed_by_user`: boolean — whether the user has validated this relationship

The mapping file is updated whenever:
- The agent infers a new relationship and the user confirms it
- The user corrects a join used in a query
- A new table is introspected via `hq schema` and added to the agent's knowledge

### Knowledge Evolution

Both files grow over time through use. The agent must:
- Read both files at the start of each query session (if they exist)
- Write updates immediately after a user confirms a new mapping or corrects a query
- Never overwrite existing confirmed entries unless the user explicitly requests a correction
