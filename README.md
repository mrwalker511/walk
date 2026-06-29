# walk

> _"Slow down. Use less. Save more."_

`walk` is a CLI tool that optimizes LLM token usage across agentic coding workflows. It acts as an intelligent proxy layer — analyzing, compressing, and monitoring LLM payloads before, during, and after sessions with Claude Code, OpenAI Codex, and local llama.cpp.

## Install

```bash
go install github.com/mrwalker511/walk@latest
```

Or download a binary from [Releases](https://github.com/mrwalker511/walk/releases).

## Quickstart

```bash
# 1. Initialize (detects llama.cpp, sets API keys, configures budget)
walk init

# 2. Analyze a prompt before sending
walk analyze prompt.md

# 3. Scrub secrets before sending (exits 1 if found — CI friendly)
walk scrub payload.txt

# 4. Compress with local llama.cpp
cat big_context.md | walk compress

# 5. Compare two versions
walk diff original.md optimized.md

# 6. Check today's spend
walk budget --status
```

## Commands

| Command | Description |
|---|---|
| `walk init` | Interactive setup wizard |
| `walk analyze [file]` | Token count, cost estimate, dead-weight detection, secret scan |
| `walk compress [--file f]` | Compress via llama.cpp before sending to cloud |
| `walk diff <orig> <opt>` | Side-by-side token and cost comparison |
| `walk watch` | Live token burn rate monitor with budget enforcement |
| `walk report` | Post-session cost breakdown |
| `walk cache analyze [file]` | Prefix cache optimization recommendations |
| `walk scrub [file]` | Redact secrets and PII (exits 1 if found) |
| `walk budget` | Manage daily spend limits |

## Global Flags

```
--json          Output as JSON (pipeline-friendly)
--quiet         Suppress decorative output (for CI)
--dry-run       Show what would happen without making changes
--model <name>  Override default model
--config-dir    Config directory (default: ~/.walk)
```

## Configuration

Run `walk init` to create `~/.walk/config.yaml`. See [`walk-spec.md`](walk-spec.md) for the full schema.

## Supported Models

| Model | Input/1M | Output/1M | Cached/1M |
|---|---|---|---|
| claude-sonnet-4-5 | $3.00 | $15.00 | $0.30 |
| claude-haiku-3-5 | $0.80 | $4.00 | $0.08 |
| gpt-4o | $2.50 | $10.00 | $1.25 |
| gpt-4o-mini | $0.15 | $0.60 | $0.075 |
| gemini-2.5-flash | $0.075 | $0.30 | — |
| llama.cpp (local) | $0.00 | $0.00 | — |

## Development

```bash
make build      # Build binary to bin/walk
make test       # Unit tests
make bench      # Benchmarks
make coverage   # HTML coverage report
make lint       # golangci-lint
```

## Security

- API keys are **never** stored — only `${ENV_VAR}` references in config
- `walk scrub` scans every outbound payload for secrets and PII
- Audit log stores SHA-256 hashes of payloads, never plaintext
- `walk analyze` and `walk scrub` work fully offline — no cloud calls
- No telemetry — walk never phones home
