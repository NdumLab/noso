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
cli-helper incident-status --query "why is worker 2 not up?"
cli-helper incident-history --status open
cli-helper incident-ingest --query "api availability alert" --source alertmanager --severity critical --summary "API error rate above threshold" --label service=api --label namespace=prod
cli-helper incident-ingest --input alert.json
cat alert.json | cli-helper incident-ingest --input -
cli-helper incident-observe --query "why is worker 2 not up?"
cli-helper incident-resolve --query "why is worker 2 not up?" --summary "Deployment image pull secret fixed"
```

Troubleshoot threads now also persist likely root-cause scores inferred from live findings. That lets repeated runs converge toward explanations such as a missing systemd unit, a crashing container, a permission failure, a database dependency problem, or a Kubernetes CrashLoopBackOff instead of only moving through probe families.

Those ranked causes also shape the follow-up guidance. When one cause becomes dominant, `troubleshoot` now appends targeted next checks for that cause instead of only restating generic branch probes.

Later evidence can also retire stale causes. For example, if one run looks like a generic service startup failure but the next probe proves the unit does not exist, the service-failure cause is down-ranked out of the thread and the missing-unit explanation takes over.

Explicit operator clarifications now reweight the active troubleshoot thread immediately. If the next message is something like `it's actually a pod`, `it's actually a container`, or `it's actually a service`, `troubleshoot` reuses the latest thread, pivots the branch selection to the clarified object family, and retires incompatible causes instead of treating that clarification as a brand-new outage.

Before deeper probing, the planner now also tries to discover whether the inferred target already exists as a systemd unit, a container name, or a Kubernetes pod. Those existence signals are used to bias the first branch, so `troubleshoot` prefers discovered objects over purely linguistic guesses whenever the local tools can confirm them safely.

That discovery signal is now rendered directly in the CLI output under a `Discovery` section, for example when `troubleshoot` finds no matching systemd unit or does find a matching pod name before it runs the first deeper probe.

Those discovery negatives are also persisted in the troubleshoot thread. If the first run shows `No matching systemd unit name found`, the next run will skip back toward runtime or Kubernetes probes sooner instead of re-centering on the already disproven service branch.

When there is no exact local match, the same discovery section now also shows a few nearby object names such as `worker@2`, `worker-2`, or `worker2-api` when those are present locally. That gives the operator immediate candidate names to check before continuing with a wrong target.

Those nearby names now also turn into explicit follow-up suggestions. If `troubleshoot` sees a close pod, container, or unit name, it adds a direct corrected-target command suggestion like `kubectl describe pod worker-2` or `systemctl status worker@2 --no-pager -l` so the operator can pivot immediately.

If the next troubleshoot query uses one of those discovered nearby names, the active thread now adopts that corrected target directly instead of continuing to plan around the older inferred name. That keeps follow-up probes grounded on the closest real object the host or cluster actually exposed.

When that happens, the CLI now prints an explicit `Adopted target:` line so the operator can see that the active thread pivoted to a discovered nearby object instead of silently inferring it from the command alone.

That adoption is now treated as a real target correction, not just a display hint. Once the thread pivots, stale discovery, findings, and incompatible cause follow-ups from the older inferred target are retired so the next reasoning cycle is centered on the adopted object.

The active thread can now also absorb lightweight context refinements without starting over. Follow-ups like `worker-2 in namespace prod` or `it is podman` update the current target context so the next planned probe uses the corrected namespace or runtime identity instead of re-deriving everything from the original wording.

Those refinements are now checked against live inventory too. When `kubectl` is available, `troubleshoot` can confirm whether the matching pod name actually exists in the requested namespace or only in other namespaces. When a runtime hint like `podman` or `docker` is present, the discovery output now confirms whether that runtime is actually available on the host before it biases the next probe.

The live evidence loop now keeps that namespace context too. Once a namespaced pod is confirmed, the evidence follow-ups generated from `kubectl describe pod`, `kubectl logs`, and `kubectl get events` stay scoped to that namespace instead of falling back to cluster-wide or placeholder commands.

The evidence loop also now correlates some dependency failures across object boundaries. If pod logs, container logs, or journal output point to database connectivity or DNS resolution failures, `troubleshoot` adds the next read-only infrastructure probes automatically, such as `dig +short` or `nslookup` for the dependency hostname and listener validation guidance for the expected upstream port.

When the logs include the concrete upstream hostname, those follow-ups now use it directly. For example, a log line mentioning `db.internal` or `api.internal` will now produce `dig +short db.internal` rather than a generic placeholder probe.

When the logs also include an explicit upstream port, `troubleshoot` now carries that forward too. A message like `db.internal port 5432` now produces a concrete socket probe such as `nc -vz db.internal 5432` instead of only generic listener guidance.

Those infrastructure probes are now also specialized to the current host. If `dig` is missing but `nslookup` exists, `troubleshoot` rewrites the DNS probe accordingly. If `nc` is unavailable but the shell is `bash`, it falls back to a `</dev/tcp/...>` reachability check instead of keeping a probe the host cannot run.

For active Kubernetes threads, that specialization now also distinguishes node-local and in-cluster checks. Dependency probes derived from pod-local evidence are rewritten into `kubectl exec` follow-ups against the current pod context instead of assuming the check belongs on the node.

When pod describe output reveals a specific failing container name, those in-cluster probes now include `-c <container>` as well. That now covers both `kubectl exec` dependency checks and `kubectl logs` follow-ups, which matters for multi-container pods where the evidence loop needs to stay in the same container context that emitted the original failure.

That container hint no longer depends only on the `Containers:` section of `kubectl describe pod`. The interpreter now also extracts container names from Kubernetes event text and pod-log preambles such as `Defaulted container "api" out of: ...`, so the troubleshoot thread can stay container-aware even when the first useful signal comes from events or logs.

`kubectl get events` is now interpreted directly too. When recent event rows identify a failing pod and container, the live troubleshoot path can move straight from the event stream into `kubectl logs -c <container> --previous` instead of dropping back to a generic pod-only follow-up.

That event interpretation is now reason-aware as well. Restart/backoff events still lead to previous-container logs, but image-pull, scheduling, and mount failures now branch into more appropriate follow-ups such as workload image/imagePullSecrets review, scheduler-capacity checks, and PVC or mount inspection instead of pretending container logs are always the next best probe.

Image-pull event handling is now concrete when the event stream includes the actual image reference. If the row mentions something like `ghcr.io/...` or `registry.internal:5000/...`, `noso` now extracts that registry endpoint and adds exact DNS and socket probes for the registry host instead of leaving connectivity checks generic.

Scheduler event handling is now more specific too. `FailedScheduling` rows are no longer treated as a generic capacity problem: `noso` now distinguishes memory pressure, CPU pressure, taint or toleration mismatches, node-affinity mismatches, and unbound PVC scheduling blockers, and tailors the next checks accordingly.

Mount and storage failures are now more concrete as well. When `FailedMount` or related event text names a missing PVC, secret, or config map, `noso` now carries that exact object name into the next step instead of only saying “inspect storage.”

Those storage follow-ups are now rendered as concrete read-only commands too. For example, a missing PVC, Secret, or ConfigMap in the event stream now becomes a specific `kubectl describe pvc|secret|configmap -n <namespace> <name>` step rather than a prose-only hint.

When scheduler messages include a concrete node name, `noso` now uses that too. Instead of only saying “check capacity” or “review taints,” it adds an exact `kubectl describe node <name>` follow-up so the operator can inspect allocatable resources, labels, taints, and conditions on the named candidate node.

Those Kubernetes infrastructure objects are now stateful troubleshoot targets too. If an earlier run surfaced a PVC, Secret, ConfigMap, or node as the next concrete object to inspect, a follow-up query that names that object can now pivot the active troubleshoot thread directly to the matching `kubectl describe ...` command instead of re-running the generic outage classifier.

That continuity now extends through refinement as well. Once the active thread is already on a PVC, Secret, ConfigMap, or node, follow-up queries keep that object type and namespace instead of collapsing back into a pod-first outage guess.

The same owner-object promotion now also covers higher-level Kubernetes workload objects discovered in event text. When the event stream points at a Deployment or Service, `troubleshoot` now surfaces exact `kubectl describe deployment ...` or `kubectl describe service ...` follow-ups, and repeated queries can adopt those objects into the active thread just like pods, PVCs, and nodes.

Troubleshoot runs now also update a lightweight incident record. Each incident tracks the original query, open or resolved status, current target, latest probe, likely causes, next steps, and probe history so operators can come back to an outage without re-reading raw troubleshoot output.

Use the incident commands to inspect or close those records directly:

```bash
cli-helper incident-status --query "why is worker 2 not up?"
cli-helper incident-history --status open
cli-helper incident-ingest --query "api availability alert" --source alertmanager --severity critical --summary "API error rate above threshold" --label service=api --label namespace=prod
cli-helper incident-observe --query "why is worker 2 not up?"
cli-helper incident-resolve --query "why is worker 2 not up?" --summary "Deployment image pull secret fixed"
```

`incident-ingest` opens or updates an incident from a structured external signal. It is a local ingest path for alert sources like Alertmanager, schedulers, or other automation that can call the CLI with a stable query or fingerprint plus metadata such as source, severity, summary, and labels.

It now supports both flat flags and payload ingestion:

```bash
cli-helper incident-ingest --input alert.json
cat alert.json | cli-helper incident-ingest --input -
```

Supported payloads:
- native `noso` alert JSON using the `query`, `source`, `severity`, `summary`, `fingerprint`, and `labels` fields
- an Alertmanager-style envelope with `alerts`, `commonLabels`, and `commonAnnotations`

Incident ingest now also performs simple correlation-aware dedup. When alerts share the same source plus workload labels like `cluster`, `namespace`, `service`, `deployment`, `pod`, `host`, `instance`, `job`, or `alertname`, `noso` folds them into the same incident and increments the incident’s alert count instead of opening a parallel duplicate record for every slightly different alert summary.

Incident ingest now also promotes workload labels into incident targeting. If an alert already names a `pod`, `deployment`, `service`, `persistentvolumeclaim`/`pvc`, `secret`, `configmap`, `node`, `host`, or `instance`, the incident opens with that active target and an initial read-only probe such as `kubectl describe pod ...` or `kubectl describe service ...` already queued for `incident-observe`.

`incident-observe` is the first policy-controlled execution surface in `noso`. It only runs explicit, low-risk, read-only probes from the incident’s queued next steps, and it refuses mutation-oriented commands even if they appear in the incident guidance. Today that allowlist covers observation commands such as `systemctl status`, `journalctl -u`, `docker|podman ps`, `docker|podman logs`, `kubectl get|describe|logs`, `dig`, `nslookup`, `nc -vz`, and `ss -ltnp`.

You can also let it advance through multiple approved probes in one pass:

```bash
cli-helper incident-observe --query "why is worker 2 not up?" --max-steps 3
```

That loop stays bounded and read-only. It carries unread approved next steps forward between probe executions, updates the incident record after each step, and stops when it runs out of allowed probes or reaches the requested step limit.

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
