#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."

# Build first.
bash "${SCRIPT_DIR}/build.sh"

INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
mkdir -p "${INSTALL_DIR}"

cp "${ROOT}/bin/cli-helper" "${INSTALL_DIR}/cli-helper"
chmod 755 "${INSTALL_DIR}/cli-helper"
echo "Installed cli-helper → ${INSTALL_DIR}/cli-helper"

# Install bash completion if the directory is writable.
COMPLETION_DIR="${BASH_COMPLETION_DIR:-/etc/bash_completion.d}"
if [[ -d "${COMPLETION_DIR}" && -w "${COMPLETION_DIR}" ]]; then
    "${INSTALL_DIR}/cli-helper" completion bash > "${COMPLETION_DIR}/cli-helper"
    echo "Installed bash completion → ${COMPLETION_DIR}/cli-helper"
else
    echo "Tip: run 'cli-helper completion bash > ~/.bash_completion' to enable tab completion."
fi
