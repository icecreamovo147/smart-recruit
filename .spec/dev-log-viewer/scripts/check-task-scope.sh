#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/dev-log-viewer"
STATE_FILE="${FEATURE_DIR}/pipeline-state.json"
TASK_ID="${1:-}"
BASE_TREE="${TASK_BASE_TREE:-}"

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required" >&2
  echo "usage: TASK_BASE_TREE=<tree> bash $0 <TASK-ID>" >&2
  exit 2
fi

if [ -z "${BASE_TREE}" ] && [ -f "${STATE_FILE}" ]; then
  BASE_TREE="$(node -e "const s=require(process.argv[1]); process.stdout.write(s.task_runs?.[process.argv[2]]?.base_tree || '')" "${STATE_FILE}" "${TASK_ID}")"
fi

if [ -z "${BASE_TREE}" ]; then
  echo "ERROR: no reliable TASK base_tree is recorded for ${TASK_ID}" >&2
  exit 2
fi

exec node "${ROOT}/.agents/skills/spec-harness/scripts/check-task-scope.mjs" \
  --root "${ROOT}" \
  --feature-dir "${FEATURE_DIR}" \
  --task "${TASK_ID}" \
  --base-tree "${BASE_TREE}"
