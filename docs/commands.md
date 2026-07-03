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

---

## `walk analyze [file]`

Inspect a prompt or payload before sending it to an LLM.

```bash
walk analyze prompt.md
cat context.md | walk analyze
```

**Flags**

| Flag | Description |
|---|---|
| `--model <name>` | Model to use for cost estimates (default: from config) |
| `--json` | Output as JSON |
| `--quiet` | Suppress decorative output |

**Output**

- Token count (model-aware approximation)
- Estimated input and output cost
- Dead-weight warnings: repeated instructions, long lines of raw data
- Repetition fingerprint: duplicate content blocks flagged by line number
- Secret/PII scan results
- Compression recommendation if token count exceeds 1,000

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | Analysis complete, no secrets found |
| `1` | Secrets detected (when `scrubber.block_on_detect: true`) |

---

## `walk compress [--file <path>]`

Summarise or compress content using the local llama.cpp server.

```bash
cat big_context.md | walk compress
walk compress --file prompt.md
walk compress --file prompt.md --model gemma-4-27b-q8_0
```

**Flags**

| Flag | Description |
|---|---|
| `--file <path>` | Read from file instead of stdin |
| `--model <name>` | Override local model from config |
| `--local` | Force local inference even if a cloud model is configured |
| `--dry-run` | Show what would be sent without making the API call |
| `--json` | Output as JSON |

**Output**

- Compressed text on stdout
- Compression ratio and token delta on stderr

Requires a running llama.cpp server. See [examples/llama-cpp.md](examples/llama-cpp.md).

---

## `walk diff <original> <optimized>`

Side-by-side token and cost comparison between two payload versions.

```bash
walk diff original.md optimized.md
walk diff original.md optimized.md --model claude-sonnet-4-5
```

**Flags**

| Flag | Description |
|---|---|
| `--model <name>` | Model for cost calculations |
| `--json` | Output as JSON |

**Output**

- Token count for each file
- Delta (tokens saved / added)
- Cost delta at the specified model's pricing

---

## `walk watch [--tool <name>]`

Live session monitor. Tracks token usage in real time and enforces budget caps.

```bash
walk watch
walk watch --tool claude-code
walk watch --tool codex
```

**Flags**

| Flag | Description |
|---|---|
| `--tool <name>` | Tool to attach to: `claude-code` or `codex` |
| `--json` | Emit JSON events instead of a live table |
| `--quiet` | Suppress decorative output |

**Output**

- Real-time token burn rate (tokens/min)
- Budget cap status (warn at threshold, hard-stop at cap)
- Burn rate projection: "at this rate: $X total"
- Context rot alerts when active context grows stale
- "Lost in the middle" warning when context exceeds 60% fill

---

## `walk report [--session <id|last|all>] [--format <fmt>]`

Post-session cost breakdown and savings summary.

```bash
walk report                      # last session
walk report --session all        # all sessions
walk report --session abc123     # specific session ID
walk report --format csv > sessions.csv
```

**Flags**

| Flag | Description |
|---|---|
| `--session <id\|last\|all>` | Which session(s) to report (default: `last`) |
| `--format <table\|json\|csv>` | Output format (default: `table`) |
| `--quiet` | Suppress decorative output |

**Output**

- Tokens used: input / output / cached
- Cache hit ratio and estimated cache savings
- Cost attribution by tag or project
- Cumulative savings vs. unoptimised baseline

---

## `walk cache analyze <file>`

Analyse a prompt for prefix cache optimisation.

```bash
walk cache analyze prompt.md
cat system_prompt.md | walk cache analyze
```

**Flags**

| Flag | Description |
|---|---|
| `--model <name>` | Model for savings estimates |
| `--json` | Output as JSON |

**Output**

- Sections classified as stable (system prompt, instructions) vs. dynamic (user input)
- Reorder recommendation if dynamic content precedes stable content
- Estimated savings per request at Anthropic and OpenAI cached-input pricing

---

## `walk scrub <file>`

Scan a payload for secrets and PII before it leaves your machine.

```bash
walk scrub payload.txt
cat payload.txt | walk scrub
walk scrub payload.txt > clean.txt
```

**Flags**

| Flag | Description |
|---|---|
| `--json` | Output redaction report as JSON |
| `--quiet` | Suppress redaction report; only emit clean payload |

**Output**

- Clean payload on stdout (secrets replaced with `[REDACTED]`)
- Redaction report on stderr listing each finding: type, line, match excerpt

**Exit codes**

| Code | Meaning |
|---|---|
| `0` | No secrets found |
| `1` | One or more secrets detected |

Detected patterns: API keys, JWTs, AWS credentials, SSH keys, email addresses, SSNs, phone numbers, and high-entropy strings (threshold: 4.5 by default).

---

## `walk budget`

Manage daily and session spend limits.

```bash
walk budget --status           # show today's spend vs. cap
walk budget --set 5.00         # set $5.00 daily cap
walk budget --reset            # reset today's counter to $0.00
```

**Flags**

| Flag | Description |
|---|---|
| `--status` | Print current spend and cap |
| `--set <amount>` | Set a new daily cap in USD |
| `--reset` | Reset today's spend counter |
| `--json` | Output as JSON |

---

## Global Flags

These flags are available on every command.

| Flag | Description |
|---|---|
| `--json` | Output as JSON (pipeline-friendly) |
| `--quiet` | Suppress decorative output (for CI) |
| `--dry-run` | Show what would happen without making changes or API calls |
| `--model <name>` | Override the default model from config |
| `--config-dir <path>` | Config directory (default: `~/.walk`) |
