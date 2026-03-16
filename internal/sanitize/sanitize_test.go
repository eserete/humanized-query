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
