#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-}"
ROOT="$(git rev-parse --show-toplevel)"
SCOPE_FILE="$ROOT/.spec/legacydomain-retirement/task-scope.json"
STATE_FILE="$ROOT/.spec/legacydomain-retirement/pipeline-state.json"

if [[ -z "$TASK_ID" ]]; then
  echo "usage: $0 <TASK-ID>" >&2
  exit 2
fi

if [[ ! -f "$SCOPE_FILE" ]]; then
  echo "task-scope.json not found: $SCOPE_FILE" >&2
  exit 2
fi

cd "$ROOT"

CHANGED_FILE="$(mktemp)"
STATUS_FILE="$(mktemp)"
BASE_INDEX=""
trap 'rm -f "$CHANGED_FILE" "$STATUS_FILE" "$BASE_INDEX"' EXIT

BASE_TREE="${TASK_SCOPE_BASE_TREE:-}"
if [[ -z "$BASE_TREE" && -f "$STATE_FILE" ]]; then
  BASE_TREE="$(node -e 'const fs=require("fs"); const s=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); process.stdout.write((s.task_runs && s.task_runs[process.argv[2]] && s.task_runs[process.argv[2]].base_tree) || "");' "$STATE_FILE" "$TASK_ID")"
fi

if [[ -n "$BASE_TREE" ]]; then
  BASE_INDEX="$(mktemp)"
  GIT_INDEX_FILE="$BASE_INDEX" git read-tree "$BASE_TREE"
  GIT_INDEX_FILE="$BASE_INDEX" git diff --name-only > "$CHANGED_FILE"
  GIT_INDEX_FILE="$BASE_INDEX" git ls-files --others --exclude-standard | while IFS= read -r file; do
    if git cat-file -e "$BASE_TREE:$file" 2>/dev/null && git show "$BASE_TREE:$file" | cmp -s - "$file"; then
      continue
    fi
    echo "$file"
  done >> "$CHANGED_FILE"
  sort -u "$CHANGED_FILE" -o "$CHANGED_FILE"
  GIT_INDEX_FILE="$BASE_INDEX" git diff --name-status --find-renames > "$STATUS_FILE"
else
  {
    git diff --name-only
    git diff --cached --name-only
    git ls-files --others --exclude-standard
  } | sort -u > "$CHANGED_FILE"

  {
    git diff --name-status --find-renames
    git diff --cached --name-status --find-renames
  } > "$STATUS_FILE"
fi

node - "$TASK_ID" "$SCOPE_FILE" "$CHANGED_FILE" "$STATUS_FILE" <<'NODE'
const fs = require("fs");
const [taskId, scopeFile, changedFile, statusFile] = process.argv.slice(2);
const scope = JSON.parse(fs.readFileSync(scopeFile, "utf8"));
const changed = fs.readFileSync(changedFile, "utf8").split(/\r?\n/).filter(Boolean);
const statuses = fs.readFileSync(statusFile, "utf8").split(/\r?\n/).filter(Boolean);

if (scope.schemaVersion !== 1 || scope.feature_name !== "legacydomain-retirement") {
  console.error("unsupported task-scope schema or feature_name");
  process.exit(2);
}

const task = scope.tasks && scope.tasks[taskId];
if (!task) {
  console.error(`unknown TASK: ${taskId}`);
  process.exit(2);
}

const alwaysAllowed = [
  `.spec/legacydomain-retirement/reports/${taskId}-report.md`,
  `.spec/legacydomain-retirement/reports/${taskId}-evidence.json`,
  ".spec/legacydomain-retirement/pipeline-state.json",
  ".spec/legacydomain-retirement/scripts/check-task-scope.sh"
];
const allowed = [...(task.allowedFiles || []), ...alwaysAllowed];
const forbidden = task.forbiddenFiles || [];
const allowedActions = new Set(task.allowedActions || []);

function escapeRegex(s) {
  return s.replace(/[|\\{}()[\]^$+?.]/g, "\\$&");
}

function globToRegExp(glob) {
  let out = "";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (glob[i + 1] === "*") {
        out += ".*";
        i++;
      } else {
        out += "[^/]*";
      }
    } else if (c === "?") {
      out += "[^/]";
    } else {
      out += escapeRegex(c);
    }
  }
  return new RegExp(`^${out}$`);
}

function matches(filePath, patterns) {
  return patterns.some((pattern) => {
    if (pattern.endsWith("/**")) {
      const prefix = pattern.slice(0, -3);
      if (!/[?*]/.test(prefix)) {
        return filePath === prefix || filePath.startsWith(prefix + "/");
      }
    }
    return globToRegExp(pattern).test(filePath);
  });
}

const pureRenameTargets = new Set();
for (const line of statuses) {
  const parts = line.split(/\t/);
  const status = parts[0] || "";
  if (status !== "R100" || parts.length < 3) continue;
  const [, from, to] = parts;
  if (allowedActions.has("move") && matches(from, allowed) && matches(to, allowed) && matches(to, forbidden)) {
    pureRenameTargets.add(to);
  }
}

const violations = [];
for (const file of changed) {
  if (matches(file, forbidden)) {
    if (pureRenameTargets.has(file)) continue;
    violations.push(`${file} (forbidden)`);
    continue;
  }
  if (!matches(file, allowed)) {
    violations.push(`${file} (out of scope)`);
  }
}

if (violations.length) {
  console.error(`Scope check failed for ${taskId}:`);
  for (const file of violations) console.error(` - ${file}`);
  process.exit(1);
}

console.log(`Scope check passed for ${taskId}. Changed files: ${changed.length}`);
NODE
