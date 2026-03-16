# Chunk 1: internal/masking package

**Spec ref:** Layer 1 — Output Masking

**Files:**
- Create: `internal/masking/patterns.go`
- Create: `internal/masking/masking.go`
- Create: `internal/masking/masking_test.go`

---

### Task 1.1: Built-in patterns

**Files:**
- Create: `internal/masking/patterns.go`

- [ ] **Step 1: Create `patterns.go` with compiled built-in rules**

```go
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
	{"phone_br", `(\+55\s?)?(\(?\d{2}\)?\s?)?\d{4,5}[-\s]?\d{4}`, `(**) *****-****`},
	{"email", `[^\s@]+@[^\s@]+\.[^\s@]+`, `***@***.***`},
	{"credit_card", `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`, `**** **** **** ****`},
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
```

- [ ] **Step 2: Verify file compiles**

```bash
cd /Users/eduardoserete/agents/humanized-query
go build ./internal/masking/...
```

Expected: no output (compile error would mean syntax issue — fix before continuing).

---

### Task 1.2: masking engine + Rule type

**Files:**
- Create: `internal/masking/masking.go`

- [ ] **Step 1: Write failing test first**

Create `internal/masking/masking_test.go`:

```go
// internal/masking/masking_test.go
package masking_test

import (
	"regexp"
	"testing"

	"github.com/eduardoserete/humanized-query/internal/masking"
)

func TestApply_cpf(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("cliente cpf 123.456.789-09 aqui", rules)
	want := "cliente cpf ***.***.***-** aqui"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_cnpj(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("empresa 12.345.678/0001-90 ok", rules)
	want := "empresa **.***.***/****-** ok"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_email(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("joao@empresa.com", rules)
	want := "***@***.***"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_creditCard(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("cartao 4111 1111 1111 1111 ok", rules)
	want := "cartao **** **** **** **** ok"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_ipv4(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("ip 192.168.1.100", rules)
	want := "ip ***.***.***.***"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_noMatch_passthrough(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("nenhum dado sensivel aqui", rules)
	want := "nenhum dado sensivel aqui"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_emptyString(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("", rules)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestApply_customRuleAfterBuiltin(t *testing.T) {
	builtin := masking.BuiltinRules()
	custom := masking.Rule{
		Name:        "token",
		Re:          regexp.MustCompile(`tok_[a-zA-Z0-9]{8}`),
		Replacement: "tok_***",
	}
	rules := append(builtin, custom)
	got := masking.Apply("tok_abcd1234", rules)
	want := "tok_***"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestApply_phoneBR(t *testing.T) {
	rules := masking.BuiltinRules()
	got := masking.Apply("fone (11) 99999-8888", rules)
	// phone_br replacement
	if got == "fone (11) 99999-8888" {
		t.Errorf("phone_br was not masked: %q", got)
	}
}
```

- [ ] **Step 2: Run tests — confirm they fail (package doesn't exist yet)**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/masking/...
```

Expected: `cannot find package` or `undefined: masking.Apply`

- [ ] **Step 3: Create `masking.go` with Rule type, Apply, BuiltinRules**

```go
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
```

- [ ] **Step 4: Run tests — confirm they pass**

```bash
cd /Users/eduardoserete/agents/humanized-query
go test ./internal/masking/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/eduardoserete/agents/humanized-query
git add internal/masking/
git commit -m "feat: add internal/masking package with built-in PII rules"
```
