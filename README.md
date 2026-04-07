# kubectl-loginsight

A kubectl plugin that analyzes Kubernetes logs using a local LLM. Pipe logs through it to get an instant summary or answer a specific question — no more `grep` needle-in-a-haystack.

## Installation

### Prerequisites

- [LM Studio](https://lmstudio.ai) with a model loaded and the local server running, or any other OpenAI-compatible LLM server (Ollama, vLLM, real OpenAI, etc.)

### Recommended machine specs (Mac)

**Baseline / reference setup:** MacBook Pro M2 Pro, 32 GB unified memory, running `qwen/qwen3-9b` GGUF via LM Studio.

### Install

```bash
# Install directly
go install github.com/InsomniaCoder/kubectl-loginsight@latest
```

Make sure `~/go/bin` is in your PATH:
```bash
export PATH="$PATH:$HOME/go/bin"  # add to ~/.zshrc to make it permanent
```

Verify kubectl discovers it:
```bash
kubectl plugin list  # should show /Users/<you>/go/bin/kubectl-loginsight
```

> **How kubectl discovers plugins:** kubectl scans every directory in `$PATH` for executables named `kubectl-*` and exposes them as subcommands. Because this binary is named `kubectl-loginsight`, you can invoke it as either `kubectl loginsight` or `kubectl-loginsight`.

## Usage

```bash
# Summarize what's happening in the logs
kubectl logs <pod> | kubectl-loginsight --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1

# Ask a specific question
kubectl logs <pod> | kubectl-loginsight --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1 -q "why did the pod crash?"

# Read from a saved log file
kubectl logs <pod> > ./app.log
kubectl-loginsight --file ./app.log --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1 -q "any OOMKilled signs?"
```

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--model` / `-m` | Yes | — | Model name to pass to the LLM API |
| `--question` / `-q` | No | — | Question to ask. If omitted, summarize mode is used |
| `--base-url` | No | `http://localhost:1234/v1` | OpenAI-compatible API base URL |
| `--api-key` | No | `test` | API key (not needed for local models) |
| `--max-tokens` | No | `6500` | Max tokens of log content to send (model context size minus ~1700 headroom for prompt + response) |
| `--file` / `-f` | No | — | Read logs from file instead of stdin |

## Config File

To avoid typing flags every time, create `~/.kube/log-insight.yaml`:

```yaml
model: qwen/qwen3.5-9b
base-url: http://localhost:1234/v1
api-key: test
max-tokens: 6500
```

Flags always override config file values.

## Using with other LLM backends

Since `--base-url` accepts any OpenAI-compatible endpoint, you can point it at any backend:

```bash
# Ollama
kubectl logs <pod> | kubectl-loginsight --model qwen2.5:7b --base-url http://localhost:11434/v1

# OpenAI
kubectl logs <pod> | kubectl-loginsight --model gpt-4o --base-url https://api.openai.com/v1 --api-key sk-...
```

## Large logs

If logs exceed `--max-tokens`, the oldest lines are dropped and a warning is printed to stderr:

```
⚠  Logs truncated to 6500 tokens (oldest lines removed)
```

Use `kubectl logs --tail=200 <pod>` to limit log output before piping if needed.

**Baseline context setup:** LM Studio loaded with 8192 context (Model Settings → Context Length), `--max-tokens 6500` leaves ~1700 tokens headroom for the system prompt and response. Thinking mode is disabled by default to improve speed.

## Testing

Run the unit tests:
```bash
go test ./...
```

End-to-end smoke test with a sample log file:
```bash
# 1. Start LM Studio, load qwen/qwen3.5-9b GGUF, and start the local server
#    (Server runs at http://localhost:1234/v1 by default)

# 2. Create a sample log file
cat > /tmp/test.log <<EOF
2024-01-01T10:00:00Z INFO  Starting server on :8080
2024-01-01T10:01:00Z INFO  Connected to database
2024-01-01T10:02:00Z ERROR Failed to process request: connection refused
2024-01-01T10:02:01Z WARN  Retrying (attempt 1/3)
2024-01-01T10:02:03Z ERROR Max retries exceeded, giving up
EOF

# 3. Summarize
kubectl-loginsight --file /tmp/test.log --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1

# 4. Ask a question
kubectl-loginsight --file /tmp/test.log --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1 -q "what went wrong?"
```

## Example output

```
➜ kubectl logs prometheus-prometheus-1 | kubectl-loginsight --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1 -q "do you find any problem"
⚠  Logs truncated to 6500 tokens (oldest lines removed)

Yes, I found several significant problems in these logs:

## Critical Issues

### 1. Rule Evaluation Failures (Most Severe)
Multiple recording rules are consistently failing with the error:
"vector contains metrics with the same labelset after applying rule labels"

**Root Cause:** These rules are trying to add labels but multiple metrics end up with
identical labelsets after transformation. Prometheus doesn't allow duplicate time series.

### 2. Sample Dropped from Ingestion Errors
3 samples were dropped due to conflicting values at the same timestamp for
coredns_dns_responses_total:sum_rate2m

## Recommended Actions
1. Fix Recording Rules — add unique identifiers to ensure each rule produces unique labelsets
2. Review pod info joins — rules joining on pods need unique grouping keys

⏱  57.8s
```
