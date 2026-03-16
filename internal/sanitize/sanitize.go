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
