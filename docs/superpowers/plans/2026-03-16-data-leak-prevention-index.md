# Data Leak Prevention — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 6 security layers to the `hq` CLI to prevent sensitive company data from leaking via query results, logs, files, or prompt injection.

**Architecture:** A new `internal/masking` package provides regexp-based PII masking applied inside `StreamCSV` before writing CSV output. A new `internal/sanitize` package detects prompt injection payloads. Supporting layers harden file permissions, DSN handling, and DB privilege verification.

**Tech Stack:** Go stdlib (`regexp`, `os`, `os/expand`), go-sqlmock (tests), existing cobra/yaml stack.

**Chunks:**
- [Chunk 1](2026-03-16-data-leak-prevention-chunk1.md) — `internal/masking` package
- [Chunk 2](2026-03-16-data-leak-prevention-chunk2.md) — `internal/sanitize` package
- [Chunk 3](2026-03-16-data-leak-prevention-chunk3.md) — Integrate masking+sanitize into `StreamCSV` + audit log masking
- [Chunk 4](2026-03-16-data-leak-prevention-chunk4.md) — DSN env expansion + file permissions (0600)
- [Chunk 5](2026-03-16-data-leak-prevention-chunk5.md) — Read-only user verification (Postgres + MariaDB)
