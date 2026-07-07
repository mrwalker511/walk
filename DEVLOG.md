# walk — Development Log

Plain-English record of what was built, why, and what changed. Newest sessions at the top.

---

## Session 9 — 2026-07-06

**PRs: #17**

Documentation completeness pass before release. Two Explore agents cross-referenced every doc file against the codebase and found factual inaccuracies, missing flags, and spec aspirations presented as shipped features.

`docs/commands.md` received the most changes. `walk watch` had three factual errors: the burn rate description said "tokens/min" when the code emits "USD/hr"; the projection said "at this rate: $X total" when the code prints "8h proj: $X.XX" (8-hour window, hardcoded); and two unimplemented features ("context rot alerts" and "'lost in the middle' warnings") were listed as if they existed. The `--interval` flag was entirely absent from the flags table. Added the missing `walk scrub --output`/`-o` flag (writes clean output to a file) which had no documentation at all. Added complete JSON output schemas for every command (`analyze`, `compress`, `diff`, `watch`, `scrub`, `budget`, `cache analyze`). Clarified that `walk budget --set` is in-memory only and does not persist to config. Clarified that `walk watch` polls SQLite and does not intercept the Claude Code process. Removed unimplemented report output claims (cache hit ratio, savings baseline). Added format precedence note for `walk report`. Added CSV column list.

`docs/security.md`: fixed the audit log format example — removed `model=` and `tokens=` fields that don't exist in the actual log line written by `session.go`.

`docs/examples/codex.md`: removed `--model gpt-4o` from a `walk report` invocation — `walk report` has no `--model` flag; the command would have errored.

`docs/examples/claude-code.md`: replaced the watch output sample with the actual format emitted by `watch.go` (pipe-separated status line with `burn: $X/hr` and `8h proj: $X.XX` fields).

`walk-spec.md`: checked all MVP checklist items (`- [x]` for all — every item is shipped); removed the claim that walk auto-detects Metal acceleration on M-series Mac (not implemented); updated Homebrew tap entry to "planned, not yet available"; updated version timestamp to 2026-07-06.

---

## Session 8 — 2026-07-04

**PRs: #16**

Added `internal/router/router_integration_test.go` under the `//go:build integration` build tag. The file follows the same pattern as `internal/compressor/compressor_integration_test.go` — both require a real llama.cpp server running at `localhost:8080` and are excluded from `go test ./... -short`. Two tests: `TestRouteIntegration` calls `Route(ctx, false)` and asserts the decision routes locally, and `TestCheckLocalHealthIntegration` calls `CheckLocalHealth` directly and asserts the server is reachable. The existing unit tests in `router_test.go` already exercise all routing logic via `httptest.Server` and `NewWithClient`; the integration test adds end-to-end coverage against a live server. All short tests remain green.

---

## Session 7 — 2026-07-03

**PRs: #14, #15**

Two PRs shipped back-to-back covering documentation and test coverage.

PR #14 populated all nine previously-empty docs files. `docs/getting-started.md` covers prerequisites, installation via `go install`, the `walk init` walkthrough, and a first-commands sequence. `docs/configuration.md` is a field-by-field reference for `~/.walk/config.yaml` — every provider, budget, scrubber, session, and local-model key with type and default. `docs/commands.md` documents every subcommand including flags, output format, and exit codes. `docs/troubleshooting.md` covers seven common failure modes with concrete fixes. `docs/security.md` explains the API-key model (env-var references only), the dual-strategy scrubber (regex + Shannon entropy), the SHA-256-only audit log, offline mode, and the no-telemetry guarantee. The three example files (`docs/examples/claude-code.md`, `docs/examples/codex.md`, `docs/examples/llama-cpp.md`) provide end-to-end workflow walkthroughs for each integration scenario.

PR #15 added `cmd/cmd_test.go` — the first test file for the `cmd/` package, which previously had 0% coverage. Thirty-six tests exercise: pure utility functions (`formatTokens`, `errorHint`, `contains`, `printSavings`), all six testable subcommand runners (`runAnalyze`, `runDiff`, `runScrub`, `runCacheAnalyze`, `runBudget`, `runReport`), JSON and CSV output paths, and stdin reading. A `captureStdout` helper using `os.Pipe()` intercepts output without touching `os.Stdout` globally; `resetGlobals()` zeroes all package-level cobra flag vars between tests to prevent state leakage. Final coverage: all `internal/` packages ≥ 80% (several at 100%); `cmd/` reached ~56%, with the remaining gap attributable entirely to server-dependent commands (`runCompress`, `loadDefaultConfig`), interactive-prompt commands (`runInit`, `prompt`, `checkLlamaHealth`), and an infinite-loop command (`runWatch`) — none of which are reachable in `-short` mode.

---

## Session 6 — 2026-07-03

**PRs: #13**

One combined PR covering a doc sync and test coverage improvements for two packages.

The doc sync piece added the Session 5 entry to DEVLOG.md and updated CLAUDE.md with status rows for PRs #11 and #12 — same housekeeping pattern as previous sessions.

`internal/analyzer` gained six new tests. `TestAnalyzeCleanText` establishes a no-warnings baseline for short clean input. `TestAnalyzeLongLineBoundary` pins the `> 500` boundary condition — exactly 500 chars must not warn, 501 must. `TestAnalyzeRepetitionShortChunksIgnored` verifies that `detectRepetition` skips windows with fewer than 10 words, which is the intended behaviour for trivially short repeated blocks. `TestAnalyzeTotalCostIsSum` asserts the `TotalCost == InputCost + OutputCost` invariant. `TestAnalyzeNoRepetitionFewLines` checks that text shorter than the 5-line window size doesn't panic and doesn't produce a false `DUPLICATE_BLOCK` warning. `TestAnalyzeWarningSeverities` verifies the `Severity` and `Hint` fields for all three warning codes: `LONG_LINE` (info), `DUPLICATE_BLOCK` (warning), and `SECRET_*` (error).

`internal/cache` gained six new tests mirroring the same audit pattern. `TestAnalyzeNoReorderNeeded` verifies that correct stable-before-dynamic ordering produces `ReorderRecommended = false`. `TestAnalyzeEmptyText` confirms empty input doesn't panic and returns all-zero fields. `TestAnalyzeAllStable` checks that purely stable content sets `DynamicTokens = 0`. `TestAnalyzeZeroSavingsNoStableTokens` confirms both savings figures are zero when there are no stable tokens. `TestAnalyzeDynamicHeavyRecommendation` exercises the third recommendation branch (`dynamicToks > stableToks`). `TestAnalyzeSavingsFormula` verifies the per-1M rates hardcoded in `Analyze`: Anthropic `(3.00 − 0.30) / 1_000_000` and OpenAI `(2.50 − 1.25) / 1_000_000`.

---

## Session 5 — 2026-07-01 / 2026-07-02

**PRs: #11, #12**

Two housekeeping PRs to keep documentation and the tokenizer current.

PR #11 was a doc sync pass: added the Session 4 entry to DEVLOG.md (the log was missing its own creation), added the PR #10 row to the CLAUDE.md status table, and fixed the Go version in walk-spec.md from `Go 1.22+` to `Go 1.25.0 (pinned by modernc.org/sqlite)` — the spec was written before the sqlite dependency locked the version and was never updated.

PR #12 updated the tokenizer for the current Claude model generation. Added `claude-sonnet-5`, `claude-haiku-4-5`, `claude-opus-4-8`, and `claude-fable-5` to `PricingTable`. All new model prices (and the existing OpenAI/Google entries) were set to `0.000` placeholder pending verification from official pricing pages — a TODO comment at the top of `PricingTable` links directly to Anthropic, OpenAI, and Google pricing. Legacy entries (`claude-sonnet-4-5`, `claude-haiku-3-5`) were kept with their real historical prices for backwards compatibility with existing session log records. README.md and walk-spec.md pricing tables were updated to show `$X.XX` placeholders with links to the provider pricing pages. Tests for the now-zero-priced models were updated to expect `0.00`, and the direction-logic tests (case-insensitivity, unknown-direction fallback) were switched to use `claude-sonnet-4-5` so they remain meaningful. `TestKnownModels` now checks model membership rather than exact count so adding models in future doesn't require a test change.

---

## Session 4 — 2026-06-30

**PRs: #10**

Added two documentation files to the repo root.

`CLAUDE.md` is a lean agent context file that Claude Code loads automatically at the start of each session. It holds the package map, hard constraints (env-var-only API keys, SHA-256-only audit log), errcheck conventions, current implementation status, and a pointer to the backlog. The goal is to prevent agents from re-deriving the same conventions each session and avoid bloating context with the full spec on every turn.

`DEVLOG.md` (this file) is a plain-English record of what happened in each development session — what was built, why each decision was made, and what changed. It also carries the Future Scope backlog so there is always one authoritative place to see what's left without opening the spec.

---

## Session 3 — 2026-06-30

**PRs: #8**

Fixed a silent data loss bug in the session package and plugged three test coverage gaps.

The bug was in `EndSession`: when it upserts the daily spend record, it was computing `tokens_total = tokensIn + tokensOut` and quietly dropping `tokensCached`. This meant every session that used Anthropic prefix caching under-reported total token usage in `walk report` and `walk watch`. One-word fix — added `+tokensCached` to the SQL expression. Also changed `ListSessions` to return an empty slice instead of nil when there are no rows (callers that check `len > 0` work fine either way, but nil is a trap for callers that check `!= nil`).

New tests added: `TestTodaySpendIncludesCached` (regression guard for the bug), `TestListSessionsEmpty` (nil-vs-empty guard), `TestAuditLogHashValue` (verifies the exact SHA-256 hash written to the audit log, not just that the prefix `sha256=` appears).

Also fixed eight more errcheck violations in this branch — `cmd/report.go` CSV writes, `cmd/watch.go` JSON encode, and `internal/compressor/compressor_test.go` HTTP handler mocks. These were leftover from the main lint sweep in PR #6 (the local branch had predated that merge).

---

## Session 2 — 2026-06-29 (late) / 2026-06-30 (early)

**PRs: #6, #7**

Two problems tackled: a CI toolchain mismatch and a batch of errcheck lint violations.

The CI was failing because golangci-lint downloaded a pre-built binary compiled with Go 1.24, which hard-rejects Go 1.25 targets at startup (version check in the binary itself). Fixed by replacing `golangci/golangci-lint-action@v6` in `.github/workflows/ci.yml` with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`, which compiles the linter from source using the Go 1.25 toolchain already set up by `setup-go`. After that, 19 errcheck violations surfaced across `cmd/budget.go`, `cmd/init.go`, `cmd/report.go`, `cmd/watch.go`, `internal/compressor/compressor.go`, `internal/router/router.go`, and `internal/session/session.go`. All fixed in PR #6 using the conventions now documented in CLAUDE.md.

PR #7 added four missing test cases to `internal/scrubber`: `TestScrubAnthropicKey` (the `ant[A-Za-z0-9_-]{30,}` pattern was untested), `TestScrubEntropyFindingType` (verify `TypeHighEntropy` appears in findings), `TestScrubEntropyThresholdFallback` (verify threshold ≤ 0 defaults to 4.5), and `TestScrubRedactedField` (verify `Finding.Redacted` is set and appears in the clean output).

---

## Session 1 — 2026-06-28 / 2026-06-29

**PRs: #1, #2, #4**

PR #1 was the initial full CLI implementation — all cobra commands wired up, all internal packages scaffolded, README and walk-spec.md written.

PR #2 added `ExpandVars` to `internal/config`. The config loader was storing literal strings like `${ANTHROPIC_API_KEY}` from YAML but never resolving them. Added `ExpandVars(*Config)` using `os.Expand(s, os.Getenv)` called at the end of `LoadFrom()`. Tests cover expansion, passthrough when no var, and empty-var edge case.

PR #4 added `CountTokens` and `EstimateCost` to `internal/tokenizer`. The lower-level `Count` and `Cost` functions already existed; these wrappers give callers the standard signature expected by other packages. `EstimateCost` accepts a direction string (`"input"`, `"output"`, `"cached"`) case-insensitively and defaults to input. Table-driven tests cover all six supported models (claude-sonnet-4-5, claude-haiku-3-5, gpt-4o, gpt-4o-mini, gemini-2.5-flash, llama.cpp) and all three cost directions. PR #4 also carried the first attempt at fixing the golangci-lint/Go 1.25 mismatch; the full fix landed in PR #6.

---

## Future Scope

Ordered by priority. Each item is a self-contained unit of work that follows the same pattern as the completed packages above: read the existing code, identify the gap, implement, test, verify clean build + lint, push.

### Next up

**`internal/analyzer` — coverage and any gaps**
The analyzer handles dead-weight detection and repetition fingerprinting. Check test coverage, add any missing table-driven tests, and look for bugs analogous to the tokensCached issue in session (e.g. any counters that silently drop data).

**`internal/cache` — coverage and any gaps**
The cache package handles prefix-cache analysis and reorder recommendations. Same pattern: audit the tests, fill gaps, fix any logic bugs.

### Near-term

**`docs/` directory — populate all doc files**
The spec references six docs files (`getting-started.md`, `configuration.md`, `commands.md`, `troubleshooting.md`, `security.md`) and three example files (`claude-code.md`, `codex.md`, `llama-cpp.md`). All directories exist but all files are empty. Writing these now would prevent duplicated explanations across README and walk-spec.

**Integration tests**
`walk-spec.md` requires integration tests under the `//go:build integration` tag for packages that talk to llama.cpp or an API. `internal/compressor` and `internal/router` are the main candidates.

**Coverage targets — ≥ 80% per package**
Run `make coverage` to get a baseline. Any package below 80% needs targeted tests. `cmd/` packages have no tests at all and would benefit from integration-style tests using `cobra.Command.Execute()` with a test DB.

### Later

**Pricing table refresh**
The tokenizer hardcodes pricing. As model prices change, there should be a way to update without a code change — perhaps a YAML pricing file in `~/.walk/` that overrides defaults.

**`walk diff` token delta**
The diff command exists but the token-delta highlighting (removed sections shown inline) may not be fully implemented. Worth verifying against the spec.

**`walk watch` context rot alerts**
The spec calls for staleness scoring and "lost in the middle" warnings when context exceeds 60% fill. Check whether the implementation covers this or if it's placeholder.

**Homebrew tap**
`.goreleaser.yaml` exists. Once the binary is stable, set up the Homebrew tap referenced in the README.
