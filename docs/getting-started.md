# Getting Started with walk

## Prerequisites

- **Go 1.25.0 or later** — required by the `modernc.org/sqlite` dependency.
- **llama.cpp** _(optional)_ — needed for `walk compress`. Not required for `walk analyze`, `walk scrub`, or `walk budget`.

## Install

```bash
# From source (recommended)
go install github.com/mrwalker511/walk@latest

# Or download a pre-built binary from GitHub Releases
# https://github.com/mrwalker511/walk/releases
```

Verify the install:

```bash
walk --help
```

## Initial Setup

Run the setup wizard once per machine:

```bash
walk init
```

The wizard:
1. Detects whether a llama.cpp server is running at `http://localhost:8080`
2. Prompts for your Anthropic and/or OpenAI API keys — these are written as `${ENV_VAR}` references in `~/.walk/config.yaml`, **never as plaintext**
3. Sets a daily budget cap and warning threshold
4. Writes `~/.walk/config.yaml`

You can re-run `walk init` at any time to reconfigure. Existing values are shown as defaults.

## First Commands

### Analyze a prompt before sending

```bash
walk analyze prompt.md
```

Output shows token count, estimated cost, dead-weight warnings, repetition findings, and any detected secrets.

```bash
# Pipe from stdin
echo "You are a helpful assistant. $(cat my_context.md)" | walk analyze
```

### Scrub secrets before sending

```bash
walk scrub payload.txt
```

Prints a clean version to stdout and a redaction report to stderr. Exits with code 1 if secrets are found — useful in CI:

```bash
walk scrub payload.txt > clean.txt || exit 1
```

### Compress with llama.cpp

```bash
cat big_context.md | walk compress
```

Summarises the input using your local llama.cpp model and prints the compressed result. Shows the compression ratio and token delta.

### Check today's spend

```bash
walk budget --status
```

## Next Steps

- See [commands.md](commands.md) for a full reference of every subcommand and flag.
- See [configuration.md](configuration.md) for the full `config.yaml` schema.
- See [examples/claude-code.md](examples/claude-code.md) for a complete Claude Code workflow.
