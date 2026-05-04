# Contributing to AgnosticOS

Thank you for your interest in contributing! This document covers the development workflow, conventions, and pull request process.

---

## 🧰 Development environment

### Prerequisites

- **Go** 1.22 or later
- **GNU Make**
- **git**
- **golangci-lint** (optional, for `make lint`)
- **xorriso**, **qemu-system-x86**, **ovmf** (for ISO testing)

### Clone & build

```bash
git clone https://github.com/ElioNeto/agnostikos.git
cd agnostikos
make deps
make build
./build/agnostic --help
```

### Run tests

```bash
make test                    # unit tests with race detector
go test -v -race ./...       # same, verbose
```

### Lint

```bash
make lint                    # golangci-lint
make fmt                     # go fmt
```

---

## 📝 Commit conventions

We follow **Conventional Commits** strictly. Every commit message must use one of the following prefixes:

| Prefix      | Purpose                                    |
|-------------|--------------------------------------------|
| `feat:`     | A new feature                              |
| `fix:`      | A bug fix                                  |
| `docs:`     | Documentation-only changes                 |
| `refactor:` | Code change that neither fixes nor adds    |
| `test:`     | Adding or fixing tests                     |
| `chore:`    | Build, CI, dependencies, tooling           |

**Format:**
```
<type>(<scope>): <imperative description>

[optional body]
```

**Examples:**
```
feat(manager): add flatpak backend
fix(config): handle missing locale field gracefully
docs(readme): update quick start example
chore(deps): bump cobra to v1.9
```

Commit messages must be in **English**, present tense, imperative mood.

---

## 🔀 Pull request process

1. **Create a branch** from `main` using the naming convention:
   - `feat/<slug>` — new feature
   - `fix/<slug>` — bug fix
   - `docs/<slug>` — documentation
   - `chore/<slug>` — tooling / dependencies

2. **Make your changes** — keep commits atomic (one responsibility per commit).

3. **Run the full validation suite** before pushing:
   ```bash
   make deps
   make build
   make test
   make bootstrap
   make iso
   make test-iso
   ```

4. **Push and open a PR** against `main`.

5. **PR description must include:**
   - What was changed and why
   - How to test the changes
   - Any relevant issue references (e.g. `Closes #15`)

6. **CI checks** must pass before merge. The pipeline includes:
   - `go build ./...`
   - `go test ./... -race`
   - `go vet ./...`
   - `golangci-lint run`

7. **A maintainer will review** your PR. Address review feedback with additional commits (no rebasing during review).

---

## ✅ PR checklist

Before submitting your PR, confirm:

- [ ] Code builds (`make build`)
- [ ] All tests pass (`make test`)
- [ ] Linter is clean (`make lint`)
- [ ] Commits follow Conventional Commits
- [ ] No debugging leftovers (`console.log`, `print`, etc.)
- [ ] No secrets or credentials in code
- [ ] New code includes tests (when applicable)
- [ ] Documentation updated (README, docs/, etc.) if public API changed

---

## 🧪 Code style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Names in lowercase, no underscores for packages
- Interfaces named with `-er` suffix (`Reader`, `Writer`, `Handler`)
- Errors must always be handled; never use `_` for error returns
- `context.Context` is always the first parameter in functions that need it
- Table-driven tests for multiple test cases

---

## 🐛 Reporting issues

- Use the GitHub issue tracker
- Include:
  - Go version (`go version`)
  - OS / distribution
  - Steps to reproduce
  - Expected vs actual behavior
  - Relevant logs or error output

---

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License (see [LICENSE](LICENSE)).
