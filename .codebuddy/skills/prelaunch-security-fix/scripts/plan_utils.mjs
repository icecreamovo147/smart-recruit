import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

export const PRIORITIES = ['P0', 'P1', 'P2', 'P3']
export const SUCCESS_STATES = new Set(['PASSED', 'ALREADY_RESOLVED', 'ACCEPTED_RISK'])
export const TERMINAL_STATES = new Set([
  'PASSED', 'ALREADY_RESOLVED', 'FAILED', 'BLOCKED', 'STALE', 'SCOPE_GAP',
  'DEFERRED', 'ACCEPTED_RISK', 'ABORTED',
])
export const TASK_STATES = new Set([
  'PENDING', 'READY', 'AWAITING_CONFIRMATION', 'IN_PROGRESS', 'VERIFYING',
  ...TERMINAL_STATES,
])

export const sha256 = (value) => crypto.createHash('sha256').update(value).digest('hex')
export const readText = (file) => fs.readFileSync(file, 'utf8')
export const asList = (value) => Array.isArray(value) ? value : []
export const now = () => new Date().toISOString()

const nonEmpty = (value) => typeof value === 'string' && value.trim() !== ''

const safeRepoPattern = (value) => {
  if (!nonEmpty(value) || path.isAbsolute(value) || value.includes('\\')) return false
  const parts = value.split('/')
  return !parts.includes('..') && !parts.includes('.')
}

const parseListSection = (block, heading) => {
  const escaped = heading.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = block.match(new RegExp(`^### ${escaped}\\s*\\n([\\s\\S]*?)(?=^### |^## |(?![\\s\\S]))`, 'm'))
  if (!match) return []
  return match[1].split('\n')
    .map((line) => line.trim())
    .filter((line) => /^[-*]\s+/.test(line) || /^\d+\.\s+/.test(line))
    .map((line) => line.replace(/^[-*]\s+/, '').replace(/^\d+\.\s+/, '').trim())
    .filter((line) => line && line !== '无' && line !== '未提供')
}

const parseTextSection = (block, heading) => {
  const escaped = heading.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = block.match(new RegExp(`^### ${escaped}\\s*\\n([\\s\\S]*?)(?=^### |^## |(?![\\s\\S]))`, 'm'))
  if (!match) return ''
  return match[1].trim().replace(/^[-*]\s+/, '')
}

const tableValue = (block, label) => {
  const rows = [...block.matchAll(/^\|\s*([^|]+?)\s*\|\s*([^|]*?)\s*\|\s*$/gm)]
  const row = rows.find((entry) => entry[1].trim() === label)
  return row ? row[2].trim() : ''
}

const splitValues = (value) => {
  if (!value || ['无', '未提供', '未映射'].includes(value)) return []
  return value.split(/[,，]/).map((item) => item.trim()).filter(Boolean)
}

const legacyManifest = (markdown) => {
  const taskMatches = [...markdown.matchAll(/^##\s+(REM-\d{3,})\s+·\s+(.+)$/gm)]
  if (taskMatches.length === 0) return null
  const tasks = taskMatches.map((match, index) => {
    const start = match.index
    const end = index + 1 < taskMatches.length ? taskMatches[index + 1].index : markdown.length
    const block = markdown.slice(start, end)
    return {
      id: match[1],
      finding_ids: splitValues(tableValue(block, '关联发现')),
      priority: tableValue(block, '优先级'),
      title: match[2].trim(),
      objective: parseTextSection(block, '目标'),
      rationale: parseTextSection(block, '安全理由'),
      status: tableValue(block, '状态') || 'PENDING',
      owner_hint: tableValue(block, '建议负责人'),
      dependencies: splitValues(tableValue(block, '依赖')),
      parallel_safe: tableValue(block, '可并行') === '是',
      human_confirmation_required: tableValue(block, '需要人工确认') === '是',
      scope: {
        allowed_paths: parseListSection(block, '允许范围'),
        excluded_paths: parseListSection(block, '排除范围'),
      },
      steps: parseListSection(block, '实施步骤'),
      acceptance_criteria: parseListSection(block, '验收标准'),
      tests: parseListSection(block, '验证命令/测试'),
      rollback: parseTextSection(block, '回滚或安全遏制'),
      security_notes: parseListSection(block, '安全注意事项'),
      deliverables: parseListSection(block, '必须交付'),
    }
  })
  const branchRepo = tableValue(markdown, '仓库/分支').split('/').map((value) => value.trim())
  return {
    schema_version: 1,
    artifact_type: 'prelaunch-security-remediation-plan',
    source_format: 'legacy-markdown',
    audit: {
      title: tableValue(markdown, '来源审查') || 'Legacy remediation plan',
      repo: branchRepo[0] || 'unknown',
      branch: branchRepo.slice(1).join('/') || 'unknown',
      commit: tableValue(markdown, '审查提交') || 'unknown',
      generated_at: tableValue(markdown, '生成时间') || 'unknown',
      verdict: tableValue(markdown, '审查结论') || 'unknown',
      overall_risk: tableValue(markdown, '整体风险') || 'unknown',
    },
    findings: [...new Set(tasks.flatMap((task) => task.finding_ids))].map((id) => ({ id })),
    remediation: { strategy: parseListSection(markdown, '修复策略'), tasks },
  }
}

export const validateManifest = (manifest) => {
  const errors = []
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) errors.push('manifest must be an object')
  if (manifest?.schema_version !== 1) errors.push('schema_version must be 1')
  if (manifest?.artifact_type !== 'prelaunch-security-remediation-plan') errors.push('artifact_type is invalid')
  if (!manifest?.audit || typeof manifest.audit !== 'object') errors.push('audit must be an object')
  if (!manifest?.remediation || typeof manifest.remediation !== 'object') errors.push('remediation must be an object')
  const tasks = asList(manifest?.remediation?.tasks)
  if (tasks.length === 0) errors.push('remediation.tasks must contain at least one task')
  const ids = new Set()
  tasks.forEach((task, index) => {
    const prefix = `tasks[${index}]`
    if (!/^REM-\d{3,}$/.test(task?.id || '')) errors.push(`${prefix}.id must match REM-NNN`)
    if (ids.has(task?.id)) errors.push(`duplicate task id: ${task?.id}`)
    ids.add(task?.id)
    if (!PRIORITIES.includes(task?.priority)) errors.push(`${prefix}.priority is invalid`)
    for (const field of ['title', 'objective', 'rationale', 'status', 'rollback']) {
      if (!nonEmpty(task?.[field])) errors.push(`${prefix}.${field} must be non-empty`)
    }
    for (const field of ['finding_ids', 'dependencies', 'steps', 'acceptance_criteria', 'tests', 'security_notes', 'deliverables']) {
      if (!Array.isArray(task?.[field])) errors.push(`${prefix}.${field} must be an array`)
    }
    if (!task?.scope || typeof task.scope !== 'object') errors.push(`${prefix}.scope must be an object`)
    for (const field of ['allowed_paths', 'excluded_paths']) {
      if (!Array.isArray(task?.scope?.[field])) errors.push(`${prefix}.scope.${field} must be an array`)
      for (const pattern of asList(task?.scope?.[field])) {
        if (!safeRepoPattern(pattern)) errors.push(`${prefix}.scope.${field} contains unsafe pattern: ${pattern}`)
      }
    }
    if (asList(task?.scope?.allowed_paths).length === 0) errors.push(`${prefix}.scope.allowed_paths must not be empty`)
    if (typeof task?.parallel_safe !== 'boolean') errors.push(`${prefix}.parallel_safe must be boolean`)
    if (typeof task?.human_confirmation_required !== 'boolean') errors.push(`${prefix}.human_confirmation_required must be boolean`)
  })
  tasks.forEach((task, index) => {
    for (const dependency of asList(task.dependencies)) {
      if (!ids.has(dependency)) errors.push(`tasks[${index}] references unknown dependency ${dependency}`)
      if (dependency === task.id) errors.push(`${task.id} cannot depend on itself`)
    }
  })
  if (errors.length === 0) {
    try { orderedTasks(tasks) } catch (error) { errors.push(error.message) }
  }
  if (errors.length > 0) throw new Error(`remediation plan validation failed:\n- ${errors.join('\n- ')}`)
  return manifest
}

export const extractPlan = (markdown) => {
  const fences = [...markdown.matchAll(/```(?:json|prelaunch-security-remediation-manifest)?\s*\n([\s\S]*?)\n```/g)]
  for (const fence of fences) {
    const body = fence[1].trim()
    if (!body.startsWith('{')) continue
    try {
      const candidate = JSON.parse(body)
      if (candidate?.artifact_type === 'prelaunch-security-remediation-plan') {
        candidate.source_format ||= 'embedded-manifest'
        return validateManifest(candidate)
      }
    } catch {
      // Ignore unrelated JSON examples; validation happens for the matching artifact.
    }
  }
  const legacy = legacyManifest(markdown)
  if (!legacy) throw new Error('no supported remediation manifest or legacy generated task sections found')
  return validateManifest(legacy)
}

export const loadPlanFile = (file) => {
  const markdown = readText(file)
  return {
    manifest: extractPlan(markdown),
    markdown,
    sha256: sha256(markdown),
  }
}

export function orderedTasks(tasks) {
  const byId = new Map(tasks.map((task, index) => [task.id, { task, index }]))
  const emitted = new Set()
  const result = []
  while (result.length < tasks.length) {
    const ready = tasks
      .map((task, index) => ({ task, index }))
      .filter(({ task }) => !emitted.has(task.id) && asList(task.dependencies).every((id) => emitted.has(id)))
      .sort((left, right) => PRIORITIES.indexOf(left.task.priority) - PRIORITIES.indexOf(right.task.priority) || left.index - right.index)
    if (ready.length === 0) {
      const remaining = [...byId.keys()].filter((id) => !emitted.has(id))
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
  fs.writeFileSync(temp, `${JSON.stringify(value, null, 2)}\n`, { encoding: 'utf8', flag: 'w' })
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
