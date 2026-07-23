#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'

const AUDIT_VERDICTS = ['UNRELIABLE', 'DRIFT_DETECTED', 'CURRENT_WITH_GAPS', 'CURRENT']
const RISKS = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW']
const COVERAGE_STATUSES = ['REVIEWED', 'PARTIAL', 'NOT_REVIEWED', 'NOT_APPLICABLE']
const CHECK_STATUSES = ['PASS', 'FAIL', 'NOT_RUN', 'BLOCKED', 'NOT_APPLICABLE']
const DOCUMENT_VERDICTS = ['UNCHANGED', 'STALE', 'CONFLICT', 'CANDIDATE', 'INVALID']
const FINDING_TYPES = ['STALE', 'CONFLICT', 'INVALID', 'COVERAGE_GAP', 'EVIDENCE_GAP']
const CONFIDENCES = ['CONFIRMED', 'HIGH', 'MEDIUM', 'LOW']
const FINDING_STATUSES = ['OPEN', 'RESOLVED', 'ACCEPTED']
const RESOLUTION_TARGETS = ['KNOWLEDGE', 'SOURCE', 'DECISION_REQUIRED']
const ATTRIBUTIONS = ['PRE_EXISTING', 'WORKTREE_INTRODUCED', 'MIXED', 'UNKNOWN']
const PRIORITIES = ['P0', 'P1', 'P2', 'P3']
const KNOWLEDGE_LEVELS = ['L1', 'L2', 'L3']
const REQUIRED_COVERAGE = [
  ['governance-tooling', '知识治理与工具链'],
  ['architecture-service-boundaries', '架构与服务边界'],
  ['frontend-shared-contracts', '前端与共享契约'],
  ['gateway-api-protobuf', 'Gateway、API 与 Proto'],
  ['persistence-migrations', '持久化与迁移'],
  ['auth-rbac-security', '认证、RBAC 与安全'],
  ['recruitment-notification', '招聘生命周期与通知'],
  ['ai-agent-mcp-retrieval', 'AI、Agent、MCP 与检索'],
  ['resume-sensitive-data', '简历与敏感数据'],
  ['billing-payment-credits', '账单、支付与积分'],
  ['development-deployment-operations', '开发、部署与运维'],
  ['cross-document-route-coverage', '跨文档一致性与路由覆盖'],
]

const esc = (value) => String(value ?? '')
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')
  .replaceAll("'", '&#39;')

const md = (value) => String(value ?? '')
  .replaceAll('|', '\\|')
  .replaceAll('\r', ' ')
  .replaceAll('\n', '<br>')

const values = (value) => Array.isArray(value) ? value : []
const valueText = (value, fallback = '未提供') => String(value ?? '').trim() || fallback
const code = (value) => `\`${String(value ?? '').replaceAll('`', '\\`')}\``
const joinMd = (items, fallback = '无') => values(items).length ? values(items).map(code).join('、') : fallback
const joinText = (items, fallback = '无') => values(items).length ? values(items).join('；') : fallback

const parseArgs = (argv) => {
  const args = { outputDir: '.knowledge-review' }
  for (let index = 2; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--sample') args.sample = true
    else if (arg === '--input') args.input = argv[++index]
    else if (arg === '--output-dir') args.outputDir = argv[++index]
    else if (arg === '--basename') args.basename = argv[++index]
    else if (arg === '--help' || arg === '-h') args.help = true
    else throw new Error(`unknown argument: ${arg}`)
  }
  return args
}

const usage = () => `Usage:
  node render_knowledge_audit.mjs --input <audit.json> [--output-dir <dir>] [--basename <name>]
  node render_knowledge_audit.mjs --sample [--output-dir <dir>] [--basename <name>]
`

const samplePayload = () => ({
  schema_version: 1,
  audit: {
    title: 'Smart Recruit 项目知识库现状审计报告',
    repo: 'smart-recruit',
    branch: 'main',
    commit: 'sample-commit',
    working_tree: 'dirty',
    generated_at: new Date().toISOString(),
    scope: '示例数据，不代表真实审计结论',
    actual_state_baseline: 'WORKTREE',
    attribution_baseline: 'HEAD sample-commit',
    verdict: 'DRIFT_DETECTED',
    overall_risk: 'MEDIUM',
    limitations: ['示例未执行语义审计。'],
  },
  executive_summary: ['这是渲染器自检使用的安全示例，不代表项目真实状态。'],
  inventory: {
    active_documents: 1,
    draft_documents: 0,
    stale_documents: 0,
    deprecated_documents: 0,
    archived_documents: 0,
    invalid_files: 1,
  },
  coverage: REQUIRED_COVERAGE.map(([id, dimension], index) => ({
    id,
    dimension,
    status: index === 0 ? 'REVIEWED' : 'NOT_REVIEWED',
    evidence: index === 0 ? ['.knowledge/README.md:1'] : [],
    limitations: index === 0 ? [] : ['示例未执行该维度审计。'],
  })),
  documents: [{
    id: 'system-overview',
    path: '.knowledge/architecture/system-overview.md',
    kind: 'architecture',
    status: 'active',
    last_verified: '2026-07-19',
    review_after: '2026-10-14',
    verdict: 'UNCHANGED',
    summary: '示例中该文档的关键声明保持一致。',
    evidence: ['README.md:1'],
    finding_ids: [],
  }],
  checks: [{
    id: 'CHK-001',
    name: '示例结构校验',
    command: 'render_knowledge_audit.mjs --sample',
    status: 'PASS',
    result: '示例输入通过契约验证。',
    evidence: [],
  }],
  positive_observations: [{
    title: '知识协议定义了权威顺序',
    evidence: ['.knowledge/README.md:15'],
    notes: '示例正向观察。',
  }],
  findings: [{
    id: 'KNO-001',
    title: '示例知识文件结构无效',
    type: 'INVALID',
    severity: 'MEDIUM',
    confidence: 'CONFIRMED',
    status: 'OPEN',
    resolution_target: 'KNOWLEDGE',
    document_ids: [],
    locations: [{ path: '.knowledge/domains/example.md', line: 1, symbol: 'frontmatter' }],
    claim: '正式知识文件应包含合法 frontmatter。',
    actual_state: '示例文件缺少 frontmatter。',
    evidence: ['示例校验错误。'],
    impact: '影响知识校验和路由。',
    attribution: 'PRE_EXISTING',
    recommendation: '确认生命周期后修复结构。',
    verification: ['知识校验通过。'],
  }],
  coverage_gaps: [],
  residual_risks: ['示例未执行完整语义验证。'],
  remediation: {
    strategy: ['先恢复结构校验，再处理语义漂移。'],
    tasks: [{
      id: 'KREM-001',
      finding_ids: ['KNO-001'],
      priority: 'P1',
      title: '恢复示例知识文件有效性',
      objective: '让示例文件重新参与确定性校验。',
      rationale: '解决 KNO-001。',
      status: 'PENDING',
      resolution_target: 'KNOWLEDGE',
      knowledge_level: 'L1',
      owner_hint: 'engineering-platform',
      dependencies: [],
      parallel_safe: false,
      human_confirmation_required: true,
      scope: {
        allowed_paths: ['.knowledge/domains/example.md'],
        excluded_paths: ['smart-recruit-*-service/**'],
      },
      source_evidence: ['.knowledge/README.md'],
      steps: ['确认文件生命周期。', '按生命周期规则修复位置和元数据。'],
      acceptance_criteria: ['知识结构校验通过。'],
      tests: ['node .knowledge/scripts/validate-knowledge.mjs --root .'],
      rollback: '从任务差异恢复原文件并重新评估生命周期。',
      deliverables: ['有效知识文件', '验证证据'],
    }],
  },
})

const requireObject = (value, name, errors) => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) errors.push(`${name} must be an object`)
}

const requireArray = (value, name, errors) => {
  if (!Array.isArray(value)) errors.push(`${name} must be an array`)
}

const requireText = (value, name, errors) => {
  if (typeof value !== 'string' || value.trim() === '') errors.push(`${name} must be a non-empty string`)
}

const requireRelativePath = (value, name, errors) => {
  requireText(value, name, errors)
  if (typeof value !== 'string') return
  if (path.isAbsolute(value) || value.includes('\\') || value === '..' || value.startsWith('../')) {
    errors.push(`${name} must be a repository-relative POSIX path`)
  }
}

const validate = (data) => {
  const errors = []
  if (data?.schema_version !== 1) errors.push('schema_version must be 1')
  requireObject(data?.audit, 'audit', errors)
  requireObject(data?.inventory, 'inventory', errors)
  requireObject(data?.remediation, 'remediation', errors)
  for (const field of ['executive_summary', 'coverage', 'documents', 'checks', 'positive_observations', 'findings', 'coverage_gaps', 'residual_risks']) {
    requireArray(data?.[field], field, errors)
  }
  requireArray(data?.remediation?.strategy, 'remediation.strategy', errors)
  requireArray(data?.remediation?.tasks, 'remediation.tasks', errors)

  const audit = data?.audit || {}
  for (const field of ['title', 'repo', 'branch', 'commit', 'working_tree', 'generated_at', 'scope', 'actual_state_baseline', 'attribution_baseline', 'verdict', 'overall_risk']) {
    requireText(audit[field], `audit.${field}`, errors)
  }
  requireArray(audit.limitations, 'audit.limitations', errors)
  if (!AUDIT_VERDICTS.includes(audit.verdict)) errors.push(`audit.verdict must be one of ${AUDIT_VERDICTS.join(', ')}`)
  if (!RISKS.includes(audit.overall_risk)) errors.push(`audit.overall_risk must be one of ${RISKS.join(', ')}`)

  for (const field of ['active_documents', 'draft_documents', 'stale_documents', 'deprecated_documents', 'archived_documents', 'invalid_files']) {
    if (!Number.isInteger(data?.inventory?.[field]) || data.inventory[field] < 0) errors.push(`inventory.${field} must be a non-negative integer`)
  }

  const coverageIds = new Set()
  values(data?.coverage).forEach((item, index) => {
    requireText(item?.id, `coverage[${index}].id`, errors)
    requireText(item?.dimension, `coverage[${index}].dimension`, errors)
    requireArray(item?.evidence, `coverage[${index}].evidence`, errors)
    requireArray(item?.limitations, `coverage[${index}].limitations`, errors)
    if (!COVERAGE_STATUSES.includes(item?.status)) errors.push(`coverage[${index}].status is invalid`)
    if (coverageIds.has(item?.id)) errors.push(`duplicate coverage id: ${item?.id}`)
    coverageIds.add(item?.id)
  })
  for (const [id] of REQUIRED_COVERAGE) if (!coverageIds.has(id)) errors.push(`missing mandatory coverage dimension: ${id}`)

  const documentIds = new Set()
  values(data?.documents).forEach((item, index) => {
    const prefix = `documents[${index}]`
    for (const field of ['id', 'kind', 'status', 'last_verified', 'review_after', 'summary']) requireText(item?.[field], `${prefix}.${field}`, errors)
    requireRelativePath(item?.path, `${prefix}.path`, errors)
    requireArray(item?.evidence, `${prefix}.evidence`, errors)
    requireArray(item?.finding_ids, `${prefix}.finding_ids`, errors)
    if (item?.status !== 'active') errors.push(`${prefix}.status must be active`)
    if (!DOCUMENT_VERDICTS.includes(item?.verdict)) errors.push(`${prefix}.verdict is invalid`)
    if (documentIds.has(item?.id)) errors.push(`duplicate document id: ${item?.id}`)
    documentIds.add(item?.id)
  })
  if (Number.isInteger(data?.inventory?.active_documents) && documentIds.size !== data.inventory.active_documents) {
    errors.push(`documents count ${documentIds.size} does not match inventory.active_documents ${data.inventory.active_documents}`)
  }

  const checkIds = new Set()
  values(data?.checks).forEach((item, index) => {
    const prefix = `checks[${index}]`
    for (const field of ['id', 'name', 'command', 'result']) requireText(item?.[field], `${prefix}.${field}`, errors)
    requireArray(item?.evidence, `${prefix}.evidence`, errors)
    if (!/^CHK-\d{3,}$/.test(item?.id || '')) errors.push(`${prefix}.id must match CHK-NNN`)
    if (!CHECK_STATUSES.includes(item?.status)) errors.push(`${prefix}.status is invalid`)
    if (checkIds.has(item?.id)) errors.push(`duplicate check id: ${item?.id}`)
    checkIds.add(item?.id)
  })

  const findingIds = new Set()
  const findingById = new Map()
  values(data?.findings).forEach((item, index) => {
    const prefix = `findings[${index}]`
    for (const field of ['id', 'title', 'claim', 'actual_state', 'impact', 'recommendation']) requireText(item?.[field], `${prefix}.${field}`, errors)
    for (const field of ['document_ids', 'locations', 'evidence', 'verification']) requireArray(item?.[field], `${prefix}.${field}`, errors)
    if (!/^KNO-\d{3,}$/.test(item?.id || '')) errors.push(`${prefix}.id must match KNO-NNN`)
    if (!FINDING_TYPES.includes(item?.type)) errors.push(`${prefix}.type is invalid`)
    if (!RISKS.includes(item?.severity)) errors.push(`${prefix}.severity is invalid`)
    if (!CONFIDENCES.includes(item?.confidence)) errors.push(`${prefix}.confidence is invalid`)
    if (!FINDING_STATUSES.includes(item?.status)) errors.push(`${prefix}.status is invalid`)
    if (!RESOLUTION_TARGETS.includes(item?.resolution_target)) errors.push(`${prefix}.resolution_target is invalid`)
    if (!ATTRIBUTIONS.includes(item?.attribution)) errors.push(`${prefix}.attribution is invalid`)
    if (findingIds.has(item?.id)) errors.push(`duplicate finding id: ${item?.id}`)
    findingIds.add(item?.id)
    findingById.set(item?.id, item)
    values(item?.document_ids).forEach((id) => {
      if (!documentIds.has(id)) errors.push(`${prefix} references unknown active document ${id}`)
    })
    values(item?.locations).forEach((location, locationIndex) => {
      requireRelativePath(location?.path, `${prefix}.locations[${locationIndex}].path`, errors)
      if (location?.line !== undefined && (!Number.isInteger(location.line) || location.line < 1)) errors.push(`${prefix}.locations[${locationIndex}].line must be a positive integer`)
    })
  })

  values(data?.documents).forEach((document, index) => {
    values(document?.finding_ids).forEach((findingId) => {
      const finding = findingById.get(findingId)
      if (!finding) errors.push(`documents[${index}] references unknown finding ${findingId}`)
      else if (!values(finding.document_ids).includes(document.id)) errors.push(`finding ${findingId} does not reference document ${document.id} reciprocally`)
    })
    if (document?.verdict !== 'UNCHANGED' && !values(document?.finding_ids).length) {
      errors.push(`document ${document?.id} with verdict ${document?.verdict} must reference at least one finding`)
    }
  })
  for (const finding of values(data?.findings)) {
    for (const documentId of values(finding.document_ids)) {
      const document = values(data?.documents).find((item) => item.id === documentId)
      if (document && !values(document.finding_ids).includes(finding.id)) errors.push(`document ${documentId} does not reference finding ${finding.id} reciprocally`)
    }
  }

  const taskIds = new Set()
  const taskById = new Map()
  const mappedFindings = new Set()
  values(data?.remediation?.tasks).forEach((task, index) => {
    const prefix = `remediation.tasks[${index}]`
    for (const field of ['id', 'title', 'objective', 'rationale', 'status', 'owner_hint', 'rollback']) requireText(task?.[field], `${prefix}.${field}`, errors)
    for (const field of ['finding_ids', 'dependencies', 'source_evidence', 'steps', 'acceptance_criteria', 'tests', 'deliverables']) requireArray(task?.[field], `${prefix}.${field}`, errors)
    requireObject(task?.scope, `${prefix}.scope`, errors)
    requireArray(task?.scope?.allowed_paths, `${prefix}.scope.allowed_paths`, errors)
    requireArray(task?.scope?.excluded_paths, `${prefix}.scope.excluded_paths`, errors)
    values(task?.scope?.allowed_paths).forEach((item, pathIndex) => requireRelativePath(item, `${prefix}.scope.allowed_paths[${pathIndex}]`, errors))
    values(task?.scope?.excluded_paths).forEach((item, pathIndex) => requireRelativePath(item, `${prefix}.scope.excluded_paths[${pathIndex}]`, errors))
    if (!/^KREM-\d{3,}$/.test(task?.id || '')) errors.push(`${prefix}.id must match KREM-NNN`)
    if (!PRIORITIES.includes(task?.priority)) errors.push(`${prefix}.priority is invalid`)
    if (!KNOWLEDGE_LEVELS.includes(task?.knowledge_level)) errors.push(`${prefix}.knowledge_level is invalid`)
    if (!RESOLUTION_TARGETS.includes(task?.resolution_target)) errors.push(`${prefix}.resolution_target is invalid`)
    if (typeof task?.parallel_safe !== 'boolean') errors.push(`${prefix}.parallel_safe must be boolean`)
    if (typeof task?.human_confirmation_required !== 'boolean') errors.push(`${prefix}.human_confirmation_required must be boolean`)
    if (task?.knowledge_level === 'L3' && task?.human_confirmation_required !== true) errors.push(`${prefix}: L3 task requires human confirmation`)
    if (taskIds.has(task?.id)) errors.push(`duplicate task id: ${task?.id}`)
    taskIds.add(task?.id)
    taskById.set(task?.id, task)
    values(task?.finding_ids).forEach((findingId) => {
      const finding = findingById.get(findingId)
      if (!finding) errors.push(`${prefix} references unknown finding ${findingId}`)
      else if (finding.resolution_target !== task.resolution_target && finding.resolution_target !== 'DECISION_REQUIRED') errors.push(`${prefix} resolution_target conflicts with finding ${findingId}`)
      mappedFindings.add(findingId)
    })
  })
  for (const task of values(data?.remediation?.tasks)) {
    for (const dependency of values(task.dependencies)) {
      if (!taskById.has(dependency)) errors.push(`task ${task.id} references unknown dependency ${dependency}`)
      if (dependency === task.id) errors.push(`task ${task.id} cannot depend on itself`)
    }
  }
  const visiting = new Set()
  const visited = new Set()
  const visitTask = (taskId) => {
    if (visiting.has(taskId)) { errors.push(`task dependency cycle detected at ${taskId}`); return }
    if (visited.has(taskId) || !taskById.has(taskId)) return
    visiting.add(taskId)
    for (const dependency of values(taskById.get(taskId).dependencies)) visitTask(dependency)
    visiting.delete(taskId)
    visited.add(taskId)
  }
  for (const taskId of taskIds) visitTask(taskId)
  for (const finding of values(data?.findings)) {
    if (finding.status === 'OPEN' && !mappedFindings.has(finding.id)) errors.push(`open finding ${finding.id} has no remediation task`)
  }

  const openFindings = values(data?.findings).filter((item) => item.status === 'OPEN')
  if (audit.verdict === 'CURRENT' && openFindings.length) errors.push('CURRENT cannot contain open findings')
  if (audit.verdict === 'CURRENT' && values(data?.coverage).some((item) => !['REVIEWED', 'NOT_APPLICABLE'].includes(item.status))) errors.push('CURRENT requires complete mandatory coverage')
  if (audit.verdict === 'CURRENT' && values(data?.checks).some((item) => ['FAIL', 'BLOCKED', 'NOT_RUN'].includes(item.status))) errors.push('CURRENT cannot contain failed, blocked, or unrun checks')
  if (audit.verdict === 'UNRELIABLE' && !openFindings.some((item) => ['CRITICAL', 'HIGH'].includes(item.severity)) && !values(data?.coverage).some((item) => item.status === 'NOT_REVIEWED')) {
    errors.push('UNRELIABLE requires a high-risk open finding or material unreviewed coverage')
  }
  return errors
}

const auditVerdictZh = {
  UNRELIABLE: '不可信',
  DRIFT_DETECTED: '发现知识漂移',
  CURRENT_WITH_GAPS: '基本现行但存在缺口',
  CURRENT: '当前一致',
}

const countBy = (items, field) => values(items).reduce((result, item) => {
  const key = item?.[field] || 'UNKNOWN'
  result[key] = (result[key] || 0) + 1
  return result
}, {})

const formatCounts = (counts) => Object.entries(counts).map(([key, count]) => `${key}: ${count}`).join('；') || '无'

const listSectionMd = (title, items) => [
  `## ${title}`,
  '',
  ...(values(items).length ? values(items).map((item) => `- ${item}`) : ['- 无']),
  '',
].join('\n')

const renderAuditMarkdown = (data) => {
  const a = data.audit
  const lines = [
    `# ${a.title}`,
    '',
    '> 本报告由同一规范化审计数据生成；对应 HTML 与修复计划不得独立修改。',
    '',
    '## 审计元数据',
    '',
    '| 项目 | 值 |',
    '|---|---|',
    `| 仓库 | ${md(a.repo)} |`,
    `| 分支 | ${md(a.branch)} |`,
    `| Commit | ${md(a.commit)} |`,
    `| 工作树 | ${md(a.working_tree)} |`,
    `| 实际状态基线 | ${md(a.actual_state_baseline)} |`,
    `| 归因基线 | ${md(a.attribution_baseline)} |`,
    `| 范围 | ${md(a.scope)} |`,
    `| 生成时间 | ${md(a.generated_at)} |`,
    `| 审计结论 | **${auditVerdictZh[a.verdict]} (${a.verdict})** |`,
    `| 整体风险 | **${md(a.overall_risk)}** |`,
    '',
    '## 执行摘要',
    '',
    ...(values(data.executive_summary).length ? values(data.executive_summary).map((item) => `- ${item}`) : ['- 无']),
    '',
    '## 知识资产清单',
    '',
    '| Active | Draft | Stale | Deprecated | Archived | Invalid files |',
    '|---:|---:|---:|---:|---:|---:|',
    `| ${data.inventory.active_documents} | ${data.inventory.draft_documents} | ${data.inventory.stale_documents} | ${data.inventory.deprecated_documents} | ${data.inventory.archived_documents} | ${data.inventory.invalid_files} |`,
    '',
    '## 结论统计',
    '',
    '| 统计项 | 结果 |',
    '|---|---|',
    `| 文档结论 | ${md(formatCounts(countBy(data.documents, 'verdict')))} |`,
    `| Finding 类型 | ${md(formatCounts(countBy(data.findings, 'type')))} |`,
    `| Finding 严重度 | ${md(formatCounts(countBy(data.findings, 'severity')))} |`,
    `| Open Findings | ${data.findings.filter((item) => item.status === 'OPEN').length} |`,
    '',
    '## 覆盖维度',
    '',
    '| ID | 维度 | 状态 | 证据 | 限制 |',
    '|---|---|---|---|---|',
    ...values(data.coverage).map((item) => `| ${md(item.id)} | ${md(item.dimension)} | ${md(item.status)} | ${md(joinText(item.evidence))} | ${md(joinText(item.limitations))} |`),
    '',
    '## Active 文档逐项结论',
    '',
    '| ID | 路径 | 类型 | 上次验证 | 复核日 | 结论 | 摘要 | Findings |',
    '|---|---|---|---|---|---|---|---|',
    ...values(data.documents).map((item) => `| ${md(item.id)} | ${md(item.path)} | ${md(item.kind)} | ${md(item.last_verified)} | ${md(item.review_after)} | **${md(item.verdict)}** | ${md(item.summary)} | ${md(joinText(item.finding_ids))} |`),
    '',
    '## 验证命令',
    '',
    '| ID | 检查 | 状态 | 命令 | 结果 |',
    '|---|---|---|---|---|',
    ...values(data.checks).map((item) => `| ${md(item.id)} | ${md(item.name)} | **${md(item.status)}** | ${md(item.command)} | ${md(item.result)} |`),
    '',
    '## 审计发现',
    '',
  ]
  if (!data.findings.length) lines.push('无正式发现。', '')
  for (const item of data.findings) {
    lines.push(
      `### ${item.id} — ${item.title}`,
      '',
      `- 类型/严重度/置信度：${code(item.type)} / ${code(item.severity)} / ${code(item.confidence)}`,
      `- 状态/修复目标/归因：${code(item.status)} / ${code(item.resolution_target)} / ${code(item.attribution)}`,
      `- 关联文档：${joinMd(item.document_ids)}`,
      `- 位置：${values(item.locations).length ? item.locations.map((location) => code(`${location.path}${location.line ? `:${location.line}` : ''}${location.symbol ? `#${location.symbol}` : ''}`)).join('、') : '无'}`,
      `- 知识声明：${item.claim}`,
      `- 实际状态：${item.actual_state}`,
      `- 影响：${item.impact}`,
      `- 证据：${joinMd(item.evidence)}`,
      `- 建议：${item.recommendation}`,
      `- 验证：${joinMd(item.verification)}`,
      '',
    )
  }
  lines.push(
    listSectionMd('覆盖缺口', data.coverage_gaps).trimEnd(),
    '',
    '## 已验证的正向事实',
    '',
  )
  if (!data.positive_observations.length) lines.push('- 无', '')
  for (const item of data.positive_observations) lines.push(`- **${item.title}**：${item.notes}（证据：${joinMd(item.evidence)}）`)
  lines.push('', listSectionMd('残余风险', data.residual_risks).trimEnd(), '', listSectionMd('审计限制', a.limitations).trimEnd(), '')
  return `${lines.join('\n').trim()}\n`
}

const htmlList = (items) => values(items).length
  ? `<ul>${values(items).map((item) => `<li>${esc(item)}</li>`).join('')}</ul>`
  : '<p class="muted">无</p>'

const badge = (value) => `<span class="badge badge-${esc(String(value).toLowerCase().replaceAll('_', '-'))}">${esc(value)}</span>`

const renderAuditHtml = (data) => {
  const a = data.audit
  const coverageRows = data.coverage.map((item) => `<tr><td><code>${esc(item.id)}</code></td><td>${esc(item.dimension)}</td><td>${badge(item.status)}</td><td>${esc(joinText(item.evidence))}</td><td>${esc(joinText(item.limitations))}</td></tr>`).join('')
  const documentRows = data.documents.map((item) => `<tr><td><code>${esc(item.id)}</code></td><td><code>${esc(item.path)}</code></td><td>${esc(item.kind)}</td><td>${esc(item.last_verified)}</td><td>${esc(item.review_after)}</td><td>${badge(item.verdict)}</td><td>${esc(item.summary)}</td><td>${esc(joinText(item.evidence))}</td><td>${esc(joinText(item.finding_ids))}</td></tr>`).join('')
  const checkRows = data.checks.map((item) => `<tr><td><code>${esc(item.id)}</code></td><td>${esc(item.name)}</td><td>${badge(item.status)}</td><td><code>${esc(item.command)}</code></td><td>${esc(item.result)}</td><td>${esc(joinText(item.evidence))}</td></tr>`).join('')
  const findingCards = data.findings.length ? data.findings.map((item) => `<article class="finding">
    <div class="finding-head"><h3>${esc(item.id)} — ${esc(item.title)}</h3><div>${badge(item.type)} ${badge(item.severity)} ${badge(item.status)}</div></div>
    <dl><dt>置信度</dt><dd>${esc(item.confidence)}</dd><dt>修复目标</dt><dd>${esc(item.resolution_target)}</dd><dt>归因</dt><dd>${esc(item.attribution)}</dd><dt>关联文档</dt><dd>${esc(joinText(item.document_ids))}</dd><dt>位置</dt><dd>${esc(item.locations.map((location) => `${location.path}${location.line ? `:${location.line}` : ''}${location.symbol ? `#${location.symbol}` : ''}`).join('；') || '无')}</dd></dl>
    <h4>知识声明</h4><p>${esc(item.claim)}</p><h4>实际状态</h4><p>${esc(item.actual_state)}</p><h4>影响</h4><p>${esc(item.impact)}</p><h4>证据</h4>${htmlList(item.evidence)}<h4>建议</h4><p>${esc(item.recommendation)}</p><h4>验证</h4>${htmlList(item.verification)}
  </article>`).join('') : '<p class="muted">无正式发现。</p>'
  const positives = data.positive_observations.length ? data.positive_observations.map((item) => `<article class="positive"><h3>${esc(item.title)}</h3><p>${esc(item.notes)}</p>${htmlList(item.evidence)}</article>`).join('') : '<p class="muted">无</p>'
  return `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${esc(a.title)}</title>
<style>
:root{color-scheme:light;--bg:#f4f7fb;--panel:#fff;--ink:#172033;--muted:#637083;--line:#dce3ee;--brand:#2457d6;--danger:#b42318;--warn:#b54708;--ok:#067647;--shadow:0 12px 30px rgba(22,34,55,.08)}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}main{max-width:1440px;margin:auto;padding:32px 24px 64px}.hero{padding:28px;border-radius:18px;background:linear-gradient(135deg,#132c67,#2d64df);color:#fff;box-shadow:var(--shadow)}h1{font-size:30px;line-height:1.3;margin:0 0 12px}h2{font-size:21px;margin:34px 0 14px}h3{font-size:16px;margin:0}h4{font-size:13px;margin:16px 0 4px}.meta,.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:12px;margin-top:18px}.tile,.panel,.finding,.positive{background:var(--panel);border:1px solid var(--line);border-radius:14px;box-shadow:var(--shadow)}.tile{padding:14px;color:var(--ink)}.tile small{display:block;color:var(--muted);margin-bottom:4px}.tile strong{font-size:17px}.panel{padding:18px;overflow:auto}.stats .tile strong{font-size:24px;color:var(--brand)}table{width:100%;border-collapse:collapse;min-width:760px}th,td{padding:10px 12px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}th{background:#f8faff;color:#4b5870;position:sticky;top:0}code{font:12px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;background:#f1f4f9;padding:2px 5px;border-radius:5px;overflow-wrap:anywhere}.finding,.positive{padding:18px;margin:12px 0}.finding{border-left:5px solid var(--warn)}.positive{border-left:5px solid var(--ok)}.finding-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.badge{display:inline-block;padding:3px 8px;border-radius:999px;background:#e9eef7;color:#344054;font-size:11px;font-weight:700;white-space:nowrap}.badge-critical,.badge-high,.badge-unreliable,.badge-conflict,.badge-fail,.badge-invalid{background:#fee4e2;color:var(--danger)}.badge-medium,.badge-drift-detected,.badge-stale,.badge-partial,.badge-blocked{background:#fef0c7;color:var(--warn)}.badge-low,.badge-current,.badge-unchanged,.badge-reviewed,.badge-pass{background:#dcfae6;color:var(--ok)}.badge-not-reviewed,.badge-not-run,.badge-evidence-gap{background:#eceff3;color:#475467}dl{display:grid;grid-template-columns:110px 1fr;gap:5px 14px}dt{color:var(--muted)}dd{margin:0}.muted{color:var(--muted)}ul{padding-left:20px}@media(max-width:720px){main{padding:16px 12px}.hero{padding:20px}.finding-head{display:block}dl{grid-template-columns:1fr}table{font-size:12px}}
</style></head><body><main>
<header class="hero"><h1>${esc(a.title)}</h1><p>${esc(data.executive_summary.join('；'))}</p><div class="meta"><div class="tile"><small>审计结论</small><strong>${esc(auditVerdictZh[a.verdict])}</strong><br>${badge(a.verdict)}</div><div class="tile"><small>整体风险</small><strong>${esc(a.overall_risk)}</strong></div><div class="tile"><small>Commit / 工作树</small><strong>${esc(a.commit)}</strong><br>${esc(a.working_tree)}</div><div class="tile"><small>生成时间</small><strong>${esc(a.generated_at)}</strong></div></div></header>
<section><h2>审计边界</h2><div class="panel"><dl><dt>仓库/分支</dt><dd>${esc(a.repo)} / ${esc(a.branch)}</dd><dt>范围</dt><dd>${esc(a.scope)}</dd><dt>实际状态基线</dt><dd>${esc(a.actual_state_baseline)}</dd><dt>归因基线</dt><dd>${esc(a.attribution_baseline)}</dd></dl></div></section>
<section><h2>知识资产清单</h2><div class="stats"><div class="tile"><small>Active</small><strong>${data.inventory.active_documents}</strong></div><div class="tile"><small>Draft</small><strong>${data.inventory.draft_documents}</strong></div><div class="tile"><small>Stale</small><strong>${data.inventory.stale_documents}</strong></div><div class="tile"><small>Deprecated / Archived</small><strong>${data.inventory.deprecated_documents} / ${data.inventory.archived_documents}</strong></div><div class="tile"><small>Invalid files</small><strong>${data.inventory.invalid_files}</strong></div></div></section>
<section><h2>结论统计</h2><div class="panel"><dl><dt>文档结论</dt><dd>${esc(formatCounts(countBy(data.documents, 'verdict')))}</dd><dt>Finding 类型</dt><dd>${esc(formatCounts(countBy(data.findings, 'type')))}</dd><dt>Finding 严重度</dt><dd>${esc(formatCounts(countBy(data.findings, 'severity')))}</dd><dt>Open Findings</dt><dd>${data.findings.filter((item) => item.status === 'OPEN').length}</dd></dl></div></section>
<section><h2>覆盖维度</h2><div class="panel"><table><thead><tr><th>ID</th><th>维度</th><th>状态</th><th>证据</th><th>限制</th></tr></thead><tbody>${coverageRows}</tbody></table></div></section>
<section><h2>Active 文档逐项结论</h2><div class="panel"><table><thead><tr><th>ID</th><th>路径</th><th>类型</th><th>上次验证</th><th>复核日</th><th>结论</th><th>摘要</th><th>证据</th><th>Findings</th></tr></thead><tbody>${documentRows}</tbody></table></div></section>
<section><h2>验证命令</h2><div class="panel"><table><thead><tr><th>ID</th><th>检查</th><th>状态</th><th>命令</th><th>结果</th><th>证据</th></tr></thead><tbody>${checkRows}</tbody></table></div></section>
<section><h2>审计发现</h2>${findingCards}</section>
<section><h2>覆盖缺口</h2><div class="panel">${htmlList(data.coverage_gaps)}</div></section>
<section><h2>已验证的正向事实</h2>${positives}</section>
<section><h2>残余风险</h2><div class="panel">${htmlList(data.residual_risks)}</div></section>
<section><h2>审计限制</h2><div class="panel">${htmlList(a.limitations)}</div></section>
</main></body></html>\n`
}

const orderTasks = (input) => {
  const byId = new Map(values(input).map((task) => [task.id, task]))
  const ordered = []
  const visited = new Set()
  const visit = (task) => {
    if (visited.has(task.id)) return
    for (const dependency of values(task.dependencies)) visit(byId.get(dependency))
    visited.add(task.id)
    ordered.push(task)
  }
  const seeds = [...values(input)].sort((left, right) => PRIORITIES.indexOf(left.priority) - PRIORITIES.indexOf(right.priority) || left.id.localeCompare(right.id))
  for (const task of seeds) visit(task)
  return ordered
}

const renderRemediationMarkdown = (data) => {
  const a = data.audit
  const tasks = orderTasks(data.remediation.tasks)
  const manifest = {
    schema_version: 1,
    audit: { commit: a.commit, working_tree: a.working_tree, verdict: a.verdict, generated_at: a.generated_at },
    findings: data.findings.map((finding) => ({
      id: finding.id,
      title: finding.title,
      type: finding.type,
      severity: finding.severity,
      status: finding.status,
      resolution_target: finding.resolution_target,
    })),
    tasks,
  }
  const lines = [
    '# Smart Recruit 项目知识库修复计划',
    '',
    '> 本计划由知识审计规范化数据生成。它不自动授权修改源代码、生产系统、外部系统或任务范围外文件。',
    '',
    '## 审计关联',
    '',
    '| 项目 | 值 |',
    '|---|---|',
    `| 审计结论 | ${md(a.verdict)} |`,
    `| 整体风险 | ${md(a.overall_risk)} |`,
    `| Commit | ${md(a.commit)} |`,
    `| 工作树 | ${md(a.working_tree)} |`,
    `| 范围 | ${md(a.scope)} |`,
    `| 生成时间 | ${md(a.generated_at)} |`,
    '',
    '## 执行规则与非目标',
    '',
    '- 按依赖顺序，再按 P0 → P1 → P2 → P3 串行推进；仅在任务明确标记 `parallel_safe: true` 时并行。',
    '- 每个任务开始前重新读取 `AGENTS.md`、`.knowledge/README.md` 和任务证据；保留用户已有修改。',
    '- 不以“让文档和代码一致”为由擅自选择权威侧；遵守 `resolution_target`。',
    '- L2 需要代码/测试/契约证据和复核；L3 需要人工确认并优先形成 Inbox candidate 或 proposed ADR。',
    '- 仅在完成实质验证后更新 `last_verified`。',
    '- 本计划不要求也不自动启动 `spec-harness`。',
    '',
    '## 修复策略',
    '',
    ...(data.remediation.strategy.length ? data.remediation.strategy.map((item) => `- ${item}`) : ['- 无']),
    '',
    '## 任务顺序',
    '',
    '| 任务 | 优先级 | Findings | 目标 | 级别 | 依赖 | 人工确认 | 可并行 |',
    '|---|---|---|---|---|---|---|---|',
    ...tasks.map((task) => `| ${md(task.id)} ${md(task.title)} | ${md(task.priority)} | ${md(joinText(task.finding_ids))} | ${md(task.resolution_target)} | ${md(task.knowledge_level)} | ${md(joinText(task.dependencies))} | ${task.human_confirmation_required ? '是' : '否'} | ${task.parallel_safe ? '是' : '否'} |`),
    '',
  ]
  if (!tasks.length) lines.push('没有待执行修复任务。', '')
  for (const task of tasks) {
    lines.push(
      `## ${task.id} — ${task.title}`,
      '',
      `- 优先级/状态：${code(task.priority)} / ${code(task.status)}`,
      `- 修复目标/知识级别：${code(task.resolution_target)} / ${code(task.knowledge_level)}`,
      `- Findings：${joinMd(task.finding_ids)}`,
      `- Owner 建议：${task.owner_hint}`,
      `- 依赖：${joinMd(task.dependencies)}`,
      `- 可并行：${task.parallel_safe ? '是' : '否'}`,
      `- 需要人工确认：${task.human_confirmation_required ? '是' : '否'}`,
      '',
      '### 目标', '', task.objective, '',
      '### 理由', '', task.rationale, '',
      '### 允许范围', '', ...values(task.scope.allowed_paths).map((item) => `- ${code(item)}`), '',
      '### 排除范围', '', ...(values(task.scope.excluded_paths).length ? values(task.scope.excluded_paths).map((item) => `- ${code(item)}`) : ['- 无']), '',
      '### 源证据', '', ...(values(task.source_evidence).length ? values(task.source_evidence).map((item) => `- ${code(item)}`) : ['- 无']), '',
      '### 执行步骤', '', ...values(task.steps).map((item, index) => `${index + 1}. ${item}`), '',
      '### 验收标准', '', ...values(task.acceptance_criteria).map((item) => `- [ ] ${item}`), '',
      '### 验证命令', '', ...values(task.tests).map((item) => `- ${code(item)}`), '',
      '### 回滚/撤销', '', task.rollback, '',
      '### 交付物', '', ...values(task.deliverables).map((item) => `- ${item}`), '',
    )
  }
  const traceability = data.findings.map((finding) => ({ finding, tasks: tasks.filter((task) => task.finding_ids.includes(finding.id)) }))
  lines.push(
    '## Finding—任务追踪矩阵', '',
    '| Finding | 状态 | 严重度 | 修复目标 | 任务 |',
    '|---|---|---|---|---|',
    ...traceability.map(({ finding, tasks: linked }) => `| ${md(finding.id)} ${md(finding.title)} | ${md(finding.status)} | ${md(finding.severity)} | ${md(finding.resolution_target)} | ${md(linked.map((task) => task.id).join('、') || '无')} |`),
    '',
    '## 最终累计验证门禁', '',
    '- [ ] 所有 P0/P1 任务已完成，或由有权限的 Owner 明确接受。',
    '- [ ] 所有修改后的 active 文档完成实质复核，并具有可辩护的结论。',
    '- [ ] 知识校验器测试、仓库校验和严格引用检查全部通过。',
    '- [ ] Manifest 路由、INDEX 和正式知识目录保持一致。',
    '- [ ] 所有 SOURCE/DECISION_REQUIRED 冲突明确记录权威选择和审批。',
    '- [ ] 基于最新证据重新生成三份审计产物并重新评估结论。',
    '',
    '## Agent 执行清单', '',
    '```knowledge-remediation-manifest',
    JSON.stringify(manifest, null, 2),
    '```',
    '',
  )
  return `${lines.join('\n').trim()}\n`
}

const timestamp = () => new Date().toISOString().replace(/[-:]/g, '').replace('T', '-').slice(0, 15)

const main = () => {
  let args
  try { args = parseArgs(process.argv) } catch (error) { console.error(`error: ${error.message}`); console.error(usage()); process.exit(2) }
  if (args.help) { console.log(usage()); return }
  if (!args.sample && !args.input) { console.error('error: --input or --sample is required'); console.error(usage()); process.exit(2) }
  if (args.sample && args.input) { console.error('error: --sample and --input are mutually exclusive'); process.exit(2) }

  let data
  try { data = args.sample ? samplePayload() : JSON.parse(fs.readFileSync(args.input, 'utf8')) }
  catch (error) { console.error(`error: unable to read audit JSON: ${error.message}`); process.exit(1) }

  const errors = validate(data)
  if (errors.length) {
    for (const error of errors) console.error(`error: ${error}`)
    process.exit(1)
  }

  const basename = args.basename || `knowledge-current-state-audit-${timestamp()}`
  if (!/^[A-Za-z0-9._-]+$/.test(basename) || basename === '.' || basename === '..') {
    console.error('error: --basename may contain only letters, digits, dot, underscore, and hyphen')
    process.exit(2)
  }
  const outputDir = path.resolve(args.outputDir)
  const outputs = {
    markdown: path.join(outputDir, `${basename}-knowledge-audit-report.md`),
    html: path.join(outputDir, `${basename}-knowledge-audit-report.html`),
    remediation: path.join(outputDir, `${basename}-knowledge-remediation-plan.md`),
  }
  for (const output of Object.values(outputs)) {
    if (fs.existsSync(output)) { console.error(`error: refusing to overwrite existing file: ${output}`); process.exit(1) }
  }
  fs.mkdirSync(outputDir, { recursive: true })
  fs.writeFileSync(outputs.markdown, renderAuditMarkdown(data), 'utf8')
  fs.writeFileSync(outputs.html, renderAuditHtml(data), 'utf8')
  fs.writeFileSync(outputs.remediation, renderRemediationMarkdown(data), 'utf8')
  console.log('knowledge_audit_render: PASS')
  console.log(`markdown_report: ${outputs.markdown}`)
  console.log(`html_report: ${outputs.html}`)
  console.log(`remediation_plan: ${outputs.remediation}`)
}

main()
