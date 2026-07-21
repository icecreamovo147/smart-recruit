#!/usr/bin/env node

import assert from 'node:assert/strict'
import { extractPlan, globMatches, orderedTasks, validateManifest } from './plan_utils.mjs'
import { stateSummary } from './state_utils.mjs'

const task = (id, priority, dependencies = []) => ({
  id,
  finding_ids: [`SEC-${id.slice(4)}`],
  priority,
  title: `Task ${id}`,
  objective: 'Fix the root cause.',
  rationale: 'Security remediation.',
  status: 'PENDING',
  owner_hint: 'security',
  dependencies,
  parallel_safe: false,
  human_confirmation_required: false,
  scope: { allowed_paths: ['smart-recruit-gateway/**'], excluded_paths: ['smart-recruit-proto/**'] },
  steps: ['Implement'],
  acceptance_criteria: ['Attack is blocked'],
  tests: ['Run test'],
  rollback: 'Contain safely.',
  security_notes: [],
  deliverables: ['Patch'],
})

const tasks = [task('REM-001', 'P1'), task('REM-002', 'P0'), task('REM-003', 'P0', ['REM-001'])]
const manifest = {
  schema_version: 1,
  artifact_type: 'prelaunch-security-remediation-plan',
  audit: { commit: 'abc', verdict: 'NO_GO' },
  findings: [],
  remediation: { strategy: [], tasks },
}

validateManifest(manifest)
assert.deepEqual(orderedTasks(tasks).map((item) => item.id), ['REM-002', 'REM-001', 'REM-003'])
assert.equal(extractPlan(`before\n\`\`\`prelaunch-security-remediation-manifest\n${JSON.stringify(manifest)}\n\`\`\`\nafter`).artifact_type, 'prelaunch-security-remediation-plan')
assert.equal(globMatches('smart-recruit-gateway/**', 'smart-recruit-gateway/router/router.go'), true)
assert.equal(globMatches('smart-recruit-gateway/**', 'smart-recruit-proto/proto/recruitment.proto'), false)

const state = {
  tasks: [
    { id: 'REM-002', order: 0, dependencies: [], status: 'PENDING' },
    { id: 'REM-001', order: 1, dependencies: [], status: 'PASSED' },
    { id: 'REM-003', order: 2, dependencies: ['REM-001'], status: 'PENDING' },
  ],
}
assert.deepEqual(stateSummary(state).ready_tasks, ['REM-002', 'REM-003'])

const cyclic = [task('REM-001', 'P0', ['REM-002']), task('REM-002', 'P0', ['REM-001'])]
assert.throws(() => validateManifest({ ...manifest, remediation: { strategy: [], tasks: cyclic } }), /cycle/)

process.stdout.write('prelaunch-security-fix tests: PASS\n')

