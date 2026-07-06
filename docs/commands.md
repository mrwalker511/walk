# Command Reference

All commands share a set of global flags documented at the end of this page.

---

## `walk init`

Interactive setup wizard. Detects a running llama.cpp server, collects API keys, and writes `~/.walk/config.yaml`.

```bash
walk init
```

- Validates the llama.cpp health endpoint at `http://localhost:8080/health`
- Stores API keys as `${ENV_VAR}` references — never as plaintext
- Safe to re-run; existing values are shown as defaults
- Pass `--quiet` to skip all interactive prompts and write defaults silently
- Pass `--dry-run` to preview what would be written without creating any files

---

## `walk analyze [file]`

Inspect a prompt or payload before sending it to an LLM. Reads from `file` or stdin when no argument is given.

```bash
walk analyze prompt.md
cat context.md | walk analyze
walk analyze prompt.md --model claude-sonnet-4-5 --json
```

**Output**

- Token count (model-aware approximation)
- Estimated input and output cost
- Dead-weight warnings: repeated instructions, long lines of raw data
- Repetition fingerprint: duplicate content blocks flagged by line number
- Secret/PII scan results
- Compression recommendation if token count exceeds 1,000

**JSON output (`--json`)**

```json
{
  "model": "claude-sonnet-4-5",
  "token_count": 1203,
  "word_count": 847,
  "line_count": 62,
  "estimated_output_tokens": 300,
  "input_cost_usd": 0.003609,
  "output_cost_usd": 0.004500,
  "total_cost_usd": 0.008109,
  "warnings": [
    { "code": "LONG_LINE", "severity": "info", "line": 14, "hint": "..." }
  ],
  "has_secrets": false
}
```

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | Analysis complete, no secrets found |
| `1` | Secrets detected |

---

## `walk compress [--file <path>]`

Summarise or compress content using the local llama.cpp server.

```bash
cat big_context.md | walk compress
walk compress --file prompt.md
walk compress --file prompt.md --local
```

**Flags**

| Flag | Description |
|---|---|
| `--file <path>`, `-f` | Read from file instead of stdin |
| `--local` | Force local inference; error if llama.cpp is unavailable |
| `--dry-run` | Show token count of what would be sent without calling the API |
| `--json` | Output as JSON |

**Output**

- Compressed text on stdout
- Compression ratio and token delta on stderr (suppressed with `--quiet`)

**JSON output (`--json`)**

```json
{
  "original_tokens": 5840,
  "compressed_tokens": 1212,
  "compression_ratio": 0.208,
  "tokens_saved": 4628,
  "model": "gemma-4-27b-q8_0",
  "compressed": "..."
}
```

Requires a running llama.cpp server. See [examples/llama-cpp.md](examples/llama-cpp.md).

---

## `walk diff <original> <optimized>`

Token and cost comparison between two payload versions.

```bash
walk diff original.md optimized.md
walk diff original.md optimized.md --model claude-sonnet-4-5
walk diff original.md optimized.md --json
```

**Output**

- Token count for each file
- Delta (tokens saved / added) and percentage change
- Cost delta at the active model's input pricing

Note: `walk diff` reports numeric deltas only. It does not show inline text highlighting of removed content.

**JSON output (`--json`)**

```json
{
  "model": "claude-sonnet-4-5",
  "original_tokens": 1200,
  "optimized_tokens": 300,
  "token_delta": -900,
  "original_cost_usd": 0.003600,
  "optimized_cost_usd": 0.000900,
  "cost_delta_usd": -0.002700
}
```

---

## `walk watch [--tool <name>]`

Live session monitor. Polls the local session database and displays token spend in real time. Press Ctrl+C to stop.

```bash
walk watch
walk watch --tool claude-code
walk watch --tool codex --interval 5
```

**Flags**

| Flag | Description |
|---|---|
| `--tool <name>` | Label the monitor: `claude-code` or `codex` |
| `--interval <seconds>` | Polling interval (default: `3`) |
| `--json` | Emit one JSON object per tick instead of a live status line |
| `--quiet` | Suppress the status line; budget warnings still appear |

**Output**

A status line is printed on each tick, overwritten in place:

```
[15:04:05] spend: $8.0000 / $10.00 (80.0%) | tokens: 2.7M | burn: $0.312/hr | 8h proj: $10.50
⚠ Warning: 80% of daily budget used
```

Fields:
- **spend** — today's cumulative cost vs. the daily limit
- **tokens** — total tokens recorded today
- **burn** — cost burn rate in USD/hr (computed from spend change between ticks)
- **8h proj** — projected 8-hour total at the current burn rate

Note: `walk watch` reads from the local SQLite session database (`~/.walk/sessions.db`). It does not intercept or attach to the Claude Code or Codex process directly.

**Per-tick JSON output (`--json`)**

```json
{
  "spend_usd": 8.0,
  "limit_usd": 10.0,
  "used_percent": 80.0,
  "tokens_today": 2700000,
  "burn_rate_usd_per_hour": 0.312,
  "timestamp": "2026-07-06T15:04:05Z"
}
```

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | Stopped via Ctrl+C or SIGTERM |
| `1` | Daily budget exceeded and `budget.hard_stop: true` in config |

---

## `walk report [--session <id|last|all>] [--format <fmt>]`

Post-session cost breakdown.

```bash
walk report                      # last session (default)
walk report --session all        # all sessions
walk report --session 42         # specific session by numeric ID
walk report --format csv > sessions.csv
walk report --json               # JSON output (overrides --format)
```

**Flags**

| Flag | Description |
|---|---|
| `--session <id\|last\|all>` | Which session(s) to report (default: `last`) |
| `--format <table\|json\|csv>` | Output format (default: `table`, or `output.default_format` from config) |

**Format precedence:** `--json` global flag → `--format` flag → `output.default_format` in config → `table`.

**CSV columns**

`id`, `model`, `tag`, `started_at`, `tokens_in`, `tokens_out`, `tokens_cached`, `cost_usd`

---

## `walk cache analyze [file]`

Analyse a prompt for prefix cache optimisation. Reads from `file` or stdin when no argument is given.

```bash
walk cache analyze prompt.md
cat system_prompt.md | walk cache analyze
walk cache analyze prompt.md --json
```

**Output**

- Sections classified as stable (system prompt, instructions) vs. dynamic (user input)
- Token counts for stable and dynamic regions
- Reorder recommendation if dynamic content precedes stable content
- Estimated cache savings per request at Anthropic and OpenAI cached-input pricing

**JSON output (`--json`)**

```json
{
  "stable_tokens": 420,
  "dynamic_tokens": 85,
  "estimated_savings_anthropic_usd": 0.000126,
  "estimated_savings_openai_usd": 0.000053,
  "reorder_recommended": false,
  "recommendations": ["Consider moving dynamic content after stable instructions"],
  "sections": [
    { "label": "S", "tokens": 420, "preview": "You are a helpful assistant..." },
    { "label": "D", "tokens": 85, "preview": "User: What is 2+2?" }
  ]
}
```

---

## `walk scrub [file]`

Scan a payload for secrets and PII before it leaves your machine. Reads from `file` or stdin when no argument is given.

```bash
walk scrub payload.txt
cat payload.txt | walk scrub
walk scrub payload.txt --output clean.txt
walk scrub payload.txt --dry-run
```

**Flags**

| Flag | Description |
|---|---|
| `--output <path>`, `-o` | Write the clean payload to a file instead of stdout |
| `--dry-run` | Report findings without writing any output |
| `--json` | Output redaction report as JSON |
| `--quiet` | Suppress the redaction report; only emit the clean payload |

**Output**

- **stderr** — redaction report: each finding's type, line number, matched text, and redacted replacement
- **stdout** — clean payload with secrets replaced by `[REDACTED]`, or written to `--output <path>`

**JSON output (`--json`)**

```json
{
  "has_secrets": true,
  "findings": [
    { "type": "api_key", "line": 3, "match": "sk-ant-...", "redacted": "[REDACTED]" }
  ],
  "clean": "Here is my [REDACTED] key..."
}
```

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | No secrets found |
| `1` | One or more secrets detected |

Detected patterns: API keys (Anthropic, OpenAI, generic), JWTs, AWS credentials, SSH private keys, email addresses, SSNs, phone numbers, and high-entropy strings (default threshold: 4.5 bits/char).

---

## `walk budget`

Manage daily spend limits.

```bash
walk budget                    # same as --status (default)
walk budget --status           # show today's spend vs. cap
walk budget --set 5.00         # set $5.00 daily cap (current session only)
walk budget --reset            # reset today's counter to $0.00
walk budget --status --json    # JSON output
```

**Flags**

| Flag | Description |
|---|---|
| `--status` | Print current spend and cap (default when no flag given) |
| `--set <amount>` | Set a new daily cap in USD |
| `--reset` | Reset today's spend counter |
| `--json` | Output as JSON |
| `--dry-run` | Preview the change without applying it |

> **Note:** `--set <amount>` updates the in-memory budget for the current process only. It does **not** write to `~/.walk/config.yaml`. To persist a new limit across sessions, edit `budget.daily_limit` in config directly.

**JSON output (`--status --json`)**

```json
{
  "spend_usd": 3.42,
  "limit_usd": 10.0,
  "used_percent": 34.2,
  "tokens_today": 1140000
}
```

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | Within budget |
| `1` | Daily budget exceeded and `budget.hard_stop: true` in config |

---

## Global Flags

These flags are available on every command.

| Flag | Description |
|---|---|
| `--model <name>` | Override the default model from config (affects cost estimates in `analyze`, `diff`, `compress`, `cache analyze`) |
| `--json` | Output as JSON (pipeline-friendly) |
| `--quiet` | Suppress decorative output (for CI) |
| `--dry-run` | Show what would happen without making changes or API calls |
| `--config-dir <path>` | Config directory override (default: `~/.walk`) |
