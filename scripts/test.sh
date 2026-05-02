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

GO_BIN_DIR="$(dirname "${GO}")"
export PATH="${GO_BIN_DIR}:${PATH}"

CACHE_ROOT="${ROOT}/.cache/go"
mkdir -p "${CACHE_ROOT}/build" "${CACHE_ROOT}/mod"

export GOCACHE="${GOCACHE:-${CACHE_ROOT}/build}"
export GOMODCACHE="${GOMODCACHE:-${CACHE_ROOT}/mod}"

CGO_ENABLED_VALUE="$("${GO}" env CGO_ENABLED 2>/dev/null || echo 0)"

if [[ "${CGO_ENABLED_VALUE}" == "1" ]]; then
    echo "Running tests with race detector..."
    "${GO}" test -race -cover ./... "${@}"
else
    echo "Running tests without race detector (CGO_ENABLED=${CGO_ENABLED_VALUE})..."
    "${GO}" test -cover ./... "${@}"
fi
echo "Running vet..."
"${GO}" vet ./...
echo "All checks passed."
