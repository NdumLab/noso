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

```bash
# amd64
curl -Lo cli-helper https://github.com/<owner>/noso/releases/latest/download/cli-helper-linux-amd64
chmod +x cli-helper
sudo mv cli-helper /usr/local/bin/
```

```bash
# arm64
curl -Lo cli-helper https://github.com/<owner>/noso/releases/latest/download/cli-helper-linux-arm64
chmod +x cli-helper
sudo mv cli-helper /usr/local/bin/
```

Verify the download matches the published checksum:

```bash
curl -sL https://github.com/<owner>/noso/releases/latest/download/SHA256SUMS | sha256sum --check --ignore-missing
```

### Build from source

Requires Go 1.22 or later.

```bash
git clone https://github.com/<owner>/noso
cd noso
go build -o cli-helper ./cmd/cli-helper
```

## Quick start

```bash
# Ask a plain-English question
cli-helper "show disk free space"
cli-helper "what is using all the memory"
cli-helper "nginx is not starting"

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
# version=v1.2.0 commit=abc1234 date=2026-04-10T12:00:00Z go=go1.22.0
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
  "audit_log_path": "/home/user/.local/state/noso/audit.log"
}
```

| Field | Values | Default | Description |
|-------|--------|---------|-------------|
| `mode` | `strict-local`, `local-preferred` | `strict-local` | Controls how evidence is gathered |
| `audit_log_path` | any writable path | `~/.local/state/noso/audit.log` | JSONL file for query history |

Environment variable overrides:

| Variable | Overrides |
|----------|-----------|
| `NOSO_MODE` | `mode` |
| `NOSO_AUDIT_LOG_PATH` | `audit_log_path` |

### Audit log

Every query is appended as a JSON line to the audit log. The log directory is created with permissions `0700` and files are written at `0600`. Set `NOSO_AUDIT_LOG_PATH=/dev/null` to disable logging entirely.

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

## Coverage and CI

The test suite runs with the race detector and enforces a 65% statement-coverage floor. Release binaries are built for `linux/amd64` and `linux/arm64` on every version tag.

```
go test -race ./...
```

## License

MIT — see [LICENSE](LICENSE).
