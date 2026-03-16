# CI/CD Design — humanized-query (`hq`)

**Date:** 2026-03-16
**Status:** Approved

---

## Problem

The `hq` CLI binary has no automated test, build, or distribution pipeline. Binaries are compiled manually and not published anywhere.

## Goal

Add a GitHub Actions workflow that:
1. Runs tests on every push and pull request.
2. Compiles and publishes versioned binaries on every `v*` tag push.
3. Updates the README with clear installation instructions.

---

## Architecture

### Workflow file

Single file: `.github/workflows/ci.yml`

Two jobs:

```
push / pull_request (any branch)
  └── job: test

push with tag v*
  └── job: test
        └── job: release  (needs: test)
```

### Job: `test`

- **Trigger:** `push` and `pull_request` on any branch.
- **Runner:** `ubuntu-latest`
- **Steps:**
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` — `go-version-file: go.mod`, cache enabled
  3. `go test ./... -v -race`

### Job: `release`

- **Trigger:** `push` matching `tags: ['v*']` only.
- **Dependency:** `needs: test` — release is blocked if tests fail.
- **Runner:** `ubuntu-latest`
- **Permissions:** `contents: write` (required to create GitHub Releases)
- **Steps:**
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5`
  3. Cross-compile three targets:

| GOOS    | GOARCH | Output archive              |
|---------|--------|-----------------------------|
| `linux` | `amd64`| `hq-linux-amd64.tar.gz`     |
| `darwin`| `amd64`| `hq-darwin-amd64.tar.gz`    |
| `darwin`| `arm64`| `hq-darwin-arm64.tar.gz`    |

  Build flags: `go build -ldflags="-s -w" -o hq ./cmd/hq`
  Each binary is compressed into a `.tar.gz` named `hq-<GOOS>-<GOARCH>.tar.gz`.

  4. `softprops/action-gh-release@v2` — creates a GitHub Release named after the tag, attaches the three `.tar.gz` files as assets.

### Distribution

GitHub Releases only. Assets are downloadable from:
`https://github.com/eduardoserete/humanized-query/releases`

No external registries (npm, Homebrew, pkg.go.dev) in scope.

---

## README Changes

Add an **Installation** section with:
- Per-platform `curl` one-liners to download the latest release asset
- `chmod +x` and move to PATH instructions
- A minimal usage example (`hq --help`)

---

## Out of Scope

- Windows binaries
- Homebrew tap
- Changelog automation
- GoReleaser
- Code signing / notarization

---

## Files to Create / Modify

| File | Action |
|---|---|
| `.github/workflows/ci.yml` | Create |
| `README.md` | Modify — add Installation section |
