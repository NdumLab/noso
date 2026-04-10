#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."
TOOLS_GO="${ROOT}/.tools/go/bin/go"

# Use the bundled Go toolchain if present, otherwise fall back to PATH.
if [[ -x "${TOOLS_GO}" ]]; then
    GO="${TOOLS_GO}"
elif command -v go &>/dev/null; then
    GO="go"
else
    echo "error: go not found in .tools/go/bin or PATH" >&2
    exit 1
fi

# Inject version metadata at link time so the binary reports real values.
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

LDFLAGS="-X github.com/NdumLab/noso/pkg/buildinfo.Version=${VERSION} \
         -X github.com/NdumLab/noso/pkg/buildinfo.Commit=${COMMIT} \
         -X github.com/NdumLab/noso/pkg/buildinfo.Date=${DATE}"

OUT="${ROOT}/bin/cli-helper"
mkdir -p "${ROOT}/bin"

echo "Building cli-helper → ${OUT}"
"${GO}" build -ldflags "${LDFLAGS}" -o "${OUT}" "${ROOT}/cmd/cli-helper"
echo "Done. Version: $("${OUT}" version)"
