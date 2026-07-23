#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { atomicWriteJson, cliValues, effectiveConfirmationReasons, globMatches, loadPlanFile, now, orderedTasks, parseCli } from './plan_utils.mjs'
import { git, repositoryRoot, snapshotRepository, worktreePaths } from './repository_utils.mjs'

const dependencyClosure = (allTasks, selectedIds) => {
  const closure = new Set(selectedIds)
  let changed = true
  while (changed) {
    changed = false
    for (const task of allTasks) if (closure.has(task.id)) for (const dependency of task.dependencies) {
      if (!closure.has(dependency)) { closure.add(dependency); changed = true }
    }
  }
  return closure
}

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.plan || !args.output) {
    process.stdout.write('Usage: init_fix_state.mjs --plan <plan.md> --output <fix-state.json> [--task KREM-001]... [--priority P1]...\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const output = path.resolve(args.output)
    if (fs.existsSync(output)) throw new Error(`refusing to overwrite existing state: ${output}`)
    const planPath = path.resolve(args.plan)
    const loaded = loadPlanFile(planPath)
    const allTasks = orderedTasks(loaded.manifest.tasks)
    const taskSelection = cliValues(args.task)
    const prioritySelection = cliValues(args.priority)
    if (taskSelection.length && prioritySelection.length) throw new Error('select tasks or priorities, not both')

    let selection = { mode: 'all', values: [] }
    let selectedIds = new Set(allTasks.map((task) => task.id))
    if (taskSelection.length) {
      const unknown = taskSelection.filter((id) => !allTasks.some((task) => task.id === id))
      if (unknown.length) throw new Error(`unknown selected task IDs: ${unknown.join(', ')}`)
      selectedIds = dependencyClosure(allTasks, taskSelection)
      selection = { mode: 'task', values: taskSelection }
    } else if (prioritySelection.length) {
      const invalid = prioritySelection.filter((priority) => !['P0', 'P1', 'P2', 'P3'].includes(priority))
      if (invalid.length) throw new Error(`invalid priorities: ${invalid.join(', ')}`)
      const matches = allTasks.filter((task) => prioritySelection.includes(task.priority)).map((task) => task.id)
      if (!matches.length) throw new Error('no tasks match the selected priority')
      selectedIds = dependencyClosure(allTasks, matches)
      selection = { mode: 'priority', values: prioritySelection }
    }
    const tasks = allTasks.filter((task) => selectedIds.has(task.id))
    const root = repositoryRoot(path.dirname(planPath))
    const dirty = worktreePaths(root)
    const snapshot = snapshotRepository(root)
    const createdAt = now()
    const state = {
      schema_version: 1,
      run_id: `knowledge-fix-${loaded.sha256.slice(0, 12)}-${createdAt.replace(/\D/g, '').slice(0, 14)}`,
      plan: {
        path: planPath,
        sha256: loaded.sha256,
        audit_commit: loaded.manifest.audit.commit,
        audit_working_tree: loaded.manifest.audit.working_tree,
        audit_verdict: loaded.manifest.audit.verdict,
        audit_generated_at: loaded.manifest.audit.generated_at,
      },
      repository: {
        root,
        branch: git(['branch', '--show-current'], root) || '(detached)',
        initial_head: git(['rev-parse', 'HEAD'], root),
        initial_worktree_paths: dirty,
        initial_snapshot: snapshot,
      },
      run_status: 'IN_PROGRESS',
      selection,
      created_at: createdAt,
      updated_at: createdAt,
      tasks: tasks.map((task, index) => {
        const dependencies = task.dependencies.filter((id) => selectedIds.has(id))
        const reasons = effectiveConfirmationReasons(task)
        const allowedOverlap = dirty.filter((candidate) => task.scope.allowed_paths.some((pattern) => globMatches(pattern, candidate)))
        const excludedOverlap = dirty.filter((candidate) => task.scope.excluded_paths.some((pattern) => globMatches(pattern, candidate)))
        return {
          id: task.id,
          order: index,
          priority: task.priority,
          title: task.title,
          finding_ids: task.finding_ids,
          resolution_target: task.resolution_target,
          knowledge_level: task.knowledge_level,
          dependencies,
          parallel_safe: task.parallel_safe,
          manifest_confirmation_required: task.human_confirmation_required,
          effective_confirmation_required: reasons.length > 0,
          effective_confirmation_reasons: reasons,
          allowed_paths: task.scope.allowed_paths,
          excluded_paths: task.scope.excluded_paths,
          preexisting_allowed_overlap: allowedOverlap,
          preexisting_excluded_overlap: excludedOverlap,
          status: dependencies.length ? 'PENDING' : 'READY',
          modified_paths: [],
          confirmations: [],
          notes: [],
          checks: [],
          scope_baseline: null,
          scope_result: null,
          updated_at: createdAt,
        }
      }),
      events: [{ at: createdAt, type: 'RUN_INITIALIZED', detail: `selection=${selection.mode}` }],
    }
    atomicWriteJson(output, state)
    process.stdout.write(`${JSON.stringify({ state: output, run_id: state.run_id, tasks: state.tasks.map((task) => task.id), gated_tasks: state.tasks.filter((task) => task.effective_confirmation_required).map((task) => task.id) }, null, 2)}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
