# Agent Integration Documentation — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand the `## Integration with AI Agents` section in `README.md` with concrete setup instructions for configuring any AI agent to use `hq`.

**Architecture:** Single file edit. Replace lines 179–193 in `README.md` with an expanded section. No new files, no code changes.

**Tech Stack:** Markdown only.

---

## Chunk 1: Edit README.md

**Files:**
- Modify: `README.md:179-193`

- [ ] **Step 1: Replace the existing `## Integration with AI Agents` section**

The current section (lines 179–193) reads:

```markdown
## Integration with AI Agents

`hq` is designed to be called as a subprocess by agents such as [OpenCode](https://opencode.ai). The agent reads knowledge files from `~/.hq/knowledge/`, generates SQL based on natural language input, calls `hq query`, and interprets the CSV output.

```
Agent
  ├── reads ~/.hq/knowledge/glossary.md     # business term → table mappings
  ├── reads ~/.hq/knowledge/mapping.json    # table relationships
  ├── calls hq query --db <name> --sql "..."
  └── interprets CSV from stdout
```

`hq` also tracks table usage in `~/.hq/cache/table_usage.json` after each successful query. Agents can use this to prioritize the most relevant table definitions in their prompt.

See [`AGENTS.md`](./AGENTS.md) for the full agent instruction set used with this project.
```

Replace with the expanded section below.

- [ ] **Step 2: Verify the README renders correctly**

Read the file and confirm the section looks correct, headings are at the right level, and code blocks are properly closed.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: expand agent integration section in README"
```

---

## Expanded section content

```markdown
## Integration with AI Agents

`hq` is the execution layer. An AI agent — any LLM-based assistant that can run shell commands — is the reasoning layer. The agent understands natural language, generates SQL, calls `hq`, and interprets the results. `hq` enforces safety.

### Quick setup

Paste the contents of [`AGENTS.md`](./AGENTS.md) as the system prompt (or "custom instructions") of whatever AI agent you are using. That's it — one prompt, no plugins, no additional configuration.

The `AGENTS.md` file contains a complete instruction set that teaches the agent:
- How to discover configured databases (`hq db list`)
- How to introspect schemas (`hq schema`)
- How to generate and confirm SQL before executing
- How to paginate and present results
- How to handle errors returned by `hq`

This works with any agent that supports a system prompt: OpenCode, Claude, ChatGPT custom instructions, Cursor rules, Copilot instructions, or any other LLM interface.

### Knowledge files (optional, built over time)

The agent reads two files from `~/.hq/knowledge/` at the start of each session:

```
~/.hq/knowledge/
├── glossary.md     ← business terms mapped to database concepts
└── mapping.json    ← table relationships, inferred joins, confirmed mappings
```

These files are written by the agent — not by you. Whenever you correct a query or confirm a table mapping, the agent updates them. Over time, the agent learns the language of your business and stops asking the same clarifying questions.

You do not need to create these files. The agent will create and update them automatically.

### How it works

```
User asks a question in natural language
  └── Agent reads glossary.md + mapping.json
  └── Agent calls: hq schema --db <name> --table <table>
  └── Agent generates SQL and shows it for confirmation
  └── User confirms
  └── Agent calls: hq query --db <name> --sql "..." --header
  └── hq streams CSV to stdout, metadata to stderr
  └── Agent interprets and responds in natural language
```

### Table usage cache

After each successful query, `hq` appends an entry to `~/.hq/cache/table_usage.json`. The agent uses this to prioritize the most frequently queried tables when deciding which schemas to load — keeping its context lean and relevant.
```
