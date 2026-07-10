#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/populate-development-agent-knowledge"

cd "${ROOT}"

echo "==> validate feature Harness structure"
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}"

echo "==> validate shell syntax"
bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
bash -n "${FEATURE_DIR}/scripts/agent-check.sh"

echo "==> validate patch whitespace"
git diff --check

echo "==> validate knowledge script syntax"
for script in .knowledge/scripts/*.mjs; do
  [ -e "${script}" ] || continue
  node --check "${script}"
done

echo "==> run knowledge tool tests"
node .knowledge/scripts/knowledge-validator.test.mjs

echo "==> validate knowledge repository"
node .knowledge/scripts/validate-knowledge.mjs --root .

echo "==> validate knowledge references"
node .knowledge/scripts/check-references.mjs --root .

echo "agent-check: PASS"
