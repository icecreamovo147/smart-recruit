#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const MANIFEST_FENCE = 'code-review-sdd-manifest'
const PRIORITY_RANK = { P0: 0, P1: 1, P2: 2, P3: 3 }
const PROTECTED_PATHS = ['.git', '.code-review-sdd']
const FINDING_STATUSES = new Set(['OPEN', 'CLOSED'])
const GLOB_SYNTAX = /[*?\[\]{}()!+@]/

const usage = () => `Usage: node parse_code_review_sdd.mjs --input <code-review-sdd.md> [--output <plan.json>] [--task CR-001]... [--priority P1]...`

const asList = (value) => (Array.isArray(value) ? value : [])
const nonEmpty = (value) => String(value ?? '').trim().length > 0
const isRecord = (value) => value !== null && typeof value === 'object' && !Array.isArray(value)
const displayPath = (value) => JSON.stringify(value) ?? String(value)

const parseArgs = (argv) => {
  const args = { tasks: [], priorities: [] }
  for (let i = 2; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--input') args.input = argv[++i]
    else if (arg === '--output') args.output = argv[++i]
    else if (arg === '--task') args.tasks.push(argv[++i])
    else if (arg === '--priority') args.priorities.push(String(argv[++i] || '').toUpperCase())
    else if (arg === '--help' || arg === '-h') {
      console.log(usage())
      process.exit(0)
    } else {
      throw new Error(`Unknown argument: ${arg}`)
    }
  }
  if (!args.input) throw new Error('--input is required')
  return args
}

const extractManifest = (markdown) => {
  const pattern = new RegExp(`\`\`\`${MANIFEST_FENCE}\\s*\\n([\\s\\S]*?)\\n\`\`\``, 'g')
  const matches = [...markdown.matchAll(pattern)]
  if (matches.length !== 1) {
    throw new Error(`expected exactly one ${MANIFEST_FENCE} fence, found ${matches.length}`)
  }
  try {
    return JSON.parse(matches[0][1])
  } catch (error) {
    throw new Error(`invalid ${MANIFEST_FENCE} JSON: ${error.message}`)
  }
}

const isUnsafePath = (value) => {
  if (typeof value !== 'string') return true
  const text = value.trim()
  if (!text) return true
  if (/[\u0000-\u001F\u007F]/.test(text)) return true
  if (/^[A-Za-z][A-Za-z0-9+.-]*:/.test(text)) return true
  if (path.posix.isAbsolute(text) || path.win32.isAbsolute(text)) return true
  if (/^[A-Za-z]:/.test(text)) return true
  if (text.startsWith('~')) return true

  const normalized = path.posix.normalize(text.replaceAll('\\', '/'))
  if (normalized === '.' || normalized === './' || normalized === '..' || normalized.startsWith('../') || normalized.startsWith('/')) return true
  if (/^(?:\*{1,2})(?:\/\*{1,2})*$/.test(normalized)) return true
  const rootSegment = normalized.split('/')[0]
  if (GLOB_SYNTAX.test(rootSegment) && !/^[A-Za-z0-9]/.test(rootSegment)) return true

  const normalizedLower = normalized.toLowerCase()
  if (PROTECTED_PATHS.some((protectedPath) => (
    normalizedLower === protectedPath || normalizedLower.startsWith(`${protectedPath}/`)
  ))) return true
  return false
}

const orderedTasks = (tasks) => {
  const byId = new Map(tasks.map((task) => [task.id, task]))
  const visiting = new Set()
  const visited = new Set()
  const ordered = []

  const visit = (task) => {
    if (!task || visited.has(task.id)) return
    if (visiting.has(task.id)) throw new Error(`dependency cycle involving ${task.id}`)
    visiting.add(task.id)
    for (const dep of asList(task.dependencies)) {
      const parent = byId.get(dep)
      if (!parent) throw new Error(`unknown dependency ${dep} from ${task.id}`)
      visit(parent)
    }
    visiting.delete(task.id)
    visited.add(task.id)
    ordered.push(task)
  }

  const seeds = [...tasks].sort((left, right) => {
    const rank = (PRIORITY_RANK[left.priority] ?? 99) - (PRIORITY_RANK[right.priority] ?? 99)
    if (rank !== 0) return rank
    return String(left.id).localeCompare(String(right.id))
  })
  for (const task of seeds) visit(task)
  return ordered
}

const validateManifest = (manifest) => {
  const errors = []
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
    errors.push('manifest must be an object')
    return errors
  }
  if (manifest.schema_version !== 1) errors.push('schema_version must be 1')
  if (!manifest.review || typeof manifest.review !== 'object') errors.push('review must be an object')
  if (!Array.isArray(manifest.findings)) errors.push('findings must be an array')
  if (!Array.isArray(manifest.tasks)) errors.push('tasks must be an array')
  if ('no_action' in manifest && typeof manifest.no_action !== 'boolean') {
    errors.push('no_action must be a boolean')
  }
  if (manifest.no_action === true) {
    if (asList(manifest.findings).length !== 0) errors.push('no_action manifest must not contain findings')
    if (asList(manifest.tasks).length !== 0) errors.push('no_action manifest must not contain tasks')
  } else if (asList(manifest.tasks).length === 0) {
    errors.push('tasks must contain at least one task unless no_action is true')
  }

  const findingIds = new Set()
  for (const [index, finding] of asList(manifest.findings).entries()) {
    const prefix = `findings[${index}]`
    if (!isRecord(finding)) {
      errors.push(`${prefix} must be an object`)
      continue
    }
    if (!nonEmpty(finding?.id)) errors.push(`${prefix}.id is required`)
    else if (findingIds.has(finding.id)) errors.push(`duplicate finding id ${finding.id}`)
    else findingIds.add(finding.id)

    const status = String(finding?.status || 'OPEN').trim().toUpperCase()
    if (!FINDING_STATUSES.has(status)) {
      errors.push(`${prefix}.status must be OPEN or CLOSED`)
    } else {
      finding.status = status
    }
  }

  const taskIds = new Set()
  const referencedFindingIds = new Set()
  for (const [index, task] of asList(manifest.tasks).entries()) {
    const prefix = `tasks[${index}]`
    if (!nonEmpty(task?.id)) errors.push(`${prefix}.id is required`)
    else if (!/^CR-\d{3,}$/.test(task.id)) errors.push(`${prefix}.id must match CR-NNN`)
    else if (taskIds.has(task.id)) errors.push(`duplicate task id ${task.id}`)
    else taskIds.add(task.id)

    if (!['P0', 'P1', 'P2', 'P3'].includes(String(task?.priority || ''))) {
      errors.push(`${prefix}.priority must be P0-P3`)
    }
    if (!nonEmpty(task?.title)) errors.push(`${prefix}.title is required`)
    if (!Array.isArray(task?.finding_ids) || task.finding_ids.length === 0) {
      errors.push(`${prefix}.finding_ids must be a non-empty array`)
    } else {
      for (const findingId of task.finding_ids) {
        if (!findingIds.has(findingId)) errors.push(`${prefix} references unknown finding ${findingId}`)
        else referencedFindingIds.add(findingId)
      }
    }
    if (!task?.scope || typeof task.scope !== 'object') {
      errors.push(`${prefix}.scope is required`)
    } else {
      const allowed = asList(task.scope.allowed_paths)
      const excluded = asList(task.scope.excluded_paths)
      if (!allowed.length) errors.push(`${prefix}.scope.allowed_paths must not be empty`)
      for (const item of allowed) {
        if (isUnsafePath(item)) errors.push(`${prefix}.scope.allowed_paths contains unsafe path: ${displayPath(item)}`)
      }
      for (const item of excluded) {
        if (isUnsafePath(item)) errors.push(`${prefix}.scope.excluded_paths contains unsafe path: ${displayPath(item)}`)
      }
    }
    if (!Array.isArray(task?.steps)) errors.push(`${prefix}.steps must be an array`)
    if (!Array.isArray(task?.acceptance_criteria) || task.acceptance_criteria.length === 0) {
      errors.push(`${prefix}.acceptance_criteria must be a non-empty array`)
    }
    if (!Array.isArray(task?.tests)) errors.push(`${prefix}.tests must be an array`)
    if (!Array.isArray(task?.dependencies)) errors.push(`${prefix}.dependencies must be an array`)
  }

  for (const finding of asList(manifest.findings)) {
    if (nonEmpty(finding?.id) &&
        finding.status === 'OPEN' &&
        !referencedFindingIds.has(finding.id)) {
      errors.push(`open finding ${finding.id} is not referenced by any task`)
    }
  }

  if (!errors.length) {
    try {
      orderedTasks(manifest.tasks)
    } catch (error) {
      errors.push(error.message)
    }
  }
  return errors
}

const selectTasks = (tasks, args) => {
  let selected = [...tasks]
  if (args.tasks.length) {
    const wanted = new Set(args.tasks)
    selected = selected.filter((task) => wanted.has(task.id))
    for (const id of wanted) {
      if (!tasks.some((task) => task.id === id)) throw new Error(`unknown task selection: ${id}`)
    }
  }
  if (args.priorities.length) {
    const wanted = new Set(args.priorities)
    selected = selected.filter((task) => wanted.has(task.priority))
  }
  if (!selected.length) throw new Error('no tasks matched the selection')

  const byId = new Map(tasks.map((task) => [task.id, task]))
  const closure = new Map()
  const visit = (task) => {
    if (!task || closure.has(task.id)) return
    for (const dep of asList(task.dependencies)) visit(byId.get(dep))
    closure.set(task.id, task)
  }
  for (const task of selected) visit(task)
  return orderedTasks([...closure.values()])
}

const main = () => {
  let args
  try {
    args = parseArgs(process.argv)
  } catch (error) {
    console.error(`error: ${error.message}`)
    console.error(usage())
    process.exit(2)
  }

  try {
    const markdown = fs.readFileSync(args.input, 'utf8')
    const manifest = extractManifest(markdown)
    const errors = validateManifest(manifest)
    if (errors.length) {
      for (const error of errors) console.error(`error: ${error}`)
      process.exit(1)
    }

    if (manifest.no_action === true && (args.tasks.length || args.priorities.length)) {
      throw new Error('no tasks can be selected from a no_action manifest')
    }
    const ordered = manifest.no_action === true ? [] : selectTasks(manifest.tasks, args)
    const result = {
      plan_path: path.resolve(args.input),
      schema_version: manifest.schema_version,
      no_action: manifest.no_action === true,
      review: manifest.review,
      findings: manifest.findings,
      strategy: asList(manifest.strategy),
      tasks: ordered,
      task_count: ordered.length,
    }

    const json = `${JSON.stringify(result, null, 2)}\n`
    if (args.output) {
      fs.mkdirSync(path.dirname(args.output), { recursive: true })
      fs.writeFileSync(args.output, json, 'utf8')
      console.log(`plan_json: ${args.output}`)
    } else {
      process.stdout.write(json)
    }
    const suffix = manifest.no_action === true ? '; no_action' : ''
    console.error(`code_review_sdd_parse: PASS (${ordered.length} tasks${suffix})`)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.error(`error: ${message}`)
    process.exit(1)
  }
}

main()
