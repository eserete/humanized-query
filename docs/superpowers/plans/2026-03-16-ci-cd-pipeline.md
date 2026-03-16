# CI/CD Pipeline Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Criar um workflow GitHub Actions que roda testes em todo push/PR e publica binários compilados para 3 plataformas como GitHub Release em pushes de tag `v*`.

**Architecture:** Um único arquivo `.github/workflows/ci.yml` com dois jobs: `test` (roda em todo push/PR) e `release` (roda apenas em tags `v*`, depende de `test` passar). O job `release` faz cross-compile para `linux/amd64`, `darwin/amd64` e `darwin/arm64`, empacota cada binário em `.tar.gz` e publica como assets numa GitHub Release. O README é atualizado com instruções de instalação via `curl`.

**Tech Stack:** Go 1.25, GitHub Actions (`actions/checkout@v4`, `actions/setup-go@v5`, `softprops/action-gh-release@v2`), `tar`, `chmod`.

**Spec:** `docs/superpowers/specs/2026-03-16-ci-cd-design.md`

---

## Chunk 1: Workflow CI/CD

### Task 1: Criar estrutura de diretórios e o workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Criar o diretório `.github/workflows/`**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Criar o arquivo `.github/workflows/ci.yml`**

Conteúdo completo:

```yaml
name: CI

on:
  push:
    branches: ["**"]
    tags: ["v*"]
  pull_request:
    branches: ["**"]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Run tests
        run: go test ./... -v -race

  release:
    name: Release
    runs-on: ubuntu-latest
    needs: test
    if: startsWith(github.ref, 'refs/tags/v')
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Build binaries
        run: |
          set -e
          mkdir -p dist

          targets=(
            "linux/amd64"
            "darwin/amd64"
            "darwin/arm64"
          )

          for target in "${targets[@]}"; do
            GOOS="${target%/*}"
            GOARCH="${target#*/}"
            output="dist/hq-${GOOS}-${GOARCH}"
            echo "Building $GOOS/$GOARCH -> $output"
            GOOS=$GOOS GOARCH=$GOARCH go build \
              -ldflags="-s -w" \
              -o "$output" \
              ./cmd/hq
            tar -czf "dist/hq-${GOOS}-${GOARCH}.tar.gz" \
              -C dist "hq-${GOOS}-${GOARCH}"
            rm "$output"
          done

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*.tar.gz
          generate_release_notes: true
```

- [ ] **Step 3: Verificar que o YAML é válido**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))" && echo "YAML válido"
```

Esperado: `YAML válido`

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add test and release workflow"
```

---

## Chunk 2: README — Seção de Instalação

### Task 2: Atualizar o README com instruções de instalação via binário

**Files:**
- Modify: `README.md`

A seção `## Installation` atual instrui o usuário a instalar via `go install`, exigindo Go instalado. Substituir por instruções de download do binário pré-compilado como método principal, mantendo `go install` como alternativa para desenvolvedores.

- [ ] **Step 1: Substituir a seção `## Installation` no README**

Localizar e substituir o bloco atual:

```markdown
## Installation

Requires Go 1.22+.

```bash
go install github.com/eduardoserete/humanized-query/cmd/hq@latest
```
```

Pelo novo bloco:

```markdown
## Installation

### Download binary (recommended)

Download the pre-compiled binary for your platform from the [latest release](https://github.com/eduardoserete/humanized-query/releases/latest).

**macOS (Apple Silicon — arm64):**
```bash
curl -L https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-darwin-arm64.tar.gz | tar -xz
chmod +x hq
sudo mv hq /usr/local/bin/hq
```

**macOS (Intel — amd64):**
```bash
curl -L https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-darwin-amd64.tar.gz | tar -xz
chmod +x hq
sudo mv hq /usr/local/bin/hq
```

**Linux (amd64):**
```bash
curl -L https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-linux-amd64.tar.gz | tar -xz
chmod +x hq
sudo mv hq /usr/local/bin/hq
```

Verify:
```bash
hq --help
```

### Install from source

Requires Go 1.22+.

```bash
go install github.com/eduardoserete/humanized-query/cmd/hq@latest
```
```

- [ ] **Step 2: Verificar que o README não tem broken markdown**

```bash
python3 -c "
with open('README.md') as f:
    content = f.read()
assert '## Installation' in content
assert 'curl -L https://github.com/eduardoserete/humanized-query/releases/latest/download/hq-darwin-arm64.tar.gz' in content
assert 'hq --help' in content
print('README OK')
"
```

Esperado: `README OK`

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add binary installation instructions to README"
```

---

## Verificação Final

- [ ] **Confirmar que os dois arquivos foram criados/modificados**

```bash
git log --oneline -3
```

Esperado: dois commits recentes — `ci: add test and release workflow` e `docs: add binary installation instructions to README`.

- [ ] **Simular o build local para os 3 alvos (smoke test)**

```bash
mkdir -p /tmp/hq-smoke-test
for target in "linux/amd64" "darwin/amd64" "darwin/arm64"; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" \
    -o "/tmp/hq-smoke-test/hq-${GOOS}-${GOARCH}" \
    ./cmd/hq
  echo "OK: hq-${GOOS}-${GOARCH}"
done
rm -rf /tmp/hq-smoke-test
```

Esperado: três linhas `OK: hq-<plataforma>` sem erros.

- [ ] **Rodar os testes localmente**

```bash
go test ./... -v -race
```

Esperado: todos os testes passando (`PASS`).
