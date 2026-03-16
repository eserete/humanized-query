# Design: Natural Language Database Query Agent

**Date:** 2026-03-16
**Status:** Approved

---

## Overview

`hq` is a safe, read-only SQL execution CLI with no embedded LLM. AI agents — Claude Code, OpenCode, GitHub Copilot, and others — serve as the LLM layer, owning all semantic reasoning and SQL generation. `hq` owns safety enforcement and execution, ensuring that no destructive operations can reach the database regardless of what the agent produces. The single deliverable of this design is an `AGENTS.md` file at the repository root that instructs any conforming AI agent to behave as a conversational data analyst on top of `hq`.

---

## Deliverable

| File | Description |
|---|---|
| `AGENTS.md` | Root-level instruction file that configures any AI agent to act as a conversational, read-only database analyst using `hq` |

---

## Spec Chunks

| Chunk | File | Contents |
|---|---|---|
| 1 | `2026-03-16-agent-instructions-design-chunk1.md` | Purpose, Design Principles, High-Level Architecture |
| 2 | `2026-03-16-agent-instructions-design-chunk2.md` | Roles & Responsibilities, Supported Databases |
| 3 | `2026-03-16-agent-instructions-design-chunk3.md` | Knowledge Assets (Glossary + Mapping File) |
| 4 | `2026-03-16-agent-instructions-design-chunk4.md` | Execution Workflow, Query Confirmation Policy, Relationship Inference |
| 5 | `2026-03-16-agent-instructions-design-chunk5.md` | SQL Generation, Result Presentation, Correction Loop, Limit Override |
| 6 | `2026-03-16-agent-instructions-design-chunk6.md` | Error Handling, Edge Cases, Security, Language Rule, Non-Goals |

---

## How to Read This Spec

Each chunk is self-contained and can be read independently. There is no required reading order. Start with Chunk 1 for foundational context, or go directly to the chunk most relevant to your area of concern. Cross-references between chunks are noted inline where relevant.
