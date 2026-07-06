#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "${ROOT}" || exit 1

echo "Changed files in current git diff:"
git diff --name-only

echo ""
echo "Changed staged files:"
git diff --cached --name-only

echo ""
echo "Untracked files:"
git ls-files --others --exclude-standard

echo ""
echo "Please compare the files above with:"
echo ".spec/bailian-embedding-provider/task-scope.json"
echo ""
echo "This script reports the diff surface only. The implementing Agent must manually verify the active TASK's allowedFiles and forbiddenFiles."
