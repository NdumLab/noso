# noso

`cli-helper` is a plain-English terminal assistant for Linux and DevOps operators. Ask it a question in natural language and it returns the right command to run — with a risk rating, confidence score, expected output, and follow-up suggestions, all sourced from your local environment rather than the network.

```
$ cli-helper "what process is using port 8080"

Command:      ss -ltnp | grep :8080
Explanation:  Inspect listeners on TCP port 8080 ...
Risk:         Low
Confidence:   High
Verified from: exec.LookPath, ss --help
Next step:    Run journalctl -u <service> if the owning process is a systemd unit
```

## Install

### Download a release binary (Linux, no Go required)

Choose the binary for your architecture:

- **amd64** — most Intel and AMD 64-bit Linux systems (servers, desktops, most cloud VMs)
- **arm64** — ARM 64-bit systems (AWS Graviton, Raspberry Pi 4+, Ampere)

```bash
# Intel/AMD 64-bit
curl -Lo cli-helper https://github.com/NdumLab/noso/releases/latest/download/cli-helper-linux-amd64

# Verify the checksum before installing
curl -sL https://github.com/NdumLab/noso/releases/latest/download/SHA256SUMS \
  | sha256sum --check --ignore-missing

chmod +x cli-helper
sudo mv cli-helper /usr/local/bin/
```

```bash
# ARM 64-bit
curl -Lo cli-helper https://github.com/NdumLab/noso/releases/latest/download/cli-helper-linux-arm64

# Verify the checksum before installing
curl -sL https://github.com/NdumLab/noso/releases/latest/download/SHA256SUMS \
  | sha256sum --check --ignore-missing

chmod +x cli-helper
sudo mv cli-helper /usr/local/bin/
```

The checksum step downloads the `SHA256SUMS` file published alongside each release and checks that your downloaded binary matches. You should see `cli-helper-linux-amd64: OK` (or `arm64`) before proceeding. If the check fails, delete the file and re-download.

### Install with `go install`

Requires Go 1.22 or later.

```bash
go install github.com/NdumLab/noso/cmd/cli-helper@latest
```

### Build from source

Requires Go 1.22 or later.

```bash
git clone https://github.com/NdumLab/noso
cd noso
go build -o cli-helper ./cmd/cli-helper
sudo mv cli-helper /usr/local/bin/
```

## Quick start

```bash
# Ask a plain-English question
cli-helper "show disk free space"
cli-helper "what is using all the memory"
cli-helper "nginx is not starting"
cli-helper troubleshoot "why is worker 2 not up?"

# Prefix with 'ask' if you prefer explicit subcommands
cli-helper ask "show pods in namespace prod"

# Get JSON output for scripting
cli-helper --json "git log"

# Suppress warnings and next-step hints
cli-helper --quiet "show disk free space"
```

## Subcommands

### `ask` — query mode (default)

Translate a plain-English question into a safe command with context.

```bash
cli-helper "what process is on port 443"
cli-helper ask "show kubernetes deployments in namespace staging"
cli-helper ask "explain terraform destroy"
```

### `interpret` — output interpretation

Pipe or paste captured command output for analysis.

```bash
df -h | cli-helper interpret --command "df -h"
cli-helper interpret --command "free -h" --input "$(free -h)"
cli-helper interpret --command "kubectl get pods -n prod" --input "$(kubectl get pods -n prod)"
```

### `doctor` — environment check

Inspect your local environment for common issues and missing tools.

```bash
cli-helper doctor
cli-helper doctor --json
```

### `env` — environment snapshot

Print detected OS, package manager, shell, and tool availability.

```bash
cli-helper env
cli-helper env --json
```

### `history` — audit log viewer

Browse or filter past queries from the local audit log.

```bash
cli-helper history
cli-helper history --limit 20 --match kubectl
cli-helper history --json
```

### `runbook` — generate a runbook

Build a structured runbook from your audit history.

```bash
cli-helper runbook --limit 20 --format markdown --output incident.md
cli-helper runbook --match nginx --format json
```

### `version`

```bash
cli-helper version
# version=v0.1.0 commit=abc1234 date=2026-04-10T12:00:00Z go=go1.22.5
```

### `completion` — shell completion

```bash
# bash
cli-helper completion bash > /etc/bash_completion.d/cli-helper
source /etc/bash_completion.d/cli-helper

# zsh
cli-helper completion zsh > "${fpath[1]}/_cli-helper"

# fish
cli-helper completion fish > ~/.config/fish/completions/cli-helper.fish
```

## Configuration

Configuration is optional. Defaults work out of the box.

| Source | Path |
|--------|------|
| Config file | `$XDG_CONFIG_HOME/noso/config.json` or `~/.config/noso/config.json` |
| Override via env | `NOSO_CONFIG=/path/to/config.json` |

```json
{
  "mode": "strict-local",
  "audit_log_path": "/home/user/.local/state/noso/audit.log",
  "llm_enabled": false,
  "llm_endpoint": "http://127.0.0.1:15321/v1/interpret",
  "llm_timeout_ms": 1500
}
```

| Field | Values | Default | Description |
|-------|--------|---------|-------------|
| `mode` | `strict-local`, `local-preferred` | `strict-local` | Controls how evidence is gathered |
| `audit_log_path` | any writable path | `~/.local/state/noso/audit.log` | JSONL file for query history |
| `llm_enabled` | `true`, `false` | `false` | Enables optional local fallback interpretation for ambiguous or unsupported queries |
| `llm_endpoint` | local HTTP URL | `http://127.0.0.1:15321/v1/interpret` | Endpoint for the separate `noso-llm` service |
| `llm_timeout_ms` | positive integer | `1500` | Timeout for local LLM fallback requests |

Environment variable overrides:

| Variable | Overrides |
|----------|-----------|
| `NOSO_MODE` | `mode` |
| `NOSO_AUDIT_LOG_PATH` | `audit_log_path` |
| `NOSO_LLM_ENABLED` | `llm_enabled` |
| `NOSO_LLM_ENDPOINT` | `llm_endpoint` |
| `NOSO_LLM_TIMEOUT_MS` | `llm_timeout_ms` |
| `NOSO_LLM_LOG_PATH` | `llm_log_path` |

### Audit log

Every query is appended as a JSON line to the audit log. The log directory is created with permissions `0700` and files are written at `0600`. Set `NOSO_AUDIT_LOG_PATH=/dev/null` to disable logging entirely.

### Doctor and local LLM health

When `NOSO_LLM_ENABLED=1`, `cli-helper doctor` also probes the local LLM health endpoint derived from `llm_endpoint`. A healthy fallback is reported in the doctor summary; timeout, availability, transient upstream, and invalid-response failures are surfaced as doctor warnings with a next step to check `/health` or disable fallback temporarily.

### Local LLM log inspection

When `NOSO_LLM_LOG_PATH` or `llm_log_path` is configured, `cli-helper llm-log` renders recent local LLM fallback events without requiring manual `tail` or `jq` work.

```bash
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --limit 10
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --match timeout --json
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --since 2h --match transient
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --provider ollama --error-only
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --clarification-only
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --stats
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --stats --format markdown --output llm-summary.md
NOSO_LLM_LOG_PATH=/var/log/noso-llm.jsonl cli-helper llm-log --provider ollama --error-only --format csv --output llm-errors.csv
```

`--since` accepts either an RFC3339 timestamp or a relative duration like `15m`, `2h`, or `24h`.
`--provider` narrows entries to a specific backend such as `heuristic` or `ollama`, and `--error-only` shows only failed fallback events.
`--clarification-only` isolates ambiguous-query cases where the local LLM asked for more specificity instead of returning a direct candidate.
`--stats` renders aggregate counts by provider, error type, and top intent instead of raw entries.
`--format` supports `text`, `json`, `markdown`, and `csv`, and `--output` writes the rendered entries or summary to a file for incident notes or spreadsheet import.

### Optional local LLM fallback

`cli-helper` can call a separate local service when the rule-based registry cannot confidently classify a query. The local service only returns structured intent candidates or a clarification question; `cli-helper` still owns command generation, evidence, and risk labels.

```bash
go build -o bin/noso-llm ./cmd/noso-llm
./bin/noso-llm -listen 127.0.0.1:15321

NOSO_LLM_ENABLED=1 cli-helper "why is worker 2 not up?"
```

This keeps the LLM optional and replaceable. If the fallback service is unavailable, `cli-helper` falls back to normal deterministic behavior.

For ambiguous outage-style questions, `cli-helper troubleshoot` builds a ranked, read-only plan across likely object types such as services, containers, and pods, then picks the safest first probe instead of returning `unsupported_query`.

```bash
cli-helper troubleshoot "why is worker 2 not up?"
```

For service-style outages, `troubleshoot` also runs the first low-risk probe locally, interprets the result, and folds that evidence back into the explanation and next steps. The current evidence loop covers:

- `systemctl status` with optional `journalctl -u <service> -n 50 --no-pager`
- `docker ps -a` or `podman ps -a` with optional container logs when the target appears exited or unhealthy
- `kubectl get pods` with optional `kubectl describe pod` and `kubectl logs` when an unhealthy pod is visible

If the required local tool is missing, `troubleshoot` reports that the live probe was unavailable instead of fabricating evidence.

Live probe results are now surfaced in a dedicated `Findings` section instead of being folded into freeform explanation text. That keeps the top-level reasoning stable while making machine- and human-scannable evidence easier to extend.

`troubleshoot` also keeps a lightweight local thread so repeated runs of the same outage question can advance to the next unread probe instead of repeating the same first command every time. The default thread file lives under the local state directory and can be overridden with `NOSO_TROUBLESHOOT_STATE_PATH`.

Branch selection now uses prior findings too. For example, if the first `systemctl status` result shows that the unit does not exist, the next run will prefer runtime or Kubernetes probes over more systemd-specific follow-ups.

Those branch preferences are now persisted as family scores in the local troubleshoot thread, so repeated runs can keep strengthening or weakening service, runtime, and Kubernetes hypotheses instead of recalculating from scratch every time.

Each troubleshoot thread now also keeps a probe history with timestamps, commands, summaries, findings, and warnings. Operators can inspect or clear that state directly from the CLI instead of editing the state file by hand:

```bash
cli-helper troubleshoot-history --query "why is worker 2 not up?"
cli-helper troubleshoot-reset --query "why is worker 2 not up?"
```

You can also run `noso-llm` against a real local model runtime through Ollama while keeping the same JSON contract:

```bash
ollama serve
ollama pull qwen2.5:7b-instruct

./bin/noso-llm \
  -provider ollama \
  -model qwen2.5:7b-instruct \
  -ollama-endpoint http://127.0.0.1:11434/api/chat
```

The `cli-helper` side does not change. It still talks to `llm_endpoint`, and `noso-llm` decides whether to use the built-in heuristic backend or Ollama.

`noso-llm` also exposes lightweight observability:

- `GET /health` for provider and model status
- `GET /metrics` for request, clarification, retry, and error counters
- `-log-path /path/to/noso-llm.jsonl` for append-only JSONL request summaries

Example:

```bash
./bin/noso-llm \
  -provider ollama \
  -model qwen2.5:7b-instruct \
  -ollama-endpoint http://127.0.0.1:11434/api/chat \
  -log-path /var/log/noso-llm.jsonl

curl http://127.0.0.1:15321/metrics
```

For upstream model backends, `noso-llm` validates and ranks responses before returning them to `cli-helper`:

- unsupported intents are dropped
- tool hints are cleared if they are not present in the detected environment
- malformed or empty clarification responses are rejected
- candidates are sorted by confidence and trimmed to the requested limit

If the provider returns unusable output, `cli-helper` falls back to its normal deterministic unsupported-query behavior instead of trusting the model.

Transient fallback failures are handled separately from unsupported queries:

- local timeout errors are surfaced as timeout warnings
- unavailable local endpoints are surfaced as availability warnings
- Ollama transport failures and `429` or `5xx` responses are retried briefly before being reported
- invalid upstream payloads are rejected and treated as fallback failures, not as trusted intent output

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Internal or configuration error |
| `2` | Bad arguments or missing required flag |
| `3` | Query received but no intent matched |
| `4` | Required tool not found on this host (reserved) |

## Security

`cli-helper` is a read-only advisory tool. It does not run the commands it suggests. Evidence about installed tools is gathered through `exec.LookPath` and direct binary invocation — no shell intermediary is used. See [SECURITY.md](SECURITY.md) for the vulnerability reporting process and full security design notes.

## Contributing

Release binaries for `linux/amd64` and `linux/arm64` are published automatically on every version tag. The test suite enforces a 65% statement-coverage floor and runs with the race detector.

```bash
git clone https://github.com/NdumLab/noso
cd noso
go test ./...
```

## License

MIT — see [LICENSE](LICENSE).
