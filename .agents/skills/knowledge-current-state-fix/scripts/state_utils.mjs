import path from 'node:path'
import { asList, loadPlanFile, SUCCESS_STATES, TASK_STATES, TERMINAL_STATES } from './plan_utils.mjs'
import { repositoryRoot } from './repository_utils.mjs'

export const stateSummary = (state) => {
  const byId = new Map(asList(state.tasks).map((task) => [task.id, task]))
  const ready = asList(state.tasks).filter((task) => task.status === 'READY').sort((a, b) => a.order - b.order)
  const dependencyBlocked = asList(state.tasks).filter((task) =>
    task.status === 'PENDING' && asList(task.dependencies).some((id) => TERMINAL_STATES.has(byId.get(id)?.status) && !SUCCESS_STATES.has(byId.get(id)?.status)))
  return {
    next_task: ready[0]?.id || null,
    ready_tasks: ready.map((task) => task.id),
    awaiting_confirmation: asList(state.tasks).filter((task) => task.status === 'AWAITING_CONFIRMATION').map((task) => task.id),
    dependency_blocked: dependencyBlocked.map((task) => task.id),
    terminal: asList(state.tasks).filter((task) => TERMINAL_STATES.has(task.status)).map((task) => task.id),
  }
}

export const validateState = (state, statePath) => {
  const errors = []
  if (state?.schema_version !== 1) errors.push('state schema_version must be 1')
  if (!state?.plan?.path || !state?.plan?.sha256) errors.push('state plan identity is incomplete')
  if (!state?.repository?.root || !state?.repository?.initial_head || !Array.isArray(state?.repository?.initial_worktree_paths)) errors.push('repository baseline is incomplete')
  if (!state?.repository?.initial_snapshot || typeof state.repository.initial_snapshot !== 'object') errors.push('repository fingerprint baseline is incomplete')
  if (!Array.isArray(state?.tasks) || !state.tasks.length) errors.push('state tasks must not be empty')
  const ids = new Set()
  for (const [index, task] of asList(state?.tasks).entries()) {
    if (ids.has(task.id)) errors.push(`duplicate task ID: ${task.id}`)
    ids.add(task.id)
    if (!TASK_STATES.has(task.status)) errors.push(`tasks[${index}] has invalid status ${task.status}`)
    if (!Array.isArray(task.allowed_paths) || !Array.isArray(task.excluded_paths)) errors.push(`tasks[${index}] scope is invalid`)
    if (!Array.isArray(task.effective_confirmation_reasons)) errors.push(`tasks[${index}] confirmation metadata is invalid`)
  }
  for (const task of asList(state?.tasks)) for (const dependency of asList(task.dependencies)) {
    if (!ids.has(dependency)) errors.push(`${task.id} references unavailable selected dependency ${dependency}`)
  }
  try {
    const loaded = loadPlanFile(state?.plan?.path)
    if (loaded.sha256 !== state?.plan?.sha256) errors.push('remediation plan hash changed; start a new run or explicitly reconcile the plan')
  } catch (error) { errors.push(error.message) }
  if (statePath) {
    const actualRoot = repositoryRoot(path.dirname(path.resolve(statePath)))
    if (path.resolve(actualRoot) !== path.resolve(state?.repository?.root || '')) errors.push('state belongs to a different repository')
  }
  if (errors.length) throw new Error(`knowledge fix state validation failed:\n- ${errors.join('\n- ')}`)
  return stateSummary(state)
}
