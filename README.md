# kubectl-log-insight

A kubectl plugin that analyzes Kubernetes logs using a local LLM. Pipe logs through it to get an instant summary or answer a specific question — no more `grep` needle-in-a-haystack.

## Usage

```bash
# Summarize what's happening in the logs (LM Studio default port)
kubectl logs <pod> | kubectl log-insight --model qwen/qwen3-9b --base-url http://localhost:1234/v1

# Ask a specific question
kubectl logs <pod> | kubectl log-insight --model qwen/qwen3-9b --base-url http://localhost:1234/v1 --question "why did the pod crash?"

# Read from a saved log file
kubectl logs <pod> > ./app.log
kubectl log-insight --file ./app.log --model qwen/qwen3-9b --base-url http://localhost:1234/v1 --question "any OOMKilled signs?"
```

## Installation

### Prerequisites

- [LM Studio](https://lmstudio.ai) with a model loaded and the local server running, or any other OpenAI-compatible LLM server (Ollama, vLLM, real OpenAI, etc.)

### Recommended machine specs (Mac)

**Baseline / reference setup:** MacBook Pro M2 Pro, 32 GB unified memory, running `qwen/qwen3-9b` GGUF via LM Studio.

### Install

```bash
# Install directly
go install github.com/InsomniaCoder/kubectl-log-insight@latest

# Or build from source:
git clone https://github.com/InsomniaCoder/kubectl-log-insight
cd kubectl-log-insight
go build -o kubectl-log-insight .
sudo mv kubectl-log-insight /usr/local/bin/
```

Verify kubectl discovers it:
```bash
kubectl plugin list  # should show kubectl-log-insight
```

kubectl discovers plugins by scanning every directory in your `$PATH` for executables named `kubectl-*`. Because this binary is named `kubectl-log-insight`, kubectl automatically exposes it as `kubectl log-insight` — dashes after the first one become spaces. No registration or manifest file needed.

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--model` / `-m` | No | — | Model name to pass to the LLM API |
| `--question` / `-q` | No | — | Question to ask. If omitted, summarize mode is used |
| `--base-url` | No | `http://localhost:1234/v1` | OpenAI-compatible API base URL |
| `--api-key` | No | `test` | API key / default is nothing as it's a local model |
| `--max-tokens` | No | `8192` | Max tokens of log content to send (model context size minus ~1700 headroom for prompt + response) |
| `--file` / `-f` | No | — | Read logs from file instead of stdin |

## Config File

To avoid typing flags every time, create `~/.kube/log-insight.yaml`:

```yaml
model: qwen/qwen3-9b
base-url: http://localhost:1234/v1
api-key: test
max-tokens: 8192
```

Flags always override config file values.

## Using with other LLM backends

Since `--base-url` accepts any OpenAI-compatible endpoint, you can point it at any backend:

## Testing

Run the unit tests:
```bash
go test ./...
```

## Large logs

If logs exceed `--max-tokens`, the oldest lines are dropped and a warning is printed to stderr:

```
⚠  Logs truncated to 8192 tokens (oldest lines removed)
```

Use `kubectl logs --tail=200 <pod>` to limit log output before piping if needed.

**Baseline context setup:** LM Studio loaded with 8192 context (Model Settings → Context Length), `--max-tokens 6500` leaves ~1700 tokens headroom for the system prompt and response, and thinking mode is disabled by default to improve speed.

## Example

```
 ➜ kubectl logs prometheus-prometheus-1 | ./kubectl-log-insight --model qwen/qwen3.5-9b --base-url http://localhost:1234/v1 --max-tokens 8192 -q "do you find any problem"
⚠  Logs truncated to 8192 tokens (oldest lines removed)


Yes, I found several significant problems in these logs:

## Critical Issues

### 1. **Rule Evaluation Failures** (Most Severe)
Multiple recording rules are consistently failing with the error:
```
"vector contains metrics with the same labelset after applying rule labels"
```

**Affected Rules:**
- `node_filesystem_size_bytes` (2 occurrences)
- `node_network_transmit_errs_total` (3 occurrences)
- `node_timex_maxerror_seconds` (2 occurrences)
- `node_timex_offset_seconds` (2 occurrences)
- `node_network_receive_errs_total` (3 occurrences)
- `node_filesystem_free_bytes` (3 occurrences)
- `node_filesystem_readonly` (3 occurrences)
- `node_filesystem_avail_bytes` (3 occurrences)

**Root Cause:** These rules are trying to add the `grafanastack: xxxx` label, but multiple metrics with different values end up with identical labelsets after transformation. Prometheus doesn't allow duplicate time series with the same labels.

### 2. **Sample Dropped from Ingestion Errors**
```
Error on ingesting results from rule evaluation with different value but same timestamp
num_dropped=3
```

This occurs for: `coredns_dns_responses_total:sum_rate2m`
- 3 samples were dropped due to conflicting values at the same timestamp

## Issues Found

| Issue Type | Severity | Frequency | Impact |
|------------|----------|-----------|--------|
| Recording rule failures | CRITICAL | High (10+ failed evaluations) | Metrics not being recorded |
| Duplicate label sets | CRITICAL | Persistent across multiple rules | Data ingestion broken |
| Sample drops | MEDIUM | 3 samples dropped per cycle | Some metrics missing |

## Recommended Actions

1. **Fix Recording Rules** - Add unique identifiers (like node labels, pod names, or instance IDs) to ensure each rule produces unique labelsets
2. **Review `schip_pod_info` join** - The rules joining on pods need unique grouping keys
3. **Use Different PromQL Strategies** - Consider aggregating by node before applying labels

These issues are causing Prometheus to fail to properly record metrics from your node_exporter and coredns recording rules.
⏱  57.8s
```