#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { asList, atomicWriteJson, cliValues, globMatches, now, parseCli } from './plan_utils.mjs'
import { changedPaths, git, snapshotRepository } from './repository_utils.mjs'
import { validateState } from './state_utils.mjs'

export const evaluateScope = (task, paths) => {
  const checked = [...new Set(paths.map((value) => value.replace(/^\.\//, '')))].sort()
  const excluded = checked.filter((candidate) => asList(task.excluded_paths).some((pattern) => globMatches(pattern, candidate)))
  const unauthorized = checked.filter((candidate) => !asList(task.allowed_paths).some((pattern) => globMatches(pattern, candidate)))
  return {
    task: task.id,
    checked_paths: checked,
    allowed_patterns: task.allowed_paths,
    excluded_patterns: task.excluded_paths,
    excluded_paths: excluded,
    unauthorized_paths: unauthorized,
    passed: excluded.length === 0 && unauthorized.length === 0,
  }
}

const main = () => {
try {
  const args = parseCli(process.argv, new Set(['help', 'captureBaseline', 'verify', 'record', 'replaceBaseline']))
  const modes = [args.captureBaseline, args.verify, args.path !== undefined].filter(Boolean).length
  if (args.help || !args.state || !args.task || modes !== 1) {
    process.stdout.write('Usage: check_fix_scope.mjs --state <state.json> --task KREM-001 (--capture-baseline [--replace-baseline] | --verify [--record] | --path <path>...)\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    validateState(state, statePath)
    const task = asList(state.tasks).find((item) => item.id === args.task)
    if (!task) throw new Error(`task not found in state: ${args.task}`)

    if (args.captureBaseline) {
      if (task.scope_baseline && !args.replaceBaseline) throw new Error('scope baseline already exists; use --replace-baseline only after reviewing prior task changes')
      if (!['REVALIDATING', 'IN_PROGRESS'].includes(task.status)) throw new Error('capture baseline only while task is REVALIDATING or IN_PROGRESS')
      task.scope_baseline = {
        at: now(),
        head: git(['rev-parse', 'HEAD'], state.repository.root),
        snapshot: snapshotRepository(state.repository.root),
      }
      task.scope_result = null
      task.updated_at = now()
      state.updated_at = task.updated_at
      state.events.push({ at: task.updated_at, type: 'SCOPE_BASELINE_CAPTURED', task_id: task.id })
      atomicWriteJson(statePath, state)
      process.stdout.write(`${JSON.stringify({ task: task.id, captured_at: task.scope_baseline.at, paths: Object.keys(task.scope_baseline.snapshot).length }, null, 2)}\n`)
    } else {
      let changed
      if (args.verify) {
        if (!task.scope_baseline?.snapshot) throw new Error('task has no scope baseline; capture it before editing')
        changed = changedPaths(task.scope_baseline.snapshot, snapshotRepository(state.repository.root))
        const stateRelative = path.relative(fs.realpathSync(state.repository.root), fs.realpathSync(statePath)).replaceAll('\\', '/')
        changed = changed.filter((candidate) => candidate !== stateRelative)
      } else changed = cliValues(args.path)
      const result = evaluateScope(task, changed)
      if (args.record) {
        if (!args.verify) throw new Error('--record is valid only with --verify')
        task.scope_result = { ...result, at: now() }
        task.modified_paths = result.checked_paths
        task.updated_at = task.scope_result.at
        state.updated_at = task.updated_at
        state.events.push({ at: task.updated_at, type: 'SCOPE_VERIFIED', task_id: task.id, detail: result.passed ? 'PASS' : 'FAIL' })
        atomicWriteJson(statePath, state)
      }
      process.stdout.write(`${JSON.stringify(result, null, 2)}\n`)
      if (!result.passed) process.exitCode = 1
    }
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main()
