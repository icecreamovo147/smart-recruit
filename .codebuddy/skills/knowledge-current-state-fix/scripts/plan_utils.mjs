import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

export const PRIORITIES = ['P0', 'P1', 'P2', 'P3']
export const RESOLUTION_TARGETS = ['KNOWLEDGE', 'SOURCE', 'DECISION_REQUIRED']
export const KNOWLEDGE_LEVELS = ['L1', 'L2', 'L3']
export const SUCCESS_STATES = new Set(['PASSED', 'ALREADY_RESOLVED', 'ACCEPTED'])
export const TERMINAL_STATES = new Set([
  ...SUCCESS_STATES, 'FAILED', 'BLOCKED', 'STALE_PLAN', 'SCOPE_GAP', 'DEFERRED', 'ABORTED',
])
export const TASK_STATES = new Set([
  'PENDING', 'READY', 'REVALIDATING', 'AWAITING_CONFIRMATION', 'IN_PROGRESS', 'VERIFYING',
  ...TERMINAL_STATES,
])

export const asList = (value) => Array.isArray(value) ? value : []
export const now = () => new Date().toISOString()
export const readText = (file) => fs.readFileSync(file, 'utf8')
export const sha256 = (value) => crypto.createHash('sha256').update(value).digest('hex')

const nonEmpty = (value) => typeof value === 'string' && value.trim() !== ''

export const safeRepoPattern = (value) => {
  if (!nonEmpty(value) || path.isAbsolute(value) || value.includes('\\') || value.includes('\0')) return false
  const parts = value.replace(/^\.\//, '').split('/')
  return !parts.includes('..') && !parts.includes('.') && parts.every(Boolean)
}

export const effectiveConfirmationReasons = (task) => {
  const reasons = []
  if (task.human_confirmation_required) reasons.push('PLAN_FLAG')
  if (task.resolution_target === 'SOURCE') reasons.push('SOURCE_TARGET')
  if (task.resolution_target === 'DECISION_REQUIRED') reasons.push('DECISION_REQUIRED')
  if (task.knowledge_level === 'L3') reasons.push('L3')
  return [...new Set(reasons)]
}

export const validateManifest = (manifest) => {
  const errors = []
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) errors.push('manifest must be an object')
  if (manifest?.schema_version !== 1) errors.push('schema_version must be 1')
  for (const field of ['commit', 'working_tree', 'verdict', 'generated_at']) {
    if (!nonEmpty(manifest?.audit?.[field])) errors.push(`audit.${field} must be non-empty`)
  }
  if (!Array.isArray(manifest?.findings)) errors.push('findings must be an array')
  if (!Array.isArray(manifest?.tasks) || manifest.tasks.length === 0) errors.push('tasks must contain at least one task')

  const findingIds = new Set()
  for (const [index, item] of asList(manifest?.findings).entries()) {
    const prefix = `findings[${index}]`
    if (!/^KNO-\d{3,}$/.test(item?.id || '')) errors.push(`${prefix}.id must match KNO-NNN`)
    if (findingIds.has(item?.id)) errors.push(`duplicate finding id: ${item?.id}`)
    findingIds.add(item?.id)
    for (const field of ['title', 'type', 'severity', 'status']) if (!nonEmpty(item?.[field])) errors.push(`${prefix}.${field} must be non-empty`)
    if (!['OPEN', 'RESOLVED', 'ACCEPTED'].includes(item?.status)) errors.push(`${prefix}.status is invalid`)
    if (!RESOLUTION_TARGETS.includes(item?.resolution_target)) errors.push(`${prefix}.resolution_target is invalid`)
  }

  const taskIds = new Set()
  const mappedFindings = new Set()
  for (const [index, task] of asList(manifest?.tasks).entries()) {
    const prefix = `tasks[${index}]`
    if (!/^KREM-\d{3,}$/.test(task?.id || '')) errors.push(`${prefix}.id must match KREM-NNN`)
    if (taskIds.has(task?.id)) errors.push(`duplicate task id: ${task?.id}`)
    taskIds.add(task?.id)
    if (!PRIORITIES.includes(task?.priority)) errors.push(`${prefix}.priority is invalid`)
    if (!RESOLUTION_TARGETS.includes(task?.resolution_target)) errors.push(`${prefix}.resolution_target is invalid`)
    if (!KNOWLEDGE_LEVELS.includes(task?.knowledge_level)) errors.push(`${prefix}.knowledge_level is invalid`)
    for (const field of ['title', 'objective', 'rationale', 'status', 'owner_hint', 'rollback']) {
      if (!nonEmpty(task?.[field])) errors.push(`${prefix}.${field} must be non-empty`)
    }
    if (task?.status !== 'PENDING') errors.push(`${prefix}.status must be PENDING`)
    for (const field of ['finding_ids', 'dependencies', 'source_evidence', 'steps', 'acceptance_criteria', 'tests', 'deliverables']) {
      if (!Array.isArray(task?.[field])) errors.push(`${prefix}.${field} must be an array`)
    }
    if (!task?.scope || typeof task.scope !== 'object' || Array.isArray(task.scope)) errors.push(`${prefix}.scope must be an object`)
    for (const field of ['allowed_paths', 'excluded_paths']) {
      if (!Array.isArray(task?.scope?.[field])) errors.push(`${prefix}.scope.${field} must be an array`)
      for (const pattern of asList(task?.scope?.[field])) {
        if (!safeRepoPattern(pattern)) errors.push(`${prefix}.scope.${field} contains unsafe pattern: ${pattern}`)
      }
    }
    if (asList(task?.scope?.allowed_paths).length === 0) errors.push(`${prefix}.scope.allowed_paths must not be empty`)
    for (const field of ['finding_ids', 'source_evidence', 'steps', 'acceptance_criteria', 'tests', 'deliverables']) {
      if (asList(task?.[field]).length === 0) errors.push(`${prefix}.${field} must not be empty`)
    }
    if (typeof task?.parallel_safe !== 'boolean') errors.push(`${prefix}.parallel_safe must be boolean`)
    if (typeof task?.human_confirmation_required !== 'boolean') errors.push(`${prefix}.human_confirmation_required must be boolean`)
    for (const findingId of asList(task?.finding_ids)) {
      if (!findingIds.has(findingId)) errors.push(`${prefix} references unknown finding ${findingId}`)
      const finding = asList(manifest?.findings).find((item) => item.id === findingId)
      if (finding && finding.resolution_target !== task.resolution_target) errors.push(`${prefix} target conflicts with finding ${findingId}`)
      mappedFindings.add(findingId)
    }
  }

  for (const [index, task] of asList(manifest?.tasks).entries()) {
    for (const dependency of asList(task.dependencies)) {
      if (!taskIds.has(dependency)) errors.push(`tasks[${index}] references unknown dependency ${dependency}`)
      if (dependency === task.id) errors.push(`${task.id} cannot depend on itself`)
    }
  }
  for (const finding of asList(manifest?.findings)) {
    if (finding.status === 'OPEN' && !mappedFindings.has(finding.id)) errors.push(`open finding ${finding.id} has no remediation task`)
  }
  if (errors.length === 0) {
    try { orderedTasks(manifest.tasks) } catch (error) { errors.push(error.message) }
  }
  if (errors.length) throw new Error(`knowledge remediation plan validation failed:\n- ${errors.join('\n- ')}`)
  return manifest
}

export const extractPlan = (markdown) => {
  const fences = [...markdown.matchAll(/```knowledge-remediation-manifest\s*\n([\s\S]*?)\n```/g)]
  if (fences.length !== 1) throw new Error(`expected exactly one knowledge-remediation-manifest fence; found ${fences.length}`)
  let manifest
  try { manifest = JSON.parse(fences[0][1].trim()) } catch (error) {
    throw new Error(`invalid knowledge-remediation-manifest JSON: ${error.message}`)
  }
  return validateManifest(manifest)
}

export const loadPlanFile = (file) => {
  const markdown = readText(file)
  return { manifest: extractPlan(markdown), markdown, sha256: sha256(markdown) }
}

export function orderedTasks(tasks) {
  const emitted = new Set()
  const result = []
  while (result.length < tasks.length) {
    const ready = tasks
      .map((task, index) => ({ task, index }))
      .filter(({ task }) => !emitted.has(task.id) && asList(task.dependencies).every((id) => emitted.has(id)))
      .sort((left, right) => PRIORITIES.indexOf(left.task.priority) - PRIORITIES.indexOf(right.task.priority) || left.index - right.index)
    if (!ready.length) {
      const remaining = tasks.filter((task) => !emitted.has(task.id)).map((task) => task.id)
      throw new Error(`task dependency cycle detected: ${remaining.join(', ')}`)
    }
    for (const item of ready) {
      emitted.add(item.task.id)
      result.push(item.task)
    }
  }
  return result
}

const regexEsc = (value) => value.replace(/[.+^${}()|[\]\\]/g, '\\$&')

export const globMatches = (pattern, candidate) => {
  const normalizedPattern = pattern.replace(/^\.\//, '')
  const normalizedCandidate = candidate.replace(/^\.\//, '')
  let source = ''
  for (let index = 0; index < normalizedPattern.length; index += 1) {
    const char = normalizedPattern[index]
    if (char === '*' && normalizedPattern[index + 1] === '*') {
      const followedBySlash = normalizedPattern[index + 2] === '/'
      source += followedBySlash ? '(?:.*/)?' : '.*'
      index += followedBySlash ? 2 : 1
    } else if (char === '*') source += '[^/]*'
    else if (char === '?') source += '[^/]'
    else source += regexEsc(char)
  }
  return new RegExp(`^${source}$`).test(normalizedCandidate)
}

export const atomicWriteJson = (file, value) => {
  const directory = path.dirname(file)
  fs.mkdirSync(directory, { recursive: true })
  const temp = path.join(directory, `.${path.basename(file)}.${process.pid}.tmp`)
  fs.writeFileSync(temp, `${JSON.stringify(value, null, 2)}\n`, 'utf8')
  fs.renameSync(temp, file)
}

export const parseCli = (argv, booleanFlags = new Set()) => {
  const args = { _: [] }
  for (let index = 2; index < argv.length; index += 1) {
    const token = argv[index]
    if (!token.startsWith('--')) args._.push(token)
    else {
      const key = token.slice(2).replace(/-([a-z])/g, (_, char) => char.toUpperCase())
      if (booleanFlags.has(key)) args[key] = true
      else {
        const value = argv[++index]
        if (value === undefined) throw new Error(`${token} requires a value`)
        if (args[key] === undefined) args[key] = value
        else if (Array.isArray(args[key])) args[key].push(value)
        else args[key] = [args[key], value]
      }
    }
  }
  return args
}

export const cliValues = (value) => value === undefined ? [] : Array.isArray(value) ? value : [value]
