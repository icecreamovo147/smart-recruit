#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FEATURE_DIR="${ROOT}/.spec/agent-harness-unification"
FINAL_AUDIT="${FINAL_AUDIT:-0}"

cd "${ROOT}"

echo "==> validate feature Harness structure"
node - "${FEATURE_DIR}" <<'NODE'
const fs = require('fs');
const path = require('path');

const featureDir = process.argv[2];
const scopePath = path.join(featureDir, 'task-scope.json');
const tasksPath = path.join(featureDir, 'TASKS.md');
const required = [
  'agent-harness-unification-SPEC.md',
  'agent-harness-unification-SDD.md',
  'TASKS.md',
  'AGENT_RULES.md',
  'task-scope.json',
  'prompts/implement-task.md',
  'prompts/self-review.md',
  'prompts/fix-check-failures.md',
  'scripts/check-task-scope.sh',
  'scripts/agent-check.sh',
];

let failed = false;
for (const relative of required) {
  if (!fs.existsSync(path.join(featureDir, relative))) {
    console.error(`MISSING: ${relative}`);
    failed = true;
  }
}

const scope = JSON.parse(fs.readFileSync(scopePath, 'utf8'));
if (scope.schemaVersion !== 1 || scope.feature_name !== 'agent-harness-unification' || !scope.tasks) {
  console.error('INVALID: task-scope.json top-level schema');
  failed = true;
}

const taskMarkdown = fs.readFileSync(tasksPath, 'utf8');
const taskIds = Object.keys(scope.tasks || {});
if (taskIds.length !== 7) {
  console.error(`INVALID: expected 7 tasks, found ${taskIds.length}`);
  failed = true;
}

for (const taskId of taskIds) {
  const task = scope.tasks[taskId];
  if (!taskMarkdown.includes(`## ${taskId} -`)) {
    console.error(`MISSING TASKS section: ${taskId}`);
    failed = true;
  }
  if (!Array.isArray(task.allowedFiles) || task.allowedFiles.length === 0) {
    console.error(`INVALID allowedFiles: ${taskId}`);
    failed = true;
  }
  if (!Array.isArray(task.forbiddenFiles) || task.forbiddenFiles.length === 0) {
    console.error(`INVALID forbiddenFiles: ${taskId}`);
    failed = true;
  }
  const acceptance = path.join(path.dirname(featureDir), '..', task.acceptance || '');
  if (!task.acceptance || !fs.existsSync(path.resolve(acceptance))) {
    console.error(`MISSING acceptance: ${taskId} -> ${task.acceptance}`);
    failed = true;
  }
  if (!task.report || !task.report.endsWith(`${taskId}-report.md`)) {
    console.error(`INVALID report path: ${taskId}`);
    failed = true;
  }
}

if (failed) process.exit(1);
console.log(`Harness structure OK (${taskIds.length} tasks)`);
NODE

echo "==> validate shell syntax"
bash -n "${FEATURE_DIR}/scripts/check-task-scope.sh"
bash -n "${FEATURE_DIR}/scripts/agent-check.sh"

echo "==> validate changed-file governance boundary"
node - "${ROOT}" <<'NODE'
const { execFileSync } = require('child_process');
const root = process.argv[2];

function git(args) {
  return execFileSync('git', ['-C', root, ...args], { encoding: 'utf8' })
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
}

const changed = new Set([
  ...git(['diff', '--name-only', 'HEAD']),
  ...git(['ls-files', '--others', '--exclude-standard']),
]);

const exact = new Set([
  'AGENTS.md',
  'CLAUDE.md',
  '.claude/CLAUDE.md',
  'docs/agent-harness/README.md',
  'docs/agent-harness/00-HARNESS.md',
  '.ai-guides/README.md',
]);
const prefixes = [
  '.spec/agent-harness-unification/',
  '.agents/skills/spec-harness/',
  '.agents/skills/harness-pipeline/',
  '.claude/skills/',
  '.claude/agents/',
  '.claude/commands/',
];

const invalid = [...changed].filter((file) => !exact.has(file) && !prefixes.some((prefix) => file.startsWith(prefix)));
if (invalid.length > 0) {
  console.error('Files outside agent-harness-unification implementation boundary:');
  for (const file of invalid.sort()) console.error(`  - ${file}`);
  process.exit(1);
}
console.log(`Governance boundary OK (${changed.size} changed files)`);
NODE

if [ "${FINAL_AUDIT}" = "1" ]; then
  echo "==> run final cross-control-plane audit"

  required_final=(
    "CLAUDE.md"
    "docs/agent-harness/README.md"
    ".ai-guides/README.md"
    ".spec/agent-harness-unification/reports/legacy-inventory.md"
    ".agents/skills/spec-harness/scripts/validate-feature.mjs"
    ".agents/skills/spec-harness/scripts/check-task-scope.mjs"
    ".agents/skills/spec-harness/scripts/validate-evidence.mjs"
    ".agents/skills/spec-harness/scripts/audit-specs.mjs"
    ".agents/skills/spec-harness/scripts/validator.test.mjs"
    ".agents/skills/harness-pipeline/scripts/validate-pipeline-state.mjs"
    ".agents/skills/harness-pipeline/scripts/pipeline-state.test.mjs"
  )

  for file in "${required_final[@]}"; do
    if [ ! -f "${file}" ]; then
      echo "MISSING final artifact: ${file}" >&2
      exit 1
    fi
  done

  adapter_files=(
    "CLAUDE.md"
    ".claude/CLAUDE.md"
    ".claude/skills/run-agent-task/SKILL.md"
    ".claude/skills/review-agent-task/SKILL.md"
    ".claude/skills/fix-agent-task/SKILL.md"
    ".claude/skills/batch-agent-coordinator/SKILL.md"
    ".claude/commands/run-phase.md"
    ".claude/agents/task-developer.md"
    ".claude/agents/task-reviewer.md"
    ".claude/agents/task-fixer.md"
    ".claude/agents/phase-implementer.md"
    ".claude/agents/phase-reviewer.md"
  )

  if rg -n '/Users/|integration/agent-platform' "${adapter_files[@]}"; then
    echo "ERROR: provider adapters still contain absolute paths or the legacy fixed integration branch" >&2
    exit 1
  fi

  for file in "${adapter_files[@]}"; do
    if ! rg -q '\.spec/|spec-harness|harness-pipeline' "${file}"; then
      echo "ERROR: adapter does not reference the canonical control plane: ${file}" >&2
      exit 1
    fi
  done

  node .agents/skills/spec-harness/scripts/validator.test.mjs
  node .agents/skills/harness-pipeline/scripts/pipeline-state.test.mjs
  node .agents/skills/spec-harness/scripts/audit-specs.mjs --root .spec

  node - <<'NODE'
const fs = require('fs');
const path = require('path');

const inventory = fs.readFileSync('.spec/agent-harness-unification/reports/legacy-inventory.md', 'utf8');
const log = fs.readFileSync('docs/agent-harness/EXECUTION_LOG.md', 'utf8');
const pending = [...log.matchAll(/^\|\s*\d+\s*\|\s*(P\d+-\d+)\s*\|.*\|\s*pending\s*\|/gm)].map((match) => match[1]);
if (pending.length !== 11) {
  console.error(`ERROR: expected 11 legacy pending tasks, found ${pending.length}`);
  process.exit(1);
}
for (const id of pending) {
  if (!inventory.includes(id)) {
    console.error(`ERROR: legacy inventory is missing ${id}`);
    process.exit(1);
  }
}

const phaseDirs = fs.readdirSync('.ai-guides', { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .filter((name) => ['constitution.md', 'spec.md', 'plan.md', 'tasks.md']
    .every((file) => fs.existsSync(path.join('.ai-guides', name, file))));
for (const phase of phaseDirs) {
  if (!inventory.includes(phase)) {
    console.error(`ERROR: legacy inventory is missing phase ${phase}`);
    process.exit(1);
  }
}
if (!inventory.includes('semantic-retrieval-score-fixes') || !inventory.includes('separate-repair')) {
  console.error('ERROR: legacy inventory is missing the semantic-retrieval-score-fixes separate-repair decision');
  process.exit(1);
}
console.log(`Legacy inventory OK (${pending.length} pending tasks, ${phaseDirs.length} phase contracts)`);
NODE
fi

echo "agent-check: PASS"
