# Chunk 1: Purpose, Design Principles, and Architecture

### 1. Purpose

The system enables users to query databases using natural language while maintaining strict execution safety and operational discipline. The system separates semantic interpretation from database execution:

- The agent is responsible for understanding intent, generating queries, and explaining results.
- A dedicated backend (`hq`) written in Go enforces execution safety.

The system ensures:

- Safe database interaction (read-only by default)
- Human validation before execution
- Explainable queries
- Continuous learning from user feedback
- Compatibility with multiple database engines

### 2. Core Design Principles

The system follows engineering practices inspired by The Pragmatic Programmer and The Clean Coder.

- **Clarity** — All generated queries must be explainable. The agent shows what it understood, which tables it used, which joins it applied, and what assumptions it made.
- **Responsibility** — The LLM is never the enforcement boundary. Safety is enforced at the execution layer, not by the agent.
- **Feedback loops** — Ambiguities must be clarified with the user before execution. Errors must be explained and corrected collaboratively.
- **Continuous improvement** — Corrections and confirmations update the system's knowledge assets (glossary and mapping file). The system gets better with use.
- **Simplicity** — Prefer simple, observable mechanisms over opaque automation. The agent's reasoning must be visible to the user.
- **Professional discipline** — Queries must never execute automatically without explicit user confirmation.

### 3. High-Level Architecture

```
User
 │
 ▼
Natural Language Request
 │
 ▼
Agent
 │
 ├─ Business Glossary (Markdown)
 ├─ Database Mapping File (JSON)
 ├─ Schema Metadata (via hq schema)
 ├─ Intent Interpretation
 ├─ Query Generation
 └─ Explanation Builder
 │
 ▼
User Confirmation (required before execution)
 │
 ▼
hq (Safe Execution Service — Go)
 │
 ├─ Query Validation
 ├─ Policy Enforcement
 ├─ Database Adapter (PostgreSQL / MariaDB)
 ├─ Timeout Control
 └─ Result Streaming (CSV)
 │
 ▼
Agent
 │
 ├─ Result Interpretation
 ├─ Natural Language Summary
 └─ Result Presentation (prose / table / editor)
 │
 ▼
User
```

The agent is the cognitive layer: it interprets intent, generates queries, and communicates with the user. `hq` is the trust boundary: it validates, enforces policy, and executes against the database. Neither layer can substitute for the other — the agent cannot execute queries directly, and `hq` has no capacity for natural language understanding or contextual reasoning.
