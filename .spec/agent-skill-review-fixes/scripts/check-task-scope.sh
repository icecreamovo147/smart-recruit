#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-}"
FEATURE_DIR=".spec/agent-skill-review-fixes"
SCOPE_FILE="$FEATURE_DIR/task-scope.json"

if [[ -z "$TASK_ID" ]]; then
  echo "usage: $0 <TASK-ID>" >&2
  exit 2
fi

if [[ ! -f "$SCOPE_FILE" ]]; then
  echo "missing scope file: $SCOPE_FILE" >&2
  exit 2
fi

node - "$TASK_ID" "$SCOPE_FILE" <<'NODE'
const fs = require('fs')
const taskId = process.argv[2]
const scopeFile = process.argv[3]
const scope = JSON.parse(fs.readFileSync(scopeFile, 'utf8'))
if (!scope.tasks || !scope.tasks[taskId]) {
  console.error(`unknown task: ${taskId}`)
  process.exit(2)
}
NODE

changed="$(git diff --name-only)"
if [[ -z "$changed" ]]; then
  echo "No changed files."
  exit 0
fi

node - "$TASK_ID" "$SCOPE_FILE" "$changed" <<'NODE'
const fs = require('fs')
const taskId = process.argv[2]
const scopeFile = process.argv[3]
const changed = process.argv[4].split('\n').filter(Boolean)
const scope = JSON.parse(fs.readFileSync(scopeFile, 'utf8'))
const task = scope.tasks[taskId]

function escapeRegex(s) {
  return s.replace(/[.+^${}()|[\]\\]/g, '\\$&')
}

function patternToRegex(pattern) {
  const parts = pattern.split('**').map(part => escapeRegex(part).replace(/\*/g, '[^/]*'))
  return new RegExp(`^${parts.join('.*')}$`)
}

const allowed = (task.allowedFiles || []).map(patternToRegex)
const forbidden = (task.forbiddenFiles || []).map(patternToRegex)
const outOfScope = []

for (const file of changed) {
  const isAllowed = allowed.some(rx => rx.test(file))
  const isForbidden = forbidden.some(rx => rx.test(file))
  if (!isAllowed || isForbidden) outOfScope.push(file)
}

if (outOfScope.length) {
  console.error(`Out-of-scope files for ${taskId}:`)
  for (const file of outOfScope) console.error(` - ${file}`)
  process.exit(1)
}

console.log(`Scope check passed for ${taskId}.`)
NODE
