package policy

import (
	"fmt"
	"regexp"
	"strings"
)

var forbiddenTokens = []string{
	"INSERT", "UPDATE", "DELETE", "DROP",
	"ALTER", "TRUNCATE", "CREATE", "EXEC", "CALL",
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

// schemaPrefixRe matches patterns like "schema_name." in queries.
var schemaPrefixRe = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*)\.`)

func checkSchemas(sql string, allowed []string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, s := range allowed {
		allowedSet[strings.ToLower(s)] = struct{}{}
	}
	matches := schemaPrefixRe.FindAllStringSubmatch(sql, -1)
	for _, m := range matches {
		schema := strings.ToLower(m[1])
		if _, ok := allowedSet[schema]; !ok {
			return fmt.Errorf("forbidden_schema: schema %q is not in the allowed list", m[1])
		}
	}
	return nil
}
