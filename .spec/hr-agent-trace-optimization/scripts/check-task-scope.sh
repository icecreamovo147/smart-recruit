#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/hr-agent-trace-optimization"
TASK_ID="${1:-}"
BASE_TREE="${TASK_BASE_TREE:-}"

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required" >&2
  echo "usage: TASK_BASE_TREE=<tree> bash $0 <TASK-ID>" >&2
  exit 2
fi

if [ -z "${BASE_TREE}" ]; then
  BASE_TREE="$(git -C "${ROOT}" rev-parse HEAD^{tree})"
  echo "WARN: TASK_BASE_TREE not set; using HEAD tree ${BASE_TREE}" >&2
fi

exec node "${ROOT}/.agents/skills/spec-harness/scripts/check-task-scope.mjs" \
  --root "${ROOT}" \
  --feature-dir "${FEATURE_DIR}" \
  --task "${TASK_ID}" \
  --base-tree "${BASE_TREE}"
