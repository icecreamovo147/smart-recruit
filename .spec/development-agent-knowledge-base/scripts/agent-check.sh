#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/development-agent-knowledge-base"

cd "${ROOT}"

echo "==> validate feature Harness structure"
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "${FEATURE_DIR}"

echo "==> validate shell syntax"
bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
bash -n "${FEATURE_DIR}/scripts/agent-check.sh"

echo "==> validate patch whitespace"
git diff --check

if [ -d .knowledge/scripts ]; then
  echo "==> validate knowledge script syntax"
  for script in .knowledge/scripts/*.mjs; do
    [ -e "${script}" ] || continue
    node --check "${script}"
  done
fi

if [ -f .knowledge/scripts/knowledge-validator.test.mjs ]; then
  echo "==> run knowledge tool tests"
  node .knowledge/scripts/knowledge-validator.test.mjs
fi

if [ -f .knowledge/scripts/validate-knowledge.mjs ]; then
  echo "==> validate knowledge repository"
  node .knowledge/scripts/validate-knowledge.mjs --root .
fi

if [ -f .knowledge/scripts/check-references.mjs ]; then
  echo "==> validate knowledge references"
  node .knowledge/scripts/check-references.mjs --root .
fi

echo "agent-check: PASS"
