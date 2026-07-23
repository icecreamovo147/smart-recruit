#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { asList, atomicWriteJson, cliValues, now, parseCli, SUCCESS_STATES, TERMINAL_STATES } from './plan_utils.mjs'
import { stateSummary, validateState } from './state_utils.mjs'

const TRANSITIONS = {
  PENDING: new Set(['READY', 'BLOCKED', 'STALE_PLAN', 'DEFERRED']),
  READY: new Set(['REVALIDATING', 'BLOCKED', 'STALE_PLAN']),
  REVALIDATING: new Set(['AWAITING_CONFIRMATION', 'IN_PROGRESS', 'ALREADY_RESOLVED', 'BLOCKED', 'STALE_PLAN', 'SCOPE_GAP']),
  AWAITING_CONFIRMATION: new Set(['IN_PROGRESS', 'DEFERRED', 'ABORTED']),
  IN_PROGRESS: new Set(['VERIFYING', 'BLOCKED', 'STALE_PLAN', 'SCOPE_GAP', 'ABORTED']),
  VERIFYING: new Set(['PASSED', 'IN_PROGRESS', 'FAILED', 'BLOCKED', 'SCOPE_GAP', 'ABORTED']),
  FAILED: new Set(['READY']),
  BLOCKED: new Set(['READY']),
  STALE_PLAN: new Set(['READY']),
  SCOPE_GAP: new Set(['READY']),
  DEFERRED: new Set(['READY', 'ACCEPTED']),
}
const REQUIRED_PASS_CHECKS = ['finding', 'acceptance', 'tests', 'scope', 'diff', 'review']

const parseCheck = (raw) => {
  const match = String(raw).match(/^([a-zA-Z0-9_-]+)=(PASS|FAIL|NOT_RUN|BLOCKED)(?::([\s\S]*))?$/)
  if (!match) throw new Error(`invalid --check value: ${raw}; expected name=PASS|FAIL|NOT_RUN|BLOCKED[:detail]`)
  return { name: match[1], status: match[2], detail: match[3] || '', at: now() }
}

const deriveRunStatus = (state) => {
  const tasks = asList(state.tasks)
  if (tasks.some((task) => ['READY', 'REVALIDATING', 'IN_PROGRESS', 'VERIFYING'].includes(task.status))) return 'IN_PROGRESS'
  if (tasks.some((task) => task.status === 'AWAITING_CONFIRMATION')) return 'AWAITING_CONFIRMATION'
  const allSuccess = tasks.every((task) => SUCCESS_STATES.has(task.status))
  if (state.selection.mode === 'all' && allSuccess) return 'REMEDIATION_COMPLETE_PENDING_REAUDIT'
  if (tasks.some((task) => SUCCESS_STATES.has(task.status))) return 'REMEDIATION_PARTIAL'
  if (tasks.some((task) => task.status === 'ABORTED')) return 'REMEDIATION_ABORTED'
  return 'REMEDIATION_BLOCKED'
}

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.state || !args.task || !args.status) {
    process.stdout.write('Usage: update_fix_state.mjs --state <state.json> --task KREM-001 --status <STATE> [--note text] [--confirmation text] [--check name=PASS:detail] [--modified-path path]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    validateState(state, statePath)
    const task = state.tasks.find((item) => item.id === args.task)
    if (!task) throw new Error(`task not found in state: ${args.task}`)
    const previous = task.status
    const target = args.status
    if (!TRANSITIONS[previous]?.has(target)) throw new Error(`invalid transition ${previous} -> ${target}`)
    const notes = cliValues(args.note)
    const confirmations = cliValues(args.confirmation)
    if (TERMINAL_STATES.has(previous) && target === 'READY' && !notes.length) throw new Error('reopening a terminal task requires --note explaining what changed')
    if (previous === 'REVALIDATING' && target === 'IN_PROGRESS' && task.effective_confirmation_required) {
      throw new Error(`${task.id} requires AWAITING_CONFIRMATION before IN_PROGRESS: ${task.effective_confirmation_reasons.join(', ')}`)
    }
    if (previous === 'AWAITING_CONFIRMATION' && target === 'IN_PROGRESS' && !confirmations.length && !task.confirmations.length) {
      throw new Error('gated task requires --confirmation with the exact authorization or decision')
    }
    if (target === 'ACCEPTED' && !confirmations.length) throw new Error('ACCEPTED requires --confirmation identifying the authorized owner and review/expiry condition')

    const timestamp = now()
    task.notes.push(...notes.map((text) => ({ at: timestamp, text })))
    task.confirmations.push(...confirmations.map((text) => ({ at: timestamp, text })))
    task.modified_paths = [...new Set([...task.modified_paths, ...cliValues(args.modifiedPath)])].sort()
    for (const raw of cliValues(args.check)) {
      const check = parseCheck(raw)
      task.checks = task.checks.filter((item) => item.name !== check.name)
      task.checks.push(check)
    }
    if (['PASSED', 'ALREADY_RESOLVED'].includes(target)) {
      const passed = new Set(task.checks.filter((check) => check.status === 'PASS').map((check) => check.name))
      const missing = REQUIRED_PASS_CHECKS.filter((name) => !passed.has(name))
      if (missing.length) throw new Error(`${target} requires PASS checks: ${missing.join(', ')}`)
      if (task.scope_result?.passed !== true) throw new Error(`${target} requires a recorded passing fingerprint scope result`)
    }
    task.status = target
    task.updated_at = timestamp
    state.events.push({ at: timestamp, type: 'TASK_TRANSITION', task_id: task.id, detail: `${previous}->${target}` })

    const byId = new Map(state.tasks.map((item) => [item.id, item]))
    for (const candidate of state.tasks) {
      if (candidate.status === 'PENDING' && candidate.dependencies.every((id) => SUCCESS_STATES.has(byId.get(id)?.status))) {
        candidate.status = 'READY'
        candidate.updated_at = timestamp
        state.events.push({ at: timestamp, type: 'TASK_READY', task_id: candidate.id })
      }
    }
    state.run_status = deriveRunStatus(state)
    state.updated_at = timestamp
    atomicWriteJson(statePath, state)
    process.stdout.write(`${JSON.stringify({ task: task.id, status: task.status, run_status: state.run_status, ...stateSummary(state) }, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
