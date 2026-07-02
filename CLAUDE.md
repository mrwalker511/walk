# CLAUDE.md — Agent Context for walk

`walk` is a Go CLI that optimizes LLM token usage. It proxies, analyzes, compresses, and monitors LLM payloads before they leave the machine. Module: `github.com/mrwalker511/walk`. Go 1.25.0 (forced by `modernc.org/sqlite@v1.53.0`).

## Build & test

```bash
go build ./...
go test ./... -short
make lint          # golangci-lint (installed via `go install`, not the action)
```

## Package map

| Package | Purpose |
|---|---|
| `internal/config` | viper config loader, `ExpandVars` for `${VAR}` expansion |
| `internal/tokenizer` | token counting, `CountTokens` / `EstimateCost` with model+direction |
| `internal/scrubber` | secret/PII redaction, Shannon entropy scan |
| `internal/session` | SQLite session ledger, daily spend tracking, audit log |
| `internal/analyzer` | dead-weight and repetition fingerprint detection |
| `internal/cache` | prefix-cache analysis and reorder recommendations |
| `internal/compressor` | llama.cpp HTTP client for context summarization |
| `internal/router` | local-vs-cloud routing with llama.cpp health check |
| `cmd/` | cobra commands — one file per subcommand |

## Hard constraints

- **API keys are never stored** — config only holds `${ENV_VAR}` references; `ExpandVars` resolves them at load time via `os.Expand`.
- **Audit log stores SHA-256 hashes only** — never payload plaintext (`internal/session/session.go: AuditLog`).
- **golangci-lint must be installed via `go install`**, not the `golangci/golangci-lint-action` — the pre-built binary rejects Go 1.25.

## errcheck conventions

All unchecked returns must be silenced explicitly:
- Deferred closes: `defer func() { _ = x.Close() }()`
- Test HTTP handlers: `_ = json.NewEncoder(w).Encode(v)`, `_, _ = w.Write(...)`
- Production CSV/JSON writes: propagate the error, don't discard it

## Current implementation status

See `DEVLOG.md` for full history. Short version:

| Area | Status |
|---|---|
| `internal/config` — ExpandVars | ✅ done (PR #2) |
| `internal/tokenizer` — CountTokens / EstimateCost | ✅ done (PR #4) |
| lint / errcheck — all violations | ✅ done (PR #6) |
| `internal/scrubber` — extra test cases | ✅ done (PR #7) |
| `internal/session` — tokensCached bug + 3 tests | ✅ done (PR #8) |
| `CLAUDE.md` + `DEVLOG.md` — agent context + session log | ✅ done (PR #10) |
| doc sync + Go version fix in walk-spec.md | ✅ done (PR #11) |
| tokenizer: current Claude models + pricing placeholders | ✅ done (PR #12) |
| `internal/analyzer` — coverage gaps | pending |
| `internal/cache` — coverage gaps | pending |
| `docs/` — all doc files | pending |
| Integration tests (build tag `integration`) | pending |
| Coverage ≥ 80% per package | pending |

## What's left (future scope)

See the **Future Scope** section in `DEVLOG.md` for the ordered backlog.
