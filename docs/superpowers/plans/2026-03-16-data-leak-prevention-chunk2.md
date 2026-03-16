# Chunk 2: internal/sanitize package

**Spec ref:** Layer 6 — Prompt Injection Sanitization

**Files:**
- Create: `internal/sanitize/sanitize.go`
- Create: `internal/sanitize/sanitize_test.go`

---

### Task 2.1: Sanitize engine

- [ ] **Step 1: Write failing tests**

Create `internal/sanitize/sanitize_test.go`:

```go
// internal/sanitize/sanitize_test.go
package sanitize_test

import (
	"strings"
	"testing"

	"github.com/eduardoserete/humanized-query/internal/sanitize"
)

func TestApply_cleanValue_passthrough(t *testing.T) {
	got := sanitize.Apply("valor limpo 123")
	if got != "valor limpo 123" {
		t.Errorf("clean value should pass through unchanged, got %q", got)
	}
}

func TestApply_controlChars_stripped(t *testing.T) {
	// \x01 is a control char; \t \n \r are allowed
	input := "abc\x01def\x00ghi"
	got := sanitize.Apply(input)
	if strings.ContainsAny(got, "\x01\x00") {
		t.Errorf("control chars should be stripped, got %q", got)
	}
	if got != "abcdefghi" {
		t.Errorf("expected %q, got %q", "abcdefghi", got)
	}
}

func TestApply_tabNewlineCarriageReturn_preserved(t *testing.T) {
	input := "a\tb\nc\rd"
	got := sanitize.Apply(input)
	if got != input {
		t.Errorf("tab/newline/CR should be preserved, got %q", got)
	}
}

func TestApply_sqlCommentDashDash_replaced(t *testing.T) {
	got := sanitize.Apply("value -- comment here")
	if strings.Contains(got, "--") {
		t.Errorf("-- should be replaced, got %q", got)
	}
	if !strings.Contains(got, "[SQL-COMMENT]") {
		t.Errorf("expected [SQL-COMMENT] replacement, got %q", got)
	}
}

func TestApply_sqlCommentSlashStar_replaced(t *testing.T) {
	got := sanitize.Apply("value /* comment */")
	if strings.Contains(got, "/*") || strings.Contains(got, "*/") {
		t.Errorf("/* and */ should be replaced, got %q", got)
	}
}

func TestApply_injectionPhrase_fullRedaction(t *testing.T) {
	phrases := []string{
		"ignore previous instructions",
		"ignore all instructions",
		"disregard previous",
		"you are now",
		"new instructions:",
		"system prompt:",
	}
	for _, phrase := range phrases {
		got := sanitize.Apply("some data " + phrase + " more data")
		if got != "[REDACTED:injection-risk]" {
			t.Errorf("injection phrase %q should trigger full redaction, got %q", phrase, got)
		}
	}
}

func TestApply_injectionPhrase_caseInsensitive(t *testing.T) {
	got := sanitize.Apply("IGNORE PREVIOUS INSTRUCTIONS")
	if got != "[REDACTED:injection-risk]" {
		t.Errorf("injection detection should be case-insensitive, got %q", got)
	}
}

func TestApply_emptyString(t *testing.T) {
	got := sanitize.Apply("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests — confirm they fail**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/sanitize/...
```

Expected: `cannot find package` or compile error.

- [ ] **Step 3: Create `sanitize.go`**

```go
// internal/sanitize/sanitize.go
package sanitize

import (
	"strings"
	"unicode"
)

// injectionPhrases is a fixed, intentionally minimal list of English trigger phrases.
// Extend in future iterations as needed.
var injectionPhrases = []string{
	"ignore previous instructions",
	"ignore all instructions",
	"disregard previous",
	"you are now",
	"new instructions:",
	"system prompt:",
}

// Apply sanitizes a single cell value:
//  1. Strips control characters (except \t, \n, \r).
//  2. Replaces SQL comment tokens (-- /* */) with [SQL-COMMENT].
//  3. Fully redacts cells containing prompt injection phrases.
//
// Returns the sanitized value. Callers should check for [REDACTED:injection-risk]
// and emit a warning to stderr.
func Apply(value string) string {
	if value == "" {
		return value
	}

	// Step 1: check injection phrases before any modification
	lower := strings.ToLower(value)
	for _, phrase := range injectionPhrases {
		if strings.Contains(lower, phrase) {
			return "[REDACTED:injection-risk]"
		}
	}

	// Step 2: strip control characters (keep \t=0x09, \n=0x0A, \r=0x0D)
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if r == '\t' || r == '\n' || r == '\r' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	value = b.String()

	// Step 3: replace SQL comment tokens
	value = strings.ReplaceAll(value, "--", "[SQL-COMMENT]")
	value = strings.ReplaceAll(value, "/*", "[SQL-COMMENT]")
	value = strings.ReplaceAll(value, "*/", "[SQL-COMMENT]")

	return value
}
```

- [ ] **Step 4: Run tests — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/sanitize/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add internal/sanitize/
git commit -m "feat: add internal/sanitize package for prompt injection protection"
```
