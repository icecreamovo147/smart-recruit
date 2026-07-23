#!/usr/bin/env node

import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { execFileSync, spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { effectiveConfirmationReasons, extractPlan, globMatches, orderedTasks, validateManifest } from './plan_utils.mjs'
import { evaluateScope } from './check_fix_scope.mjs'
import { changedPaths } from './repository_utils.mjs'
import { stateSummary } from './state_utils.mjs'

const scriptsDir = path.dirname(fileURLToPath(import.meta.url))
const run = (script, args, cwd, expected = 0) => {
  const result = spawnSync(process.execPath, [path.join(scriptsDir, script), ...args], { cwd, encoding: 'utf8' })
  assert.equal(result.status, expected, `${script} exit ${result.status}\nstdout=${result.stdout}\nstderr=${result.stderr}`)
  return result
}

const finding = (id, target) => ({ id, title: `Finding ${id}`, type: 'STALE', severity: 'MEDIUM', status: 'OPEN', resolution_target: target })
const task = (id, priority, target, level, dependencies = [], allowed = ['docs/**']) => ({
  id,
  finding_ids: [`KNO-${id.slice(5)}`],
  priority,
  title: `Task ${id}`,
  objective: 'Repair the verified root cause.',
  rationale: 'Resolve the linked finding.',
  status: 'PENDING',
  resolution_target: target,
  knowledge_level: level,
  owner_hint: 'engineering-platform',
  dependencies,
  parallel_safe: false,
  human_confirmation_required: false,
  scope: { allowed_paths: allowed, excluded_paths: ['secret/**'] },
  source_evidence: ['docs/a.md'],
  steps: ['Revalidate', 'Repair'],
  acceptance_criteria: ['Finding is resolved'],
  tests: ['node --test'],
  rollback: 'Restore the task diff after review.',
  deliverables: ['Verified repair'],
})

const tasks = [
  task('KREM-001', 'P1', 'SOURCE', 'L2', [], ['src/**']),
  task('KREM-002', 'P0', 'KNOWLEDGE', 'L1'),
  task('KREM-003', 'P0', 'KNOWLEDGE', 'L2', ['KREM-001']),
]
const manifest = {
  schema_version: 1,
  audit: { commit: 'abc123', working_tree: 'dirty', verdict: 'DRIFT_DETECTED', generated_at: '2026-07-23T00:00:00+08:00' },
  findings: [finding('KNO-001', 'SOURCE'), finding('KNO-002', 'KNOWLEDGE'), finding('KNO-003', 'KNOWLEDGE')],
  tasks,
}

validateManifest(manifest)
assert.deepEqual(orderedTasks(tasks).map((item) => item.id), ['KREM-002', 'KREM-001', 'KREM-003'])
assert.deepEqual(effectiveConfirmationReasons(tasks[0]), ['SOURCE_TARGET'])
assert.deepEqual(effectiveConfirmationReasons({ ...tasks[1], knowledge_level: 'L3' }), ['L3'])
assert.equal(extractPlan(`before\n\`\`\`knowledge-remediation-manifest\n${JSON.stringify(manifest)}\n\`\`\`\nafter`).tasks.length, 3)
assert.throws(() => extractPlan('no manifest'), /exactly one/)
assert.throws(() => extractPlan(`\`\`\`knowledge-remediation-manifest\n${JSON.stringify(manifest)}\n\`\`\`\n\`\`\`knowledge-remediation-manifest\n${JSON.stringify(manifest)}\n\`\`\``), /found 2/)
assert.throws(() => validateManifest({ ...manifest, tasks: [{ ...tasks[0], scope: { ...tasks[0].scope, allowed_paths: ['../outside'] } }] }), /unsafe pattern/)
assert.throws(() => validateManifest({ ...manifest, tasks: [{ ...tasks[0], dependencies: ['KREM-999'] }, tasks[1], tasks[2]] }), /unknown dependency/)
assert.throws(() => validateManifest({ ...manifest, tasks: [{ ...tasks[0], dependencies: ['KREM-003'] }, tasks[1], tasks[2]] }), /cycle/)
assert.equal(globMatches('smart-recruit-*-service/**', 'smart-recruit-offer-service/internal/a.go'), true)
assert.equal(globMatches('.knowledge/**', '.knowledge/INDEX.md'), true)
assert.equal(globMatches('docs/**', 'src/main.go'), false)
assert.deepEqual(changedPaths({ 'docs/a.md': 'one' }, { 'docs/a.md': 'two', 'docs/b.md': 'one' }), ['docs/a.md', 'docs/b.md'])
assert.equal(evaluateScope({ id: 'KREM-X', allowed_paths: ['docs/**'], excluded_paths: ['docs/private/**'] }, ['docs/a.md']).passed, true)
assert.equal(evaluateScope({ id: 'KREM-X', allowed_paths: ['docs/**'], excluded_paths: ['docs/private/**'] }, ['docs/private/a.md']).passed, false)
assert.deepEqual(stateSummary({ tasks: [
  { id: 'A', order: 0, dependencies: [], status: 'READY' },
  { id: 'B', order: 1, dependencies: ['A'], status: 'PENDING' },
]}).ready_tasks, ['A'])

const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'knowledge-fix-test-'))
try {
  fs.mkdirSync(path.join(temp, 'docs'), { recursive: true })
  fs.mkdirSync(path.join(temp, 'src'), { recursive: true })
  fs.writeFileSync(path.join(temp, 'docs/a.md'), 'baseline\n')
  fs.writeFileSync(path.join(temp, 'src/main.go'), 'package main\n')
  const plan = `# Plan\n\n\`\`\`knowledge-remediation-manifest\n${JSON.stringify(manifest, null, 2)}\n\`\`\`\n`
  fs.writeFileSync(path.join(temp, 'plan.md'), plan)
  execFileSync('git', ['init', '-q'], { cwd: temp })
  execFileSync('git', ['config', 'user.email', 'test@example.invalid'], { cwd: temp })
  execFileSync('git', ['config', 'user.name', 'Test'], { cwd: temp })
  execFileSync('git', ['add', '.'], { cwd: temp })
  execFileSync('git', ['commit', '-qm', 'fixture'], { cwd: temp })
  fs.appendFileSync(path.join(temp, 'docs/a.md'), 'user dirty line\n')

  const statePath = path.join(temp, 'fix-state.json')
  run('init_fix_state.mjs', ['--plan', 'plan.md', '--output', statePath], temp)
  let state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
  assert.equal(state.tasks.find((item) => item.id === 'KREM-001').effective_confirmation_required, true)
  assert.deepEqual(state.tasks.find((item) => item.id === 'KREM-002').preexisting_allowed_overlap, ['docs/a.md'])

  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-001', '--status', 'REVALIDATING'], temp)
  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-001', '--status', 'IN_PROGRESS'], temp, 1)
  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-001', '--status', 'AWAITING_CONFIRMATION'], temp)
  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-001', '--status', 'IN_PROGRESS', '--confirmation', 'User approved SOURCE task KREM-001 for this plan hash.'], temp)

  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-002', '--status', 'REVALIDATING'], temp)
  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-002', '--status', 'IN_PROGRESS'], temp)
  run('check_fix_scope.mjs', ['--state', statePath, '--task', 'KREM-002', '--capture-baseline'], temp)
  fs.appendFileSync(path.join(temp, 'docs/a.md'), 'agent line\n')
  const scope = run('check_fix_scope.mjs', ['--state', statePath, '--task', 'KREM-002', '--verify', '--record'], temp)
  assert.match(scope.stdout, /"docs\/a.md"/)
  run('check_fix_scope.mjs', ['--state', statePath, '--task', 'KREM-002', '--path', 'secret/token.txt'], temp, 1)
  run('update_fix_state.mjs', ['--state', statePath, '--task', 'KREM-002', '--status', 'VERIFYING'], temp)
  const passArgs = ['--state', statePath, '--task', 'KREM-002', '--status', 'PASSED',
    '--check', 'finding=PASS', '--check', 'acceptance=PASS', '--check', 'tests=PASS',
    '--check', 'scope=PASS', '--check', 'diff=PASS', '--check', 'review=PASS']
  run('update_fix_state.mjs', passArgs, temp)
  run('validate_fix_state.mjs', ['--state', statePath], temp)
  const summary = path.join(temp, 'summary.md')
  run('render_fix_summary.mjs', ['--state', statePath, '--output', summary], temp)
  assert.match(fs.readFileSync(summary, 'utf8'), /KREM-002/)
} finally {
  fs.rmSync(temp, { recursive: true, force: true })
}

process.stdout.write('knowledge-current-state-fix tests: PASS\n')
