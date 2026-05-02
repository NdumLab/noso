#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."

if [[ $# -lt 2 || $# -gt 3 ]]; then
    echo "usage: $0 <scenario> <operator-id> [session-label]" >&2
    exit 2
fi

SCENARIO="$1"
OPERATOR_ID="$2"
SESSION_LABEL="${3:-}"

case "${SCENARIO}" in
    service-missing|k8s-crashloop-db|k8s-pending-taint)
        ;;
    *)
        echo "unknown scenario: ${SCENARIO}" >&2
        exit 2
        ;;
esac

sanitize() {
    printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | tr -cs 'a-z0-9._-' '-'
}

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
SAFE_OPERATOR="$(sanitize "${OPERATOR_ID}")"
SAFE_LABEL="$(sanitize "${SESSION_LABEL}")"
SESSION_ID_BASE="${STAMP}-${SAFE_OPERATOR}"
if [[ -n "${SAFE_LABEL}" ]]; then
    SESSION_ID_BASE="${SESSION_ID_BASE}-${SAFE_LABEL}"
fi

SESSION_ID="${SESSION_ID_BASE}"
SESSION_DIR="${ROOT}/trials/sessions/${SESSION_ID}"
suffix=2
while [[ -e "${SESSION_DIR}" ]]; do
    SESSION_ID="${SESSION_ID_BASE}-${suffix}"
    SESSION_DIR="${ROOT}/trials/sessions/${SESSION_ID}"
    suffix=$((suffix + 1))
done
STATE_DIR="${SESSION_DIR}/state"
mkdir -p "${STATE_DIR}"

AUDIT_PATH="${STATE_DIR}/audit.log"
TROUBLESHOOT_PATH="${STATE_DIR}/troubleshoot-state.json"
INCIDENT_PATH="${STATE_DIR}/incident-state.json"
NOTES_PATH="${SESSION_DIR}/feedback.md"
MANIFEST_PATH="${SESSION_DIR}/session.env"

cp "${ROOT}/trials/feedback-template.md" "${NOTES_PATH}"

cat > "${MANIFEST_PATH}" <<EOF
export NOSO_AUDIT_LOG_PATH="${AUDIT_PATH}"
export NOSO_TROUBLESHOOT_STATE_PATH="${TROUBLESHOOT_PATH}"
export NOSO_INCIDENT_STATE_PATH="${INCIDENT_PATH}"
EOF

cat <<EOF
Started operator trial session: ${SESSION_ID}

State exports:
  source "${MANIFEST_PATH}"

Feedback notes:
  ${NOTES_PATH}

Scenario bootstrap:
  ${ROOT}/scripts/run-operator-trial.sh ${SCENARIO}

Recommended next steps:
  1. source "${MANIFEST_PATH}"
  2. bash "${ROOT}/scripts/run-operator-trial.sh" "${SCENARIO}"
  3. Fill in "${NOTES_PATH}" during or immediately after the session
EOF
