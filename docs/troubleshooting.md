# Troubleshooting

## `walk compress` fails: "connection refused" or "llama.cpp not available"

`walk compress` requires a running llama.cpp server.

**Start the server:**

```bash
llama-server --model /path/to/model.gguf --host 0.0.0.0 --port 8080
```

**Verify it's healthy:**

```bash
curl http://localhost:8080/health
```

Expected response: `{"status": "ok"}` (or similar). Once healthy, `walk compress` will connect automatically.

If your server runs on a non-default address, update `local_model.endpoint` in `~/.walk/config.yaml`:

```yaml
local_model:
  endpoint: http://localhost:9090/v1
```

---

## `walk init` says API key is empty after entering it

`walk init` writes `${ANTHROPIC_API_KEY}` (the env var reference) to config — it does not store the key value itself. You need to export the variable in your shell before running `walk`:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
walk analyze prompt.md
```

Add the export to your shell profile (`~/.zshrc`, `~/.bashrc`) to avoid repeating it.

---

## Budget hard-stop: "daily limit exceeded"

If `budget.hard_stop: true` is set, walk exits with an error once the daily cap is reached.

**Check current spend:**

```bash
walk budget --status
```

**Reset the counter** (use with care — this clears the day's tracked spend):

```bash
walk budget --reset
```

**Raise the cap:**

```bash
walk budget --set 20.00
```

Or set `budget.hard_stop: false` in `~/.walk/config.yaml` to warn instead of stopping.

---

## `go test` or `make lint` fails with "unsupported Go version"

This happens when golangci-lint was installed as a pre-built binary compiled with Go 1.24. That binary hard-rejects Go 1.25 targets at startup.

**Fix: install golangci-lint from source using the project's Go toolchain:**

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Do **not** use `golangci/golangci-lint-action` in CI — it downloads the same incompatible binary. The project's `.github/workflows/ci.yml` already uses `go install`.

---

## Config file not found

If walk can't find `~/.walk/config.yaml`, run the setup wizard:

```bash
walk init
```

Or specify an alternative directory:

```bash
walk --config-dir /path/to/config analyze prompt.md
```

---

## `walk scrub` flags everything as high entropy

The default entropy threshold is 4.5. Long random-looking strings (base64 blobs, hashes, UUIDs) may exceed this. Options:

1. Raise the threshold in config if false positives are frequent:
   ```yaml
   scrubber:
     entropy_threshold: 5.5
   ```
2. Remove the `api_key` pattern if you're working with non-secret high-entropy data and only care about structured secrets:
   ```yaml
   scrubber:
     patterns:
       - jwt
       - aws_credential
       - ssh_key
   ```

---

## Session database error: "unable to open database file"

The default DB path is `~/.walk/sessions.db`. If the `~/.walk/` directory doesn't exist:

```bash
mkdir -p ~/.walk
```

Or override the path in config:

```yaml
session:
  db_path: /tmp/walk-dev.db
```

---

## `walk watch` shows $0.00 cost for every event

Costs are `$0.00` for placeholder-priced models (all current-generation Claude, GPT-4o, Gemini). The pricing table in `internal/tokenizer/tokenizer.go` has `0.000` for these models pending verification from official provider pages. Set real prices once verified, or switch `--model` to a legacy entry (`claude-sonnet-4-5`) that has confirmed pricing.
