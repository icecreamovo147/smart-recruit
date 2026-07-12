#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/backend-ddd-microservices-evolution"
TASK_ID="${1:-}"

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required" >&2
  echo "usage: bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh <TASK-ID>" >&2
  exit 2
fi

BASE_TREE="${TASK_BASE_TREE:-}"
PIPELINE_STATE="${ROOT}/${FEATURE_DIR}/pipeline-state.json"

if [ -z "${BASE_TREE}" ] && [ -f "${PIPELINE_STATE}" ]; then
  BASE_TREE="$(node -e 'const fs=require("fs"); const p=process.argv[1]; const task=process.argv[2]; const s=JSON.parse(fs.readFileSync(p,"utf8")); const run=s.task_runs && s.task_runs[task]; process.stdout.write((run && run.base_tree) || s.base_tree || "");' "${PIPELINE_STATE}" "${TASK_ID}")"
fi

if [ -z "${BASE_TREE}" ]; then
  echo "ERROR: TASK_BASE_TREE or pipeline-state task_runs[${TASK_ID}].base_tree is required" >&2
  echo "Set TASK_BASE_TREE to the reliable pre-task tree captured by spec-harness/harness-pipeline." >&2
  exit 2
fi

node "${ROOT}/.agents/skills/spec-harness/scripts/check-task-scope.mjs" \
  --root "${ROOT}" \
  --feature-dir "${ROOT}/${FEATURE_DIR}" \
  --task "${TASK_ID}" \
  --base-tree "${BASE_TREE}"
