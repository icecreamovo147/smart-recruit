#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { asList, atomicWriteJson, now, parseCli, SUCCESS_STATES, TERMINAL_STATES } from './plan_utils.mjs'
import { stateSummary, validateState } from './state_utils.mjs'

const TRANSITIONS = {
  PENDING: new Set(['READY', 'BLOCKED', 'STALE', 'DEFERRED']),
  READY: new Set(['AWAITING_CONFIRMATION', 'IN_PROGRESS', 'ALREADY_RESOLVED', 'BLOCKED', 'STALE', 'SCOPE_GAP']),
  AWAITING_CONFIRMATION: new Set(['IN_PROGRESS', 'DEFERRED', 'ABORTED']),
  IN_PROGRESS: new Set(['VERIFYING', 'BLOCKED', 'STALE', 'SCOPE_GAP', 'ABORTED']),
  VERIFYING: new Set(['PASSED', 'IN_PROGRESS', 'FAILED', 'BLOCKED', 'SCOPE_GAP', 'ABORTED']),
  FAILED: new Set(['READY']), BLOCKED: new Set(['READY']), STALE: new Set(['READY']),
  SCOPE_GAP: new Set(['READY']), DEFERRED: new Set(['READY', 'ACCEPTED_RISK']),
}
const REQUIRED_PASS_CHECKS = ['finding', 'acceptance', 'regression', 'scope', 'security']

const values = (value) => value === undefined ? [] : Array.isArray(value) ? value : [value]

const parseCheck = (raw) => {
  const match = String(raw).match(/^([a-zA-Z0-9_-]+)=(PASS|FAIL|NOT_RUN|BLOCKED)(?::([\s\S]*))?$/)
  if (!match) throw new Error(`invalid --check value: ${raw}; expected name=PASS|FAIL|NOT_RUN|BLOCKED[:detail]`)
  return { name: match[1], status: match[2], detail: match[3] || '', at: now() }
}

const deriveRunStatus = (state) => {
  const tasks = asList(state.tasks)
  if (tasks.some((task) => task.status === 'AWAITING_CONFIRMATION')) return 'AWAITING_CONFIRMATION'
  if (!tasks.every((task) => TERMINAL_STATES.has(task.status))) return 'IN_PROGRESS'
  if (state.selection.mode === 'all' && tasks.every((task) => SUCCESS_STATES.has(task.status))) return 'REMEDIATION_COMPLETE_PENDING_REAUDIT'
  if (tasks.some((task) => SUCCESS_STATES.has(task.status))) return 'REMEDIATION_PARTIAL'
  return 'REMEDIATION_BLOCKED'
}

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.state || !args.task || !args.status) {
    process.stdout.write('Usage: update_fix_state.mjs --state <state.json> --task REM-001 --status <STATE> [--note text] [--modified-path path] [--check name=PASS:detail]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    validateState(state)
    const task = state.tasks.find((item) => item.id === args.task)
    if (!task) throw new Error(`task not found in state: ${args.task}`)
    const target = args.status
    if (!TRANSITIONS[task.status]?.has(target)) throw new Error(`invalid transition ${task.status} -> ${target}`)

    const timestamp = now()
    task.notes.push(...values(args.note).map((note) => ({ at: timestamp, text: note })))
    task.modified_paths = [...new Set([...task.modified_paths, ...values(args.modifiedPath)])].sort()
    for (const raw of values(args.check)) {
      const check = parseCheck(raw)
      task.checks = task.checks.filter((item) => item.name !== check.name)
      task.checks.push(check)
    }
    if (['PASSED', 'ALREADY_RESOLVED'].includes(target)) {
      const passed = new Set(task.checks.filter((check) => check.status === 'PASS').map((check) => check.name))
      const missing = REQUIRED_PASS_CHECKS.filter((name) => !passed.has(name))
      if (missing.length > 0) throw new Error(`${target} requires PASS checks: ${missing.join(', ')}`)
    }
    task.status = target
    task.updated_at = timestamp
    state.events.push({ at: timestamp, type: 'TASK_TRANSITION', task_id: task.id, detail: `${task.status}` })

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
