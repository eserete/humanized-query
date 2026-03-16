package policy

import (
	"fmt"
	"regexp"
	"strings"
)

var forbiddenTokens = []string{
	"INSERT", "UPDATE", "DELETE", "DROP",
	"ALTER", "TRUNCATE", "CREATE", "EXEC", "CALL", "REPLACE",
}

var forbiddenPatterns []*regexp.Regexp

func init() {
	for _, token := range forbiddenTokens {
		forbiddenPatterns = append(forbiddenPatterns,
			regexp.MustCompile(`(?i)\b`+token+`\b`))
	}
}

// Check validates a SQL query against security policies.
// allowedSchemas: if non-empty, only these schema prefixes are permitted.
// Returns a descriptive error if the query is rejected.
func Check(sql string, allowedSchemas []string) error {
	for i, re := range forbiddenPatterns {
		if re.MatchString(sql) {
			return fmt.Errorf("forbidden_statement: %s is not allowed", forbiddenTokens[i])
		}
	}
	if len(allowedSchemas) > 0 {
		if err := checkSchemas(sql, allowedSchemas); err != nil {
			return err
		}
	}
	return nil
}

// schemaPrefixRe matches schema.table patterns (two identifiers separated by a dot).
var schemaPrefixRe = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*)\.([a-z_][a-z0-9_]*)`)

// tableAliasRe matches table aliases: any word.word reference that follows a FROM/JOIN clause
// and has a trailing alias token (e.g. "public.users t" or "public.users AS t").
var tableAliasRe = regexp.MustCompile(`(?i)\b(?:from|join)\s+(?:[a-z_][a-z0-9_]*\.)?[a-z_][a-z0-9_]*\s+(?:as\s+)?([a-z_][a-z0-9_]*)`)

func checkSchemas(sql string, allowed []string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, s := range allowed {
		allowedSet[strings.ToLower(s)] = struct{}{}
	}

	// Collect table aliases so we don't treat them as schema names.
	aliasSet := make(map[string]struct{})
	for _, m := range tableAliasRe.FindAllStringSubmatch(sql, -1) {
		aliasSet[strings.ToLower(m[1])] = struct{}{}
	}

	matches := schemaPrefixRe.FindAllStringSubmatch(sql, -1)
	for _, m := range matches {
		schema := strings.ToLower(m[1])
		// Skip if the left-hand identifier is a known table alias.
		if _, isAlias := aliasSet[schema]; isAlias {
			continue
		}
		if _, ok := allowedSet[schema]; !ok {
			return fmt.Errorf("forbidden_schema: schema %q is not in the allowed list", m[1])
		}
	}
	return nil
}
