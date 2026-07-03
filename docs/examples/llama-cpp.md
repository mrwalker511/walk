# Example: walk with llama.cpp

`walk compress` routes through a local llama.cpp server — no cloud calls, no API costs, no data leaving your machine. This example shows how to set up llama.cpp and use it with walk.

## Prerequisites

- llama.cpp built and installed: `llama-server` binary on your PATH
- A model file in GGUF format (e.g., Gemma 4 27B Q8 or GPT-4o-mini GGUF Q6)

## Start the llama.cpp server

```bash
# Basic start
llama-server --model /path/to/model.gguf --host 0.0.0.0 --port 8080

# M-series Mac with Metal acceleration
llama-server --model /path/to/model.gguf \
  --host 0.0.0.0 --port 8080 \
  --n-gpu-layers 99        # offload all layers to GPU
```

walk uses the OpenAI-compatible API exposed at `http://localhost:8080/v1/chat/completions`.

## Verify the server is healthy

```bash
curl http://localhost:8080/health
# {"status":"ok","slots_idle":1,"slots_processing":0}
```

Once healthy, `walk compress` will connect automatically. You can also verify via walk:

```bash
walk init   # re-run to confirm llama.cpp is detected
```

## Compress context with walk

```bash
# Pipe mode
cat big_context.md | walk compress

# File mode
walk compress --file big_context.md

# Output to file
walk compress --file big_context.md > compressed.md
```

walk sends the content to `POST http://localhost:8080/v1/chat/completions` with a summarisation prompt, then prints the result plus compression stats:

```
Compressed 5,840 tokens → 1,212 tokens (79.2% reduction)
💾 Saved 4,628 tokens (~$0.014 at claude-sonnet-4-5 equivalent)
```

## Recommended models

For M5 Max (36 GB RAM):

| Model | VRAM | Quality | Speed |
|---|---|---|---|
| Gemma 4 27B Q8 | ~29 GB | Best | Moderate |
| GPT-4o-mini GGUF Q6 | ~18 GB | Good | Fast |

For lower-memory machines, use a Q4 or Q5 quantisation of a smaller model (e.g., Gemma 3 12B Q5).

## Configure walk to use a specific model

Update `~/.walk/config.yaml`:

```yaml
local_model:
  provider: llama.cpp
  endpoint: http://localhost:8080/v1
  model: gemma-4-27b-q8_0   # must match the model loaded by llama-server
  timeout_seconds: 60        # increase for slow hardware
  enabled: true
```

## Full local pipeline (zero cloud calls)

```bash
# 1. Start llama.cpp
llama-server --model ~/models/gemma-4-27b-q8_0.gguf --port 8080 &

# 2. Analyze payload locally (no network)
walk analyze big_context.md

# 3. Scrub secrets locally (no network)
walk scrub big_context.md > clean.md

# 4. Compress locally via llama.cpp (no cloud)
walk compress --file clean.md > compressed.md

# 5. Send compressed output to your cloud LLM
cat compressed.md | your_llm_client
```

This pipeline keeps sensitive content off the internet through steps 1–4.
