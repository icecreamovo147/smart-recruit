#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR=".spec/recruiting-intelligence-runtime-parity"
TASK_ID="${1:-}"

if [ -z "${TASK_ID}" ]; then
  echo "usage: $0 TASK-ID" >&2
  exit 2
fi

cd "${ROOT}"
BASE_TREE="$(node -e 'const fs=require("fs"); const p=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); const r=p.task_runs?.[process.argv[2]]; if (!r?.base_tree) process.exit(2); process.stdout.write(r.base_tree)' "${FEATURE_DIR}/pipeline-state.json" "${TASK_ID}")"

node .agents/skills/spec-harness/scripts/check-task-scope.mjs \
  --root "${ROOT}" \
  --feature-dir "${FEATURE_DIR}" \
  --task "${TASK_ID}" \
  --base-tree "${BASE_TREE}"
