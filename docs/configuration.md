# Configuration Reference

`walk` reads its configuration from `~/.walk/config.yaml`. Run `walk init` to create this file interactively. All fields shown below are optional — sensible defaults are used for any field that is absent.

## Full Schema

```yaml
walk:
  version: "1"

local_model:
  provider: llama.cpp
  endpoint: http://localhost:8080/v1
  model: gemma-4-27b-q8_0
  timeout_seconds: 30
  enabled: true

providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    default_model: claude-sonnet-4-5
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-4o

budget:
  daily_limit: 10.00
  session_limit: 2.00
  warn_at_percent: 80
  hard_stop: true

scrubber:
  enabled: true
  block_on_detect: true
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
  show_savings_line: true
  default_format: table
```

## Field Reference

### `walk`

| Field | Default | Description |
|---|---|---|
| `version` | `"1"` | Config schema version. Always `"1"`. |

### `local_model`

Controls the llama.cpp local inference server used by `walk compress`.

| Field | Default | Description |
|---|---|---|
| `provider` | `llama.cpp` | Local inference provider. Only `llama.cpp` is supported. |
| `endpoint` | `http://localhost:8080/v1` | Base URL for the OpenAI-compatible API. |
| `model` | _(empty)_ | Model name passed to the server. Match the `--model` flag you used when starting `llama-server`. |
| `timeout_seconds` | `30` | HTTP timeout for compression requests. |
| `enabled` | `true` | Set to `false` to disable local inference and always use cloud. |

### `providers`

API keys are **always stored as environment variable references** — walk reads the referenced variable at startup via `os.Getenv`. Never write a raw key in this file.

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}   # walk reads os.Getenv("ANTHROPIC_API_KEY")
```

| Field | Default | Description |
|---|---|---|
| `anthropic.api_key` | _(none)_ | `${ENV_VAR}` reference to your Anthropic API key. |
| `anthropic.default_model` | `claude-sonnet-4-5` | Model used when `--model` is not specified. |
| `openai.api_key` | _(none)_ | `${ENV_VAR}` reference to your OpenAI API key. |
| `openai.default_model` | `gpt-4o` | Model used when `--model` is not specified for OpenAI. |

### `budget`

| Field | Default | Description |
|---|---|---|
| `daily_limit` | `10.00` | Maximum USD spend per calendar day. |
| `session_limit` | `2.00` | Maximum USD spend per `walk watch` session. |
| `warn_at_percent` | `80` | Issue a warning when this % of the daily limit is reached. |
| `hard_stop` | `true` | Exit with an error instead of a warning when the limit is exceeded. |

### `scrubber`

Controls the secret and PII scanner that runs on every outbound payload.

| Field | Default | Description |
|---|---|---|
| `enabled` | `true` | Enable or disable the scrubber globally. |
| `block_on_detect` | `true` | Exit with code 1 when secrets are found (CI/CD friendly). |
| `patterns` | _(list below)_ | Active detection patterns. |
| `entropy_threshold` | `4.5` | Shannon entropy threshold for high-randomness string detection. Strings above this threshold are flagged even if they don't match a named pattern. |

Default active patterns: `api_key`, `jwt`, `aws_credential`, `ssh_key`, `email`, `ssn`, `phone`.

### `cache`

| Field | Default | Description |
|---|---|---|
| `track_hits` | `true` | Record cache hit/miss statistics in the session ledger. |
| `optimize_on_analyze` | `true` | Run prefix-cache analysis automatically during `walk analyze`. |

### `session`

| Field | Default | Description |
|---|---|---|
| `db_path` | `~/.walk/sessions.db` | Path to the SQLite session ledger. |
| `audit_log` | `~/.walk/audit.log` | Path to the append-only audit log. |
| `audit_enabled` | `true` | Write SHA-256 hashes of every outbound payload to the audit log. **Plaintext is never written.** |

### `output`

| Field | Default | Description |
|---|---|---|
| `color` | `true` | Enable ANSI colour output. Set to `false` in environments that don't support colour. |
| `show_savings_line` | `true` | Print a savings summary line after every command. |
| `default_format` | `table` | Default output format. Options: `table`, `json`, `csv`. |

## Environment Variable Expansion

Any value in `config.yaml` that matches `${VAR_NAME}` is expanded at load time using the value of the environment variable `VAR_NAME`. This applies to `api_key` fields and can be used for any string field.

```yaml
session:
  db_path: ${WALK_DB_PATH}   # falls back to ~/.walk/sessions.db if unset
```
