#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TASK_ID="${1:-}"

if [ "$TASK_ID" != "TASK-GCI-001" ]; then
  echo "usage: $0 TASK-GCI-001" >&2
  exit 2
fi

cd "$ROOT"

echo "feature: github-ci-current-state-refresh"
echo "task: $TASK_ID"

violations=0
while IFS= read -r file; do
  case "$file" in
    .github/workflows/ci.yml|.github/workflows/knowledge-validation.yml|.spec/github-ci-current-state-refresh/*)
      echo "ALLOWED $file"
      ;;
    "")
      ;;
    *)
      echo "FORBIDDEN $file" >&2
      violations=1
      ;;
  esac
done < <({ git diff --name-only; git ls-files --others --exclude-standard .spec/github-ci-current-state-refresh; } | sort -u)

if [ "$violations" -ne 0 ]; then
  echo "scope_result: FAIL ($TASK_ID)" >&2
  exit 1
fi

echo "scope_result: PASS ($TASK_ID)"
