# walk-spec.md

# Project Specification — `walk`

> _"Slow down. Use less. Save more."_

---

## Overview

`walk` is a Go-based CLI tool that optimizes LLM token usage across agentic coding workflows.
It acts as an intelligent proxy layer — analyzing, compressing, and monitoring LLM payloads
before, during, and after sessions with tools like Claude Code, OpenAI Codex, and local
llama.cpp inference servers.

**Philosophy:** Most LLM waste comes from moving too fast — bloated prompts, stale context,
repeated instructions, and unmonitored session drift. `walk` enforces discipline at every layer.

**Target User:** AI engineers, developers, and enterprise teams who use LLMs heavily and want
full visibility and control over token spend without sacrificing output quality.

---

## Core Identity

| Property        | Value                                        |
| --------------- | -------------------------------------------- |
| Binary name     | `walk`                                       |
| Language        | Go 1.25.0 (pinned by modernc.org/sqlite)     |
| CLI framework   | cobra + viper                                |
| Config location | `~/.walk/config.yaml`                        |
| DB              | SQLite (`~/.walk/sessions.db`)               |
| Local LLM       | llama.cpp via `http://localhost:8080/v1`     |
| Primary targets | macOS ARM64 (M-series), Linux amd64          |
| Test framework  | `testing` (stdlib) + `testify`               |
| Distribution    | Single binary, Homebrew tap, GitHub Releases |

---

## Command Reference

### `walk init`

Interactive setup wizard. Detects llama.cpp server, sets API keys, configures budget.

- Writes `~/.walk/config.yaml`
- Validates llama.cpp health endpoint
- Prompts for Anthropic / OpenAI API keys (stored as env var references, never plaintext)
- Sets daily budget cap and warning threshold

### `walk analyze <file|stdin>`

Inspect a prompt or payload before sending it to an LLM.

- Token count (model-aware: GPT-4o, Claude Sonnet, Gemma, etc.)
- Estimated cost (input + output projection)
- Dead weight detection (redundant phrases, repeated instructions)
- Repetition fingerprinting (duplicate blocks across context)
- Secret/PII scan (API keys, JWTs, emails, SSNs)
- Outputs: token count, cost estimate, warnings, optimization suggestions

### `walk compress [--local] [--model <model>]`

Summarize or compress content using llama.cpp before sending to a cloud LLM.

- Pipe mode: `cat context.md | walk compress`
- File mode: `walk compress --file prompt.md`
- Routes to llama.cpp `/v1/chat/completions` by default
- Shows compression ratio and token delta

### `walk diff <original> <optimized>`

Side-by-side token comparison between two payload versions.

- Token delta (saved / added)
- Cost delta at specified model pricing
- Highlights removed sections

### `walk watch [--tool <claude-code|codex>]`

Live session monitor. Attaches to a running Claude Code or Codex session.

- Real-time token burn rate
- Budget cap enforcement (warn at threshold, hard-stop at cap)
- Burn rate projection ("at this rate: $X total")
- Context rot alerts (staleness scoring on active context)
- "Lost in the middle" warnings when context exceeds 60% fill

### `walk report [--session <id|last|all>] [--format <table|json|csv>]`

Post-session cost breakdown and savings summary.

- Tokens used: input / output / cached
- Cache hit ratio and estimated cache savings
- Cost attribution by tag or project
- Cumulative savings vs. unoptimized baseline

### `walk cache analyze <file>`

Analyze a prompt for prefix cache optimization.

- Identify which sections are stable (cache-friendly) vs. dynamic
- Recommend reordering: stable system prompt → dynamic user content
- Estimate cache hit savings at Anthropic / OpenAI pricing tiers

### `walk scrub <file|stdin>`

Scan outbound payload for secrets and PII before it leaves your machine.

- Patterns: API keys, JWTs, AWS credentials, SSH keys, emails, SSNs, phone numbers
- Entropy analysis for high-randomness strings
- Outputs: clean payload + redaction report
- Exit code 1 if secrets found (CI/CD friendly)

### `walk budget [--set <amount>] [--reset] [--status]`

Manage daily/session spend limits.

- `walk budget --set 5.00` — set $5.00 daily cap
- `walk budget --status` — show today's spend vs. cap
- `walk budget --reset` — reset daily counter

---

## Package Architecture

```
walk/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go          # cobra root, global flags
│   ├── init.go          # walk init wizard
│   ├── analyze.go       # walk analyze
│   ├── compress.go      # walk compress
│   ├── diff.go          # walk diff
│   ├── watch.go         # walk watch
│   ├── report.go        # walk report
│   ├── cache.go         # walk cache
│   ├── scrub.go         # walk scrub
│   └── budget.go        # walk budget
├── internal/
│   ├── tokenizer/
│   │   ├── tokenizer.go         # token counting, model-aware pricing
│   │   └── tokenizer_test.go
│   ├── scrubber/
│   │   ├── scrubber.go          # secret + PII detection
│   │   └── scrubber_test.go
│   ├── compressor/
│   │   ├── compressor.go        # llama.cpp HTTP client, summarization
│   │   └── compressor_test.go
│   ├── analyzer/
│   │   ├── analyzer.go          # dead weight, repetition fingerprint
│   │   └── analyzer_test.go
│   ├── cache/
│   │   ├── cache.go             # prefix cache analysis, reorder logic
│   │   └── cache_test.go
│   ├── session/
│   │   ├── session.go           # SQLite ledger, budget tracking
│   │   └── session_test.go
│   ├── router/
│   │   ├── router.go            # model routing logic (local vs cloud)
│   │   └── router_test.go
│   └── config/
│       ├── config.go            # viper config loader + defaults
│       └── config_test.go
├── docs/
│   ├── getting-started.md
│   ├── configuration.md
│   ├── commands.md
│   ├── troubleshooting.md
│   ├── security.md
│   └── examples/
│       ├── claude-code.md
│       ├── codex.md
│       └── llama-cpp.md
├── testdata/
│   ├── sample_prompt.md
│   ├── dirty_payload.txt        # payload with fake secrets for scrubber tests
│   └── bloated_context.md
├── .github/
│   └── workflows/
│       ├── ci.yml               # test + lint on push
│       └── release.yml          # goreleaser on tag
├── .goreleaser.yaml
├── Makefile
└── README.md
```

---

## Configuration Schema (`~/.walk/config.yaml`)

```yaml
walk:
  version: "1"

local_model:
  provider: llama.cpp
  endpoint: http://localhost:8080/v1
  model: gemma-4-27b-q8_0 # or gpt-4o-mini-gguf, etc.
  timeout_seconds: 30
  enabled: true

providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY} # env var reference — never hardcoded
    default_model: claude-sonnet-4-5
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o

budget:
  daily_limit: 10.00
  session_limit: 2.00
  warn_at_percent: 80
  hard_stop: true # kills command if limit exceeded

scrubber:
  enabled: true
  block_on_detect: true # exit 1 if secrets found
  patterns:
    - api_key
    - jwt
    - aws_credential
    - ssh_key
    - email
    - ssn
    - phone
  entropy_threshold: 4.5

cache:
  track_hits: true
  optimize_on_analyze: true

session:
  db_path: ~/.walk/sessions.db
  audit_log: ~/.walk/audit.log
  audit_enabled: true

output:
  color: true
  show_savings_line: true # "💾 Saved X tokens (~$Y)"
  default_format: table
```

---

## Pricing Reference (used by tokenizer package)

| Model             | Input (per 1M) | Output (per 1M) | Cached Input |
| ----------------- | -------------- | --------------- | ------------ |
| claude-sonnet-4-5 | $3.00          | $15.00          | $0.30        |
| claude-haiku-3-5  | $0.80          | $4.00           | $0.08        |
| gpt-4o            | $2.50          | $10.00          | $1.25        |
| gpt-4o-mini       | $0.15          | $0.60           | $0.075       |
| gemini-2.5-flash  | $0.075         | $0.30           | —            |
| llama.cpp (local) | $0.00          | $0.00           | —            |

---

## Testing Requirements

Every package in `internal/` must have:

1. **Unit tests** — pure logic, no network calls, table-driven
2. **Integration tests** (build tag `//go:build integration`) — real llama.cpp endpoint
3. **Benchmark tests** — `go test -bench` for tokenizer and compressor

Test coverage target: **≥ 80%** per package.

```bash
make test          # unit tests only
make test-int      # unit + integration (requires llama.cpp running)
make bench         # run benchmarks
make coverage      # html coverage report
```

---

## Makefile Targets

```makefile
build:       go build -o bin/walk .
install:     go install .
test:        go test ./... -v -short
test-int:    go test ./... -v -tags integration
bench:       go test ./... -bench=. -benchmem
coverage:    go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
lint:        golangci-lint run
clean:       rm -rf bin/ coverage.out
release:     goreleaser release --clean
```

---

## llama.cpp Integration Notes

- Server must be running: `llama-server --model /path/to/model.gguf --host 0.0.0.0 --port 8080`
- walk calls: `POST http://localhost:8080/v1/chat/completions`
- Health check: `GET http://localhost:8080/health`
- Tokenize endpoint: `POST http://localhost:8080/tokenize`
- walk detects Metal acceleration on M-series Mac automatically via `/health` response
- For M5 Max 36GB RAM: recommended models are Gemma 4 27B Q8 or GPT-4o-mini GGUF Q6

---

## Security Model

1. **API keys** are never stored in config — only `${ENV_VAR}` references
2. **Secret scrubber** runs on every outbound payload when `scrubber.enabled: true`
3. **Audit log** stores SHA-256 hash of every payload sent (not plaintext)
4. **Offline mode**: `walk analyze` and `walk scrub` work fully offline — no cloud calls
5. **No telemetry** — walk never phones home

---

## UX Principles

- Every command shows a savings line: `💾 Saved 4,312 tokens (~$0.043 at claude-sonnet-4-5)`
- Every error message includes a resolution hint
- `--dry-run` flag on all mutating commands
- `--json` flag on all commands for pipeline-friendly output
- `--quiet` flag to suppress decorative output in CI

---

## MVP Scope (Day 1 Target)

- [ ] `walk init` — working config wizard
- [ ] `walk analyze` — token count + cost + secret scan
- [ ] `walk compress` — llama.cpp pipe compression
- [ ] `walk scrub` — standalone secret scrubber
- [ ] `walk budget --status` — show today's spend
- [ ] Unit tests for: tokenizer, scrubber, config
- [ ] README.md with install + quickstart

---

## Agent Prompting Instructions

When using this spec with Antigravity, Claude Code, or GitHub Copilot:

> "Use walk-spec.md as the authoritative source of truth. Do not deviate from the package
> structure, command names, config schema, or test requirements defined here. When in doubt,
> prefer simplicity and testability over features. Every function must have a corresponding
> unit test."

---

_Last updated: 2026-06-28 | Version: 0.1.0-spec_
