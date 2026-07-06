# Example: walk with OpenAI Codex

This example shows how to use walk alongside OpenAI Codex (or GPT-4o/GPT-4o-mini via the OpenAI API) to control token spend and scrub secrets before they leave your machine.

## Setup

```bash
# 1. Install walk
go install github.com/mrwalker511/walk@latest

# 2. Export your OpenAI API key
export OPENAI_API_KEY="sk-..."

# 3. Configure walk
walk init   # sets api_key: ${OPENAI_API_KEY} in config
```

## Analyze cost before sending

Codex pricing is model-dependent. Use walk to estimate cost before sending a payload:

```bash
walk analyze prompt.md --model gpt-4o
```

```
Tokens:          1,203
Estimated cost:  $X.XX (input) + $X.XX (output)
Warnings:        none
Secrets:         none
```

> **Note:** GPT-4o and GPT-4o-mini prices are currently `$X.XX` placeholders in the pricing table. Update `internal/tokenizer/tokenizer.go` with confirmed prices from [openai.com/pricing](https://openai.com/pricing) before using these estimates in production.

## Scrub before sending

Always scrub payloads that may contain secrets before they go to the API:

```bash
walk scrub my_codebase_context.txt > clean_context.txt
# exit code 1 if secrets found — stops the pipeline

# Then send the clean version to your Codex workflow
cat clean_context.txt | your_codex_script.sh
```

## Compress large context

GPT-4 models have context limits. Compress large inputs locally before sending:

```bash
cat large_context.md | walk compress --model gpt-4o > compressed.md
diff <(wc -w large_context.md) <(wc -w compressed.md)
```

## Watch spend during an interactive session

```bash
walk watch --tool codex
```

walk monitors token burn rate and warns when you approach the budget cap.

## Post-session report

```bash
walk report --session last
```

## Workflow summary

```bash
export OPENAI_API_KEY="sk-..."

# Pre-session
walk analyze prompt.md --model gpt-4o
walk scrub payload.txt > clean_payload.txt

# Compress if needed
cat large_context.md | walk compress > compressed.md

# Monitor spend
walk watch --tool codex

# Review after
walk report --session last
```
