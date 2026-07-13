#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-}"
ROOT="$(git rev-parse --show-toplevel)"
SCOPE_FILE="$ROOT/.spec/microservice-ddd-evolution/task-scope.json"

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
trap 'rm -f "$CHANGED_FILE"' EXIT

{
  git diff --name-only
  git diff --cached --name-only
  git ls-files --others --exclude-standard
} | sort -u > "$CHANGED_FILE"

node - "$TASK_ID" "$SCOPE_FILE" "$CHANGED_FILE" <<'NODE'
const fs = require("fs");
const [taskId, scopeFile, changedFile] = process.argv.slice(2);
const scope = JSON.parse(fs.readFileSync(scopeFile, "utf8"));
const changed = fs.readFileSync(changedFile, "utf8").split(/\r?\n/).filter(Boolean);

if (scope.schemaVersion !== 1 || scope.feature_name !== "microservice-ddd-evolution") {
  console.error("unsupported task-scope schema or feature_name");
  process.exit(2);
}

const task = scope.tasks && scope.tasks[taskId];
if (!task) {
  console.error(`unknown TASK: ${taskId}`);
  process.exit(2);
}

const alwaysAllowed = [
  `.spec/microservice-ddd-evolution/reports/${taskId}-report.md`,
  `.spec/microservice-ddd-evolution/reports/${taskId}-evidence.json`
];
const allowed = [...(task.allowedFiles || []), ...alwaysAllowed];
const forbidden = task.forbiddenFiles || [];

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

function matches(path, patterns) {
  return patterns.some((pattern) => {
    if (pattern.endsWith("/**")) {
      const prefix = pattern.slice(0, -3);
      return path === prefix || path.startsWith(prefix + "/");
    }
    return globToRegExp(pattern).test(path);
  });
}

const relevant = changed.filter(Boolean);
const violations = [];

for (const file of relevant) {
  if (matches(file, forbidden)) {
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

console.log(`Scope check passed for ${taskId}. Changed files: ${relevant.length}`);
NODE
