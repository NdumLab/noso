#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."

if [[ $# -ne 1 ]]; then
    echo "usage: $0 <service-missing|k8s-crashloop-db|k8s-pending-taint>" >&2
    exit 2
fi

SCENARIO="$1"
STATE_DIR="${ROOT}/.cache/trials/${SCENARIO}"
mkdir -p "${STATE_DIR}"

AUDIT_PATH="${STATE_DIR}/audit.log"
TROUBLESHOOT_PATH="${STATE_DIR}/troubleshoot-state.json"
INCIDENT_PATH="${STATE_DIR}/incident-state.json"

case "${SCENARIO}" in
    service-missing)
        cat <<EOF
Trial scenario: service-missing

Export these variables in your shell:
  export NOSO_AUDIT_LOG_PATH="${AUDIT_PATH}"
  export NOSO_TROUBLESHOOT_STATE_PATH="${TROUBLESHOOT_PATH}"
  export NOSO_INCIDENT_STATE_PATH="${INCIDENT_PATH}"

Suggested commands:
  cli-helper troubleshoot "why is worker 2 not up?"
  cli-helper troubleshoot "why is worker 2 not up?"
  cli-helper incident-status --query "why is worker 2 not up?"

Scenario brief:
  ${ROOT}/trials/scenarios/service-missing.md

Feedback template:
  ${ROOT}/trials/feedback-template.md
EOF
        ;;
    k8s-crashloop-db)
        cat <<EOF
Trial scenario: k8s-crashloop-db

Export these variables in your shell:
  export NOSO_AUDIT_LOG_PATH="${AUDIT_PATH}"
  export NOSO_TROUBLESHOOT_STATE_PATH="${TROUBLESHOOT_PATH}"
  export NOSO_INCIDENT_STATE_PATH="${INCIDENT_PATH}"

Suggested commands:
  cli-helper incident-ingest --input "${ROOT}/trials/fixtures/alerts/k8s-crashloop-db.json"
  cli-helper troubleshoot "worker pod alert"
  cli-helper incident-status --query "worker pod alert"

Scenario brief:
  ${ROOT}/trials/scenarios/k8s-crashloop-db.md

Feedback template:
  ${ROOT}/trials/feedback-template.md
EOF
        ;;
    k8s-pending-taint)
        cat <<EOF
Trial scenario: k8s-pending-taint

Export these variables in your shell:
  export NOSO_AUDIT_LOG_PATH="${AUDIT_PATH}"
  export NOSO_TROUBLESHOOT_STATE_PATH="${TROUBLESHOOT_PATH}"
  export NOSO_INCIDENT_STATE_PATH="${INCIDENT_PATH}"

Suggested commands:
  cli-helper incident-ingest --input "${ROOT}/trials/fixtures/alerts/k8s-pending-taint.json"
  cli-helper troubleshoot "web pod pending"
  cli-helper incident-status --query "web pod pending"

Scenario brief:
  ${ROOT}/trials/scenarios/k8s-pending-taint.md

Feedback template:
  ${ROOT}/trials/feedback-template.md
EOF
        ;;
    *)
        echo "unknown scenario: ${SCENARIO}" >&2
        exit 2
        ;;
esac
