#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/hr-agent-tool-runtime-closure"
cd "${ROOT}"

node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}" --require-pipeline
bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
bash -n "${FEATURE_DIR}/scripts/agent-check.sh"
git diff --check
