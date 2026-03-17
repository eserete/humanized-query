# Agent Integration Documentation — Design Spec

## Goal

Expand the existing `## Integration with AI Agents` section in `README.md` to document how to configure any AI agent to use `hq` as its execution layer.

## Context

The `README.md` already documents:
- How to install `hq`
- How to configure `~/.hq/config.yaml`
- All CLI commands (`hq query`, `hq schema`, `hq db list`)

What is missing: how to configure an AI agent to become a conversational database analyst using `hq`. There is already a stub section (`## Integration with AI Agents`, lines 179–193) that mentions the concept but lacks concrete setup instructions.

## Design

### Approach

Expand the existing section in-place (Option C). No new files. No restructuring.

### Content to add

**Quick setup block:**
The central message is: copy the contents of `AGENTS.md` as the system prompt (or "custom instructions") of whatever AI agent you are using. One prompt, no plugins, no additional configuration.

**Knowledge files block:**
Explain `glossary.md` and `mapping.json` — optional files that teach the agent business terminology and table relationships over time. The agent writes to them; the user does not need to create them manually.

**How it works block:**
A short diagram showing the full flow: natural language question → agent generates SQL → `hq query` → CSV → agent interprets and responds.

**Table usage cache block:**
Explain `~/.hq/cache/table_usage.json` — written automatically after each successful query, used by the agent to prioritize the most relevant tables in its context.

### Tone

- Concise and direct. No marketing language.
- Code blocks for anything the user copies or runs.
- The "one prompt" fact should be prominent — it is the key differentiator.

## Out of Scope

- Agent-specific installation guides (OpenCode, Claude Desktop, etc.)
- Changes to `AGENTS.md`
- Changes to any Go code
