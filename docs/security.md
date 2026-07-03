# Security Model

## API Key Storage

API keys are **never written to disk**. `~/.walk/config.yaml` stores only environment variable references:

```yaml
providers:
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
```

At startup, `internal/config.ExpandVars` resolves these references via `os.Getenv`. If the variable is unset, the field remains empty — walk will error when a command tries to use the key, not silently pass an empty string to the API.

This means:
- Committing `~/.walk/config.yaml` to a repository is safe — it contains no secrets.
- Rotating a key requires only updating the environment variable, not editing config.

## Secret Scrubber

`walk scrub` (and the embedded scan in `walk analyze`) detects secrets before any payload leaves the machine. Detection uses two complementary strategies:

### Pattern matching

Regex patterns tuned for each secret type:

| Pattern name | What it detects |
|---|---|
| `api_key` | OpenAI `sk-...`, Anthropic `sk-ant-...`, generic `*_KEY` / `*_SECRET` patterns |
| `jwt` | Base64-encoded JWT tokens (`eyJ...`) |
| `aws_credential` | AWS access key IDs (`AKIA...`) and secret access keys |
| `ssh_key` | PEM private key blocks (`-----BEGIN ... PRIVATE KEY-----`) |
| `email` | RFC-5321 email addresses |
| `ssn` | US Social Security Number patterns (`XXX-XX-XXXX`) |
| `phone` | North American and international phone number formats |

Pattern matching is case-insensitive. Enable or disable individual patterns in `~/.walk/config.yaml` under `scrubber.patterns`.

### Shannon entropy analysis

High-randomness strings are flagged even if they don't match a named pattern. The entropy of each token is computed using the Shannon formula:

```
H = -Σ p(c) * log2(p(c))
```

Strings whose entropy exceeds `scrubber.entropy_threshold` (default: `4.5` bits/char) are reported as `TypeHighEntropy` findings. This catches novel secret formats, random tokens, and private keys that don't match known patterns.

Raise the threshold if you work with legitimate high-entropy data (base64 payloads, embeddings) and see false positives.

### Exit code

`walk scrub` exits with code 1 when any finding is detected and `scrubber.block_on_detect: true`. This makes it CI/CD-friendly:

```bash
walk scrub payload.txt | send_to_llm || exit 1
```

## Audit Log

Every outbound payload is recorded in `~/.walk/audit.log` as a SHA-256 hash — **never as plaintext**. The format is:

```
2026-07-03T14:22:01Z sha256=e3b0c44298fc1c149afb...  model=claude-sonnet-4-5  tokens=1247
```

This lets you audit which sessions sent data and how much, without the audit log itself becoming a secrets store. The SHA-256 hash is computed in `internal/session.AuditLog` before any network call is made.

Disable the audit log by setting `session.audit_enabled: false` in config.

## Offline Mode

`walk analyze` and `walk scrub` work entirely offline — they make no network calls. Token counting, cost estimation, dead-weight detection, repetition fingerprinting, and secret scanning all run locally.

`walk compress` requires a local llama.cpp server (also offline — no cloud calls if `local_model.enabled: true`).

Only `walk watch` and commands that need to report costs for an active cloud session need network access.

## No Telemetry

walk never phones home. There are no analytics, crash reporters, or usage metrics. The only network calls made by walk are:
- The llama.cpp server at `local_model.endpoint` (local network, under your control)
- Provider APIs (`api.anthropic.com`, `api.openai.com`) when explicitly routing to cloud
