#!/usr/bin/env bash
set -euo pipefail

TASK_ID="${1:-}"
if [[ -z "$TASK_ID" ]]; then
  echo "usage: $0 <TASK-ID>" >&2
  exit 2
fi

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

SCOPE_FILE=".spec/legacydomain-migration-finalization/task-scope.json"
if [[ ! -f "$SCOPE_FILE" ]]; then
  echo "missing scope file: $SCOPE_FILE" >&2
  exit 2
fi

node - "$TASK_ID" "$SCOPE_FILE" <<'NODE'
const fs = require('fs');
const cp = require('child_process');
const taskId = process.argv[2];
const scopeFile = process.argv[3];
const scope = JSON.parse(fs.readFileSync(scopeFile, 'utf8'));
const task = scope.tasks && scope.tasks[taskId];
if (!task) {
  console.error(`unknown task: ${taskId}`);
  process.exit(2);
}

const stateFile = '.spec/legacydomain-migration-finalization/pipeline-state.json';
const baseTree = readTaskBaseTree(stateFile, taskId);
const changed = baseTree ? changedSinceBaseTree(baseTree) : changedSinceHead();

function changedSinceHead() {
  const tracked = cp.execSync('git diff --name-only HEAD', {encoding: 'utf8'})
    .split(/\r?\n/)
    .map(s => s.trim())
    .filter(Boolean);
  const untracked = cp.execSync('git ls-files --others --exclude-standard', {encoding: 'utf8'})
    .split(/\r?\n/)
    .map(s => s.trim())
    .filter(Boolean);
  return Array.from(new Set([...tracked, ...untracked])).sort();
}

function readTaskBaseTree(file, id) {
  if (!fs.existsSync(file)) return '';
  const state = JSON.parse(fs.readFileSync(file, 'utf8'));
  return state.task_runs?.[id]?.base_tree || '';
}

function changedSinceBaseTree(baseTreeId) {
  const currentTree = writeCurrentWorktreeTree();
  return cp.execFileSync('git', ['diff', '--name-only', baseTreeId, currentTree], {encoding: 'utf8'})
  .split(/\r?\n/)
  .map(s => s.trim())
  .filter(Boolean);
}

function writeCurrentWorktreeTree() {
  const indexFile = cp.execFileSync('mktemp', {encoding: 'utf8'}).trim();
  const env = {...process.env, GIT_INDEX_FILE: indexFile};
  try {
    cp.execFileSync('git', ['read-tree', 'HEAD'], {env});
    const addInput = cp.execFileSync('git', ['ls-files', '-m', '-o', '--exclude-standard', '-z']);
    if (addInput.length > 0) {
      cp.execFileSync('git', ['update-index', '--add', '-z', '--stdin'], {env, input: addInput});
    }
    const removeInput = cp.execFileSync('git', ['ls-files', '-d', '-z']);
    if (removeInput.length > 0) {
      cp.execFileSync('git', ['update-index', '--remove', '-z', '--stdin'], {env, input: removeInput});
    }
    return cp.execFileSync('git', ['write-tree'], {env, encoding: 'utf8'}).trim();
  } finally {
    try { fs.unlinkSync(indexFile); } catch {}
  }
}

function escapeRegex(s) {
  return s.replace(/[.+^${}()|[\]\\]/g, '\\$&');
}

function globToRegex(glob) {
  let out = '';
  for (let i = 0; i < glob.length; i++) {
    const ch = glob[i];
    if (ch === '*') {
      if (glob[i + 1] === '*') {
        out += '.*';
        i++;
      } else {
        out += '[^/]*';
      }
    } else {
      out += escapeRegex(ch);
    }
  }
  return new RegExp(`^${out}$`);
}

const allowed = (task.allowedFiles || []).map(globToRegex);
const forbidden = (task.forbiddenFiles || []).map(globToRegex);
const outOfScope = [];
const forbiddenHits = [];

for (const file of changed) {
  if (forbidden.some(re => re.test(file))) {
    forbiddenHits.push(file);
    continue;
  }
  if (!allowed.some(re => re.test(file))) {
    outOfScope.push(file);
  }
}

if (changed.length === 0) {
  console.log(`scope ok: no changed files for ${taskId}`);
  process.exit(0);
}

if (forbiddenHits.length || outOfScope.length) {
  if (forbiddenHits.length) {
    console.error('forbidden files:');
    for (const file of forbiddenHits) console.error(`  ${file}`);
  }
  if (outOfScope.length) {
    console.error('out-of-scope files:');
    for (const file of outOfScope) console.error(`  ${file}`);
  }
  process.exit(1);
}

console.log(`scope ok: ${changed.length} changed file(s) within ${taskId}`);
NODE
