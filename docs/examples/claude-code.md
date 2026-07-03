# Example: walk with Claude Code

This example shows a complete workflow for using walk alongside a Claude Code session to monitor token spend, scrub secrets, and compress large context files.

## Setup

```bash
# 1. Install walk
go install github.com/mrwalker511/walk@latest

# 2. Configure
walk init   # enter your Anthropic API key when prompted
```

## Before a session: analyze and scrub your context

Before starting Claude Code, inspect what you're about to feed into the session:

```bash
# Analyze token count, cost, and any dead weight
walk analyze CLAUDE.md

# Scrub secrets from a context file before it goes to the model
walk scrub payload.txt > clean_payload.txt
```

Example output from `walk analyze CLAUDE.md`:

```
Tokens:          847
Estimated cost:  $0.003 (input) + $0.001 (output)
Warnings:        none
Secrets:         none
```

## During a session: watch token burn

In a separate terminal, run the live monitor while Claude Code is running:

```bash
walk watch --tool claude-code
```

walk tracks the running token count and enforces the budget cap you set during `walk init`. When you approach the warning threshold (default: 80% of daily limit), you'll see:

```
⚠️  WARNING: 80% of daily budget used ($8.00 / $10.00)
   Burn rate: 2,400 tokens/min — projected total: $11.20
```

If `budget.hard_stop: true`, walk exits and prevents further API calls once the cap is hit.

## During a session: compress large context

If a context file is getting large, compress it before passing it back to Claude Code:

```bash
cat bloated_context.md | walk compress > compressed.md
```

Output includes the compression ratio:

```
Compressed 4,312 tokens → 1,048 tokens (75.7% reduction)
💾 Saved 3,264 tokens (~$0.010 at claude-sonnet-4-5)
```

## After a session: review the cost report

```bash
walk report --session last
```

```
Session: abc123  Model: claude-sonnet-4-5  Duration: 47m
─────────────────────────────────────────────────────────
Input tokens:    42,810    Cost: $0.128
Output tokens:   11,203    Cost: $0.168
Cached tokens:    8,400    Cost: $0.003  (saved $0.025 vs uncached)
─────────────────────────────────────────────────────────
Total:                           $0.299
Cache hit rate:  16.4%
```

## Workflow summary

```bash
# Pre-session
walk analyze CLAUDE.md
walk scrub context.txt > clean_context.txt

# During session (separate terminal)
walk watch --tool claude-code

# Compress if context grows large
cat context.md | walk compress > context_compressed.md

# Post-session
walk report --session last
walk budget --status
```
