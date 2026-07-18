#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="$ROOT/.spec/github-ci-current-state-refresh"

cd "$ROOT"

echo "==> validate feature Harness structure"
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature "$FEATURE_DIR"

echo "==> validate shell syntax"
bash -n "$FEATURE_DIR/scripts/check-task-scope.sh"
bash -n "$FEATURE_DIR/scripts/agent-check.sh"

echo "==> check workflow stale paths"
if rg -n "logic-grpc-service|web-gin-service" .github/workflows; then
  echo "ERROR: stale service paths remain in workflows" >&2
  exit 1
fi

echo "==> validate GitHub knowledge workflow equivalents"
node .knowledge/scripts/knowledge-validator.test.mjs
node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/development-agent-knowledge-base
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/knowledge-base-current-state-refresh
node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/github-ci-current-state-refresh
node .agents/skills/spec-harness/scripts/validator.test.mjs

echo "==> validate patch whitespace"
git diff --check

echo "agent-check: PASS"
