#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TASK_ID="${1:-}"
SCOPE_FILE="${ROOT}/.spec/agent-skill-selection-confirmation/task-scope.json"

cd "${ROOT}" || exit 1

if [ -z "${TASK_ID}" ]; then
  echo "ERROR: TASK-ID is required"
  echo "usage: bash $0 <TASK-ID>"
  exit 2
fi

if [ ! -f "${SCOPE_FILE}" ]; then
  echo "ERROR: task-scope.json not found: ${SCOPE_FILE}"
  exit 2
fi

TASK_EXISTS=$(node -e "const d=require('${SCOPE_FILE}'); process.stdout.write(d.tasks && d.tasks['${TASK_ID}'] ? 'yes' : 'no')" 2>/dev/null)
if [ "${TASK_EXISTS}" != "yes" ]; then
  echo "ERROR: TASK '${TASK_ID}' not found in task-scope.json"
  exit 2
fi

ALLOWED=$(node -e "const d=require('${SCOPE_FILE}'); console.log((d.tasks['${TASK_ID}'].allowedFiles||[]).join('\n'))")
FORBIDDEN=$(node -e "const d=require('${SCOPE_FILE}'); console.log((d.tasks['${TASK_ID}'].forbiddenFiles||[]).join('\n'))")
REQUIRES_CONFIRMATION=$(node -e "const d=require('${SCOPE_FILE}'); process.stdout.write(d.tasks['${TASK_ID}'].requiresHumanConfirmation ? 'true' : 'false')")

echo "=== check-task-scope.sh ==="
echo "TASK: ${TASK_ID}"
echo "requiresHumanConfirmation: ${REQUIRES_CONFIRMATION}"
echo ""

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

matches_pattern() {
  local file="$1"
  local pattern="$2"
  [ -z "${pattern}" ] && return 1
  if [ "${pattern}" = "**" ]; then
    return 0
  fi
  if [[ "${pattern}" == *"**" ]]; then
    local prefix="${pattern:0:${#pattern}-2}"
    [[ "${file}" == "${prefix}"* ]]
    return $?
  fi
  if [[ "${pattern}" == *"*"* ]]; then
    [[ "${file}" == ${pattern} ]]
    return $?
  fi
  [[ "${file}" == "${pattern}" ]]
  return $?
}

is_allowed() {
  local file="$1"
  while IFS= read -r pattern; do
    if matches_pattern "${file}" "${pattern}"; then
      return 0
    fi
  done <<< "${ALLOWED}"
  return 1
}

is_forbidden() {
  local file="$1"
  while IFS= read -r pattern; do
    if matches_pattern "${file}" "${pattern}"; then
      return 0
    fi
  done <<< "${FORBIDDEN}"
  return 1
}

FAIL=0
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
  while IFS= read -r pattern; do
    [ -n "${pattern}" ] && echo "  - ${pattern}"
  done <<< "${ALLOWED}"
  FAIL=1
fi

if [ ${FAIL} -ne 0 ]; then
  echo ""
  echo "check-task-scope.sh FAILED for ${TASK_ID}"
  exit 1
fi

echo "All changed files are within scope for ${TASK_ID}."
exit 0
