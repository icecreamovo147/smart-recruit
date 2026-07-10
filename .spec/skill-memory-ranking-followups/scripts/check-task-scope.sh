#!/usr/bin/env bash
# verify that changes in current git state are within the allowed scope of a TASK.
# usage: bash .spec/skill-memory-ranking-followups/scripts/check-task-scope.sh <TASK-ID>
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TASK_ID="${1:-}"

cd "${ROOT}" || exit 1

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required"
  echo "usage: bash $0 <TASK-ID>"
  exit 2
fi

SCOPE_FILE="${ROOT}/.spec/skill-memory-ranking-followups/task-scope.json"
if [ ! -f "${SCOPE_FILE}" ]; then
  echo "ERROR: task-scope.json not found at ${SCOPE_FILE}"
  exit 2
fi

# Extract allowedFiles and forbiddenFiles for the task.
# Falls back to an empty list if the field or task is missing.
ALLOWED=$(node -e "
const data = require('${SCOPE_FILE}');
const t = data && data.tasks && data.tasks['${TASK_ID}'];
if (!t) { process.exit(3); }
const list = (t.allowedFiles || []).map(f => '|' + f.replace(/[.+*?^$()|[\\]\\\\]/g, '\\\\\$&') + '|').join(',');
process.stdout.write(list);
" 2>/dev/null)
NODE_EXIT=$?
if [ ${NODE_EXIT} -ne 0 ]; then
  echo "ERROR: TASK '${TASK_ID}' not found in task-scope.json"
  exit 2
fi

FORBIDDEN=$(node -e "
const data = require('${SCOPE_FILE}');
const t = data && data.tasks && data.tasks['${TASK_ID}'];
const list = (t.forbiddenFiles || []).map(f => '|' + f.replace(/[.+*?^$()|[\\]\\\\]/g, '\\\\\$&') + '|').join(',');
process.stdout.write(list);
" 2>/dev/null)

REQUIRES_CONFIRMATION=$(node -e "
const data = require('${SCOPE_FILE}');
const t = data && data.tasks && data.tasks['${TASK_ID}'];
process.stdout.write(t && t.requiresHumanConfirmation ? 'true' : 'false');
")

echo "=== check-task-scope.sh ==="
echo "TASK: ${TASK_ID}"
echo "requiresHumanConfirmation: ${REQUIRES_CONFIRMATION}"
echo ""

# Collect changed + staged + untracked files.
CHANGED=$(git diff --name-only 2>/dev/null || true)
STAGED=$(git diff --cached --name-only 2>/dev/null || true)
UNTRACKED=$(git ls-files --others --exclude-standard 2>/dev/null || true)
ALL_FILES=$(printf "%s\n%s\n%s\n" "${CHANGED}" "${STAGED}" "${UNTRACKED}" | sed '/^$/d' | sort -u)

if [ -z "${ALL_FILES}" ]; then
  echo "No changed files detected."
  exit 0
fi

echo "Changed files in working tree:"
printf "%s\n" "${ALL_FILES}"
echo ""

FAIL=0

is_allowed() {
  local file="$1"
  local pattern
  IFS=',' read -ra PATTERNS <<< "${ALLOWED//|/}"
  for pattern in "${PATTERNS[@]}"; do
    [ -z "${pattern}" ] && continue
    # shellcheck disable=SC2053
    if [[ "${file}" == ${pattern} ]]; then
      return 0
    fi
  done
  return 1
}

is_forbidden() {
  local file="$1"
  local pattern
  IFS=',' read -ra PATTERNS <<< "${FORBIDDEN//|/}"
  for pattern in "${PATTERNS[@]}"; do
    [ -z "${pattern}" ] && continue
    # shellcheck disable=SC2053
    if [[ "${file}" == ${pattern} ]]; then
      return 0
    fi
  done
  return 1
}

OUT_OF_SCOPE=()
FORBIDDEN_HITS=()

while IFS= read -r file; do
  [ -z "${file}" ] && continue
  if is_forbidden "${file}"; then
    FORBIDDEN_HITS+=("${file}")
    continue
  fi
  if ! is_allowed "${file}"; then
    OUT_OF_SCOPE+=("${file}")
  fi
done <<< "${ALL_FILES}"

if [ ${#FORBIDDEN_HITS[@]} -gt 0 ]; then
  echo "ERROR: forbidden files were modified:"
  for f in "${FORBIDDEN_HITS[@]}"; do
    echo "  - ${f}"
  done
  FAIL=1
fi

if [ ${#OUT_OF_SCOPE[@]} -gt 0 ]; then
  echo "ERROR: files outside allowed scope:"
  for f in "${OUT_OF_SCOPE[@]}"; do
    echo "  - ${f}"
  done
  echo ""
  echo "Allowed patterns for ${TASK_ID}:"
  IFS=',' read -ra PATTERNS <<< "${ALLOWED//|/}"
  for pattern in "${PATTERNS[@]}"; do
    [ -z "${pattern}" ] && continue
    echo "  - ${pattern}"
  done
  FAIL=1
fi

if [ ${FAIL} -ne 0 ]; then
  echo ""
  echo "check-task-scope.sh FAILED for ${TASK_ID}"
  exit 1
fi

echo "All changed files are within scope for ${TASK_ID}."
exit 0
