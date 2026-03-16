// internal/masking/masking.go
package masking

import "regexp"

// Rule is a compiled masking rule.
type Rule struct {
	Name        string
	Re          *regexp.Regexp
	Replacement string
}

// BuiltinRules returns the compiled built-in PII rules.
// The slice is a copy — safe to append custom rules.
func BuiltinRules() []Rule {
	out := make([]Rule, len(compiledBuiltins))
	copy(out, compiledBuiltins)
	return out
}

// Apply runs all rules against value in order and returns the masked result.
func Apply(value string, rules []Rule) string {
	for _, r := range rules {
		value = r.Re.ReplaceAllString(value, r.Replacement)
	}
	return value
}
