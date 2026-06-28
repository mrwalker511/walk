# walk — LLM token optimizer

A CLI tool that intercepts, analyzes, and optimizes token payloads
before they reach your LLM. Sits alongside Claude Code, Codex, and
llama.cpp with zero workflow changes.

```
$ walk analyze --payload prompt.json
$ walk compress --model claude-sonnet-4
$ walk watch --adapter claude-code
$ walk report --session 2026-07-14
$ walk init
```

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  Your Tool   │───▶│    walk      │───▶│    LLM      │
│(Claude/Codex)│    │ (transparent  │    │(Claude/Codex│
│              │◀───│   proxy)     │◀───│ /llama.cpp) │
└─────────────┘    └──────────────┘    └─────────────┘
                        │
                  ┌─────┴──────┐
                  │  Pipeline   │
                  │ 1. Scrubber │
                  │ 2. Analyzer │
                  │ 3. Compressor│
                  │ 4. Cache     │
                  │ 5. Budget    │
                  └─────────────┘
```

## Quick start

```bash
walk init                  # create ~/.walk/config.yaml
walk watch                 # start proxy (default :9010)
walk analyze --on          # live analysis mode
walk compress --on         # enable compression via llama.cpp
```

## Config

`~/.walk/config.yaml`:

```yaml
default_provider: claude-code
providers:
  claude-code:
    adapter: claude-code
    model: claude-sonnet-4-20250514
  codex:
    adapter: codex
    model: gpt-4o
  llama:
    adapter: llama-cpp
    endpoint: http://localhost:8080

cache:
  enabled: true
  dir: ~/.walk/cache

budget:
  session_token_limit: 100000
  cost_alert: 0.50

scrub:
  enabled: true
  patterns: ["sk-[a-zA-Z0-9]+", "ghp_[a-zA-Z0-9]+"]
```