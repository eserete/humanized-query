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
