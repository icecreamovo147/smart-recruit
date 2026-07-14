#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
FEATURE_DIR="$ROOT/.spec/legacydomain-retirement"
cd "$ROOT"

required=(
  "$FEATURE_DIR/legacydomain-retirement-SPEC.md"
  "$FEATURE_DIR/legacydomain-retirement-SDD.md"
  "$FEATURE_DIR/TASKS.md"
  "$FEATURE_DIR/AGENT_RULES.md"
  "$FEATURE_DIR/task-scope.json"
  "$FEATURE_DIR/pipeline-state.json"
  "$FEATURE_DIR/prompts/implement-task.md"
  "$FEATURE_DIR/prompts/self-review.md"
  "$FEATURE_DIR/prompts/fix-check-failures.md"
  "$FEATURE_DIR/scripts/check-task-scope.sh"
)

for file in "${required[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "missing required harness file: $file" >&2
    exit 1
  fi
done

node - "$FEATURE_DIR/task-scope.json" "$FEATURE_DIR" <<'NODE'
const fs = require("fs");
const [scopePath, featureDir] = process.argv.slice(2);
const data = JSON.parse(fs.readFileSync(scopePath, "utf8"));
if (data.schemaVersion !== 1) throw new Error("schemaVersion must be 1");
if (data.feature_name !== "legacydomain-retirement") throw new Error("feature_name mismatch");
const tasks = data.tasks || {};
const ids = Object.keys(tasks).sort();
if (ids.length !== 9) throw new Error(`expected 9 tasks, got ${ids.length}`);
for (const id of ids) {
  const task = tasks[id];
  for (const key of ["title", "status", "allowedFiles", "forbiddenFiles", "acceptance", "report", "requiredKnowledgeImpact", "knowledge"]) {
    if (!(key in task)) throw new Error(`${id} missing ${key}`);
  }
  if (task.requiredKnowledgeImpact !== true) throw new Error(`${id} must require knowledge impact`);
  if (!task.allowedFiles.includes(".knowledge/**")) throw new Error(`${id} must allow .knowledge/** updates`);
  if (!fs.existsSync(task.acceptance)) throw new Error(`${id} acceptance missing: ${task.acceptance}`);
}
for (const id of ids) {
  const acceptance = `${featureDir}/acceptance/${id}.md`;
  if (!fs.existsSync(acceptance)) throw new Error(`missing acceptance file: ${acceptance}`);
}
console.log("Harness JSON validation passed.");
NODE

node .knowledge/scripts/validate-knowledge.mjs --root .
node .knowledge/scripts/check-references.mjs --root .
node scripts/check-backend-boundaries.mjs
node scripts/check-mysql-table-ownership.mjs

CHANGED_FILE="$(mktemp)"
MODULES_FILE="$(mktemp)"
trap 'rm -f "$CHANGED_FILE" "$MODULES_FILE"' EXIT

{
  git diff --name-only
  git diff --cached --name-only
  git ls-files --others --exclude-standard
} | sort -u > "$CHANGED_FILE"

: > "$MODULES_FILE"
while IFS= read -r file; do
  case "$file" in
    smart-recruit-ai-agent-service/*) echo "smart-recruit-ai-agent-service" >> "$MODULES_FILE" ;;
    smart-recruit-analytics-service/*) echo "smart-recruit-analytics-service" >> "$MODULES_FILE" ;;
    smart-recruit-commons/*) echo "smart-recruit-commons" >> "$MODULES_FILE" ;;
    smart-recruit-gateway/*) echo "smart-recruit-gateway" >> "$MODULES_FILE" ;;
    smart-recruit-identity-service/*) echo "smart-recruit-identity-service" >> "$MODULES_FILE" ;;
    smart-recruit-interview-service/*) echo "smart-recruit-interview-service" >> "$MODULES_FILE" ;;
    smart-recruit-notification-service/*) echo "smart-recruit-notification-service" >> "$MODULES_FILE" ;;
    smart-recruit-offer-service/*) echo "smart-recruit-offer-service" >> "$MODULES_FILE" ;;
    smart-recruit-platform-go/*) echo "smart-recruit-platform-go" >> "$MODULES_FILE" ;;
    smart-recruit-proto/*) echo "smart-recruit-proto" >> "$MODULES_FILE" ;;
    smart-recruit-recruitment-service/*) echo "smart-recruit-recruitment-service" >> "$MODULES_FILE" ;;
    smart-recruit-worker-service/*) echo "smart-recruit-worker-service" >> "$MODULES_FILE" ;;
  esac
done < "$CHANGED_FILE"

if [[ ! -s "$MODULES_FILE" ]]; then
  echo "No Go module changes detected; skipped go test."
  exit 0
fi

sort -u "$MODULES_FILE" | while IFS= read -r module; do
  if [[ -f "$module/go.mod" ]]; then
    echo "Running go test ./... in $module"
    (cd "$module" && go test ./...)
  else
    echo "Skipping $module: go.mod not found"
  fi
done
