#!/usr/bin/env bash
set -euo pipefail

echo "Running lightweight repository checks for agent-skill-review-fixes..."

git diff --check

if command -v pnpm >/dev/null 2>&1; then
  pnpm --filter hr-frontend typecheck
else
  echo "pnpm not found; skipping hr-frontend typecheck" >&2
fi

if command -v go >/dev/null 2>&1; then
  (cd logic-grpc-service && go test ./...)
else
  echo "go not found; skipping logic-grpc-service tests" >&2
fi

echo "agent-check completed."
