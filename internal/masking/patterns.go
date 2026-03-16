// internal/masking/patterns.go
package masking

import "regexp"

var builtinDefs = []struct {
	name        string
	pattern     string
	replacement string
}{
	{"cpf", `\d{3}\.?\d{3}\.?\d{3}-?\d{2}`, `***.***.***-**`},
	{"cnpj", `\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}`, `**.***.***/****-**`},
	{"credit_card", `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, `**** **** **** ****`},
	{"phone_br", `(\+55\s?)?(\(?\d{2}\)?\s?)?\d{4,5}[-\s]?\d{4}`, `(**) *****-****`},
	{"email", `[^\s@]+@[^\s@]+\.[^\s@]+`, `***@***.***`},
	{"ipv4", `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`, `***.***.***.***`},
}

var compiledBuiltins []Rule

func init() {
	for _, d := range builtinDefs {
		compiledBuiltins = append(compiledBuiltins, Rule{
			Name:        d.name,
			Re:          regexp.MustCompile(d.pattern),
			Replacement: d.replacement,
		})
	}
}
