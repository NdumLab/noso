#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."
TOOLS_GO="${ROOT}/.tools/go/bin/go"

if [[ -x "${TOOLS_GO}" ]]; then
    GO="${TOOLS_GO}"
elif command -v go &>/dev/null; then
    GO="go"
else
    echo "error: go not found in .tools/go/bin or PATH" >&2
    exit 1
fi

echo "Running tests with race detector..."
"${GO}" test -race -cover ./... "${@}"
echo "Running vet..."
"${GO}" vet ./...
echo "All checks passed."
