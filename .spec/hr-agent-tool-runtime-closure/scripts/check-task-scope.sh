#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/hr-agent-tool-runtime-closure"
TASK_ID="${1:-}"
BASE_TREE="${TASK_BASE_TREE:-}"

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required" >&2
  exit 2
fi
if [ -z "${BASE_TREE}" ] && [ -f "${FEATURE_DIR}/pipeline-state.json" ]; then
  BASE_TREE="$(node -e "const s=require(process.argv[1]); process.stdout.write(s.task_runs?.[process.argv[2]]?.base_tree || '')" "${FEATURE_DIR}/pipeline-state.json" "${TASK_ID}")"
fi
if [ -z "${BASE_TREE}" ]; then
  echo "ERROR: no reliable TASK base_tree for ${TASK_ID}" >&2
  exit 2
fi

exec node "${ROOT}/.agents/skills/spec-harness/scripts/check-task-scope.mjs" \
  --root "${ROOT}" --feature-dir "${FEATURE_DIR}" --task "${TASK_ID}" --base-tree "${BASE_TREE}"
