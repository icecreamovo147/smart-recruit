#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { atomicWriteJson, loadPlanFile, now, orderedTasks, parseCli } from './plan_utils.mjs'

const git = (args) => execFileSync('git', args, { encoding: 'utf8' }).trim()

const worktreePaths = () => git(['status', '--porcelain=v1'])
  .split('\n').filter(Boolean)
  .flatMap((line) => {
    const raw = line.slice(3).trim()
    if (!raw.includes(' -> ')) return [raw]
    return raw.split(' -> ').map((value) => value.trim())
  })

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.plan || !args.output) {
    process.stdout.write('Usage: init_fix_state.mjs --plan <plan.md> --output <fix-state.json> [--task REM-001 | --priority P0]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const output = path.resolve(args.output)
    if (fs.existsSync(output)) throw new Error(`refusing to overwrite existing state: ${output}`)
    const planPath = path.resolve(args.plan)
    const loaded = loadPlanFile(planPath)
    const allTasks = orderedTasks(loaded.manifest.remediation.tasks)
    let tasks = allTasks
    let selection = { mode: 'all', values: [] }
    if (args.task) {
      const selected = new Set(Array.isArray(args.task) ? args.task : [args.task])
      if ([...selected].some((id) => !allTasks.some((task) => task.id === id))) throw new Error('one or more selected task IDs do not exist')
      const closure = new Set(selected)
      let changed = true
      while (changed) {
        changed = false
        for (const task of allTasks) {
          if (closure.has(task.id)) {
            for (const dependency of task.dependencies) {
              if (!closure.has(dependency)) { closure.add(dependency); changed = true }
            }
          }
        }
      }
      tasks = allTasks.filter((task) => closure.has(task.id))
      selection = { mode: 'task', values: [...selected] }
    } else if (args.priority) {
      const selected = new Set(Array.isArray(args.priority) ? args.priority : [args.priority])
      const closure = new Set(allTasks.filter((task) => selected.has(task.priority)).map((task) => task.id))
      if (closure.size === 0) throw new Error('no tasks match the selected priority')
      let changed = true
      while (changed) {
        changed = false
        for (const task of allTasks) {
          if (closure.has(task.id)) {
            for (const dependency of task.dependencies) {
              if (!closure.has(dependency)) { closure.add(dependency); changed = true }
            }
          }
        }
      }
      tasks = allTasks.filter((task) => closure.has(task.id))
      selection = { mode: 'priority', values: [...selected] }
    }
    const createdAt = now()
    const state = {
      schema_version: 1,
      run_id: `security-fix-${loaded.sha256.slice(0, 16)}`,
      plan: {
        path: planPath,
        sha256: loaded.sha256,
        source_format: loaded.manifest.source_format,
        audit_commit: loaded.manifest.audit?.commit || 'unknown',
        audit_verdict: loaded.manifest.audit?.verdict || 'unknown',
      },
      repository: {
        root: git(['rev-parse', '--show-toplevel']),
        branch: git(['branch', '--show-current']) || '(detached)',
        initial_head: git(['rev-parse', 'HEAD']),
        initial_worktree_paths: [...new Set(worktreePaths())].sort(),
      },
      run_status: 'IN_PROGRESS',
      selection,
      created_at: createdAt,
      updated_at: createdAt,
      tasks: tasks.map((task, index) => ({
        id: task.id,
        order: index,
        priority: task.priority,
        title: task.title,
        finding_ids: task.finding_ids,
        dependencies: task.dependencies.filter((id) => tasks.some((candidate) => candidate.id === id)),
        human_confirmation_required: task.human_confirmation_required,
        allowed_paths: task.scope.allowed_paths,
        excluded_paths: task.scope.excluded_paths,
        status: 'PENDING',
        modified_paths: [],
        notes: [],
        checks: [],
        updated_at: createdAt,
      })),
      events: [{ at: createdAt, type: 'RUN_INITIALIZED', detail: `selection=${selection.mode}` }],
    }
    atomicWriteJson(output, state)
    process.stdout.write(`${JSON.stringify({ state: output, run_id: state.run_id, tasks: state.tasks.map((task) => task.id) }, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
