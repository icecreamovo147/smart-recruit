#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'

const SEVERITIES = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO']
const CONFIDENCES = ['CONFIRMED', 'HIGH', 'MEDIUM', 'LOW']
const VERDICTS = ['NO_GO', 'CONDITIONAL_GO', 'GO']
const COVERAGE_STATUSES = ['REVIEWED', 'PARTIAL', 'NOT_REVIEWED', 'NOT_APPLICABLE']
const CHECK_STATUSES = ['PASS', 'FAIL', 'NOT_RUN', 'BLOCKED', 'NOT_APPLICABLE']
const FINDING_STATUSES = ['OPEN', 'MITIGATED', 'ACCEPTED', 'FALSE_POSITIVE']
const PRIORITIES = ['P0', 'P1', 'P2', 'P3']
const REQUIRED_COVERAGE = [
  ['architecture-threat-model', '架构与威胁模型'],
  ['authentication-session', '认证与会话'],
  ['authorization-tenancy', '授权、租户隔离与业务逻辑'],
  ['gateway-api-browser', 'Gateway、API、浏览器与前端'],
  ['grpc-service-mq', '内部 gRPC、服务身份与 MQ'],
  ['resume-file-oss', '简历、文件解析与 OSS'],
  ['ai-llm-mcp', 'AI、LLM、Tool 与 MCP'],
  ['billing-integrations', '账单、支付与外部集成'],
  ['data-privacy-recovery', '数据、隐私、日志与恢复'],
  ['secrets-config-crypto', '密钥、配置与密码学'],
  ['supply-chain-cicd', '依赖、CI/CD 与供应链'],
  ['runtime-kubernetes-network', '容器、Kubernetes 与网络运行时'],
  ['availability-detection-ir', '可用性、滥用检测与应急响应'],
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

const list = (values) => Array.isArray(values) ? values : []
const text = (value, fallback = '未提供') => String(value ?? '').trim() || fallback
const isOpen = (finding) => finding.status === 'OPEN'
const isBlocking = (finding) => Boolean(finding.release_blocker) && isOpen(finding)

const parseArgs = (argv) => {
  const args = { outputDir: '.security-review' }
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
  node render_security_audit.mjs --input <audit.json> [--output-dir <dir>] [--basename <name>]
  node render_security_audit.mjs --sample [--output-dir <dir>] [--basename <name>]
`

const samplePayload = () => ({
  schema_version: 1,
  audit: {
    title: 'Smart Recruit 上线前安全审查报告',
    repo: 'smart-recruit',
    branch: 'main',
    commit: 'sample-commit',
    working_tree: 'clean',
    generated_at: new Date().toISOString(),
    scope: '示例数据，不代表真实审查结论',
    authorized_test_level: 'L0',
    standards: ['OWASP ASVS 5.0 L2'],
    limitations: ['示例未连接任何运行环境'],
    verdict: 'NO_GO',
    overall_risk: 'HIGH',
  },
  executive_summary: ['这是渲染器自检使用的安全示例。'],
  attack_surface: [{
    surface: 'Gateway',
    assets: ['会话与业务 API'],
    entry_points: ['/api/v1'],
    trust_boundaries: ['Browser -> Gateway'],
    notes: '示例攻击面',
  }],
  coverage: REQUIRED_COVERAGE.map(([id, domain], index) => ({
    id,
    domain,
    status: index === 3 ? 'REVIEWED' : 'NOT_REVIEWED',
    evidence: index === 3 ? ['smart-recruit-gateway/router/router.go:106'] : [],
    limitations: index === 3 ? ['仅进行静态验证'] : ['示例未执行该领域审查'],
  })),
  checks: [{
    id: 'CHK-001',
    domain: '结构校验',
    command: 'sample renderer validation',
    status: 'PASS',
    result: '示例输入通过契约验证',
    evidence: [],
  }],
  positive_controls: [{
    title: '固定 JWT 签名算法',
    evidence: ['smart-recruit-gateway/middleware/jwt.go:48'],
    notes: '示例正向控制',
  }],
  findings: [{
    id: 'SEC-001',
    title: '示例管理端点暴露风险',
    severity: 'HIGH',
    confidence: 'CONFIRMED',
    category: 'Exposure',
    cwe: ['CWE-200'],
    standards: ['OWASP ASVS'],
    release_blocker: true,
    affected_assets: ['Gateway'],
    locations: [{ path: 'smart-recruit-gateway/router/router.go', line: 106, symbol: 'NewRouter' }],
    evidence: ['示例证据；未执行线上请求'],
    attack_scenario: '攻击者访问不应公开的管理端点。',
    impact: '泄露运行信息。',
    likelihood: 'HIGH',
    recommendation: '迁移到内部管理面并限制网络访问。',
    verification: ['公网不可达', '内部监控仍可用'],
    status: 'OPEN',
  }],
  residual_risks: ['示例未执行动态验证。'],
  remediation: {
    strategy: ['先处理发布阻断项，再进行累计复测。'],
    tasks: [{
      id: 'REM-001',
      finding_ids: ['SEC-001'],
      priority: 'P0',
      title: '限制 Gateway 管理端点',
      objective: '移除公网访问并保留内部监控。',
      rationale: '解决 SEC-001 发布阻断项。',
      status: 'PENDING',
      owner_hint: 'platform/security',
      dependencies: [],
      parallel_safe: false,
      human_confirmation_required: true,
      scope: {
        allowed_paths: ['smart-recruit-gateway/**', 'deploy/**'],
        excluded_paths: ['smart-recruit-proto/**'],
      },
      steps: ['拆分公共和管理监听面。', '在部署层限制管理监听面。'],
      acceptance_criteria: ['公网入口无法访问管理端点。', '内部监控可继续采集。'],
      tests: ['运行 Gateway 测试。', '在预发布验证公网和内网行为。'],
      rollback: '仅在隔离环境恢复旧配置以诊断问题。',
      security_notes: ['不可用路径隐藏代替网络隔离。'],
      deliverables: ['代码/配置变更', '回归测试', '预发布证据'],
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

const validate = (data) => {
  const errors = []
  if (data?.schema_version !== 1) errors.push('schema_version must be 1')
  requireObject(data?.audit, 'audit', errors)
  for (const field of ['executive_summary', 'attack_surface', 'coverage', 'checks', 'positive_controls', 'findings', 'residual_risks']) {
    requireArray(data?.[field], field, errors)
  }
  requireObject(data?.remediation, 'remediation', errors)
  requireArray(data?.remediation?.strategy, 'remediation.strategy', errors)
  requireArray(data?.remediation?.tasks, 'remediation.tasks', errors)

  const audit = data?.audit || {}
  for (const field of ['title', 'repo', 'branch', 'commit', 'generated_at', 'scope', 'authorized_test_level', 'verdict', 'overall_risk']) {
    requireText(audit[field], `audit.${field}`, errors)
  }
  requireArray(audit.standards, 'audit.standards', errors)
  requireArray(audit.limitations, 'audit.limitations', errors)
  if (!VERDICTS.includes(audit.verdict)) errors.push(`audit.verdict must be one of ${VERDICTS.join(', ')}`)
  if (!SEVERITIES.includes(audit.overall_risk)) errors.push(`audit.overall_risk must be one of ${SEVERITIES.join(', ')}`)

  const coverage = list(data?.coverage)
  const coverageIds = new Set()
  coverage.forEach((item, index) => {
    requireText(item?.id, `coverage[${index}].id`, errors)
    requireText(item?.domain, `coverage[${index}].domain`, errors)
    if (!COVERAGE_STATUSES.includes(item?.status)) errors.push(`coverage[${index}].status is invalid`)
    requireArray(item?.evidence, `coverage[${index}].evidence`, errors)
    requireArray(item?.limitations, `coverage[${index}].limitations`, errors)
    if (coverageIds.has(item?.id)) errors.push(`duplicate coverage id: ${item?.id}`)
    coverageIds.add(item?.id)
  })
  for (const [requiredId] of REQUIRED_COVERAGE) {
    if (!coverageIds.has(requiredId)) errors.push(`missing mandatory coverage domain: ${requiredId}`)
  }

  const checks = list(data?.checks)
  const checkIds = new Set()
  checks.forEach((item, index) => {
    requireText(item?.id, `checks[${index}].id`, errors)
    requireText(item?.domain, `checks[${index}].domain`, errors)
    if (!CHECK_STATUSES.includes(item?.status)) errors.push(`checks[${index}].status is invalid`)
    if (checkIds.has(item?.id)) errors.push(`duplicate check id: ${item?.id}`)
    checkIds.add(item?.id)
  })

  const findings = list(data?.findings)
  const findingIds = new Set()
  findings.forEach((finding, index) => {
    const prefix = `findings[${index}]`
    for (const field of ['id', 'title', 'category', 'attack_scenario', 'impact', 'likelihood', 'recommendation', 'status']) {
      requireText(finding?.[field], `${prefix}.${field}`, errors)
    }
    if (!/^SEC-\d{3,}$/.test(finding?.id || '')) errors.push(`${prefix}.id must match SEC-NNN`)
    if (findingIds.has(finding?.id)) errors.push(`duplicate finding id: ${finding?.id}`)
    findingIds.add(finding?.id)
    if (!SEVERITIES.includes(finding?.severity)) errors.push(`${prefix}.severity is invalid`)
    if (!CONFIDENCES.includes(finding?.confidence)) errors.push(`${prefix}.confidence is invalid`)
    if (!FINDING_STATUSES.includes(finding?.status)) errors.push(`${prefix}.status is invalid`)
    if (typeof finding?.release_blocker !== 'boolean') errors.push(`${prefix}.release_blocker must be boolean`)
    for (const field of ['cwe', 'standards', 'affected_assets', 'locations', 'evidence', 'verification']) {
      requireArray(finding?.[field], `${prefix}.${field}`, errors)
    }
    list(finding?.locations).forEach((location, locationIndex) => {
      requireText(location?.path, `${prefix}.locations[${locationIndex}].path`, errors)
      if (path.isAbsolute(location?.path || '')) errors.push(`${prefix}.locations[${locationIndex}].path must be repository-relative`)
    })
  })

  const tasks = list(data?.remediation?.tasks)
  const taskIds = new Set()
  const mappedFindings = new Set()
  tasks.forEach((task, index) => {
    const prefix = `remediation.tasks[${index}]`
    for (const field of ['id', 'title', 'objective', 'rationale', 'status', 'rollback']) {
      requireText(task?.[field], `${prefix}.${field}`, errors)
    }
    if (!/^REM-\d{3,}$/.test(task?.id || '')) errors.push(`${prefix}.id must match REM-NNN`)
    if (taskIds.has(task?.id)) errors.push(`duplicate remediation task id: ${task?.id}`)
    taskIds.add(task?.id)
    if (!PRIORITIES.includes(task?.priority)) errors.push(`${prefix}.priority is invalid`)
    for (const field of ['finding_ids', 'dependencies', 'steps', 'acceptance_criteria', 'tests', 'security_notes', 'deliverables']) {
      requireArray(task?.[field], `${prefix}.${field}`, errors)
    }
    requireObject(task?.scope, `${prefix}.scope`, errors)
    requireArray(task?.scope?.allowed_paths, `${prefix}.scope.allowed_paths`, errors)
    requireArray(task?.scope?.excluded_paths, `${prefix}.scope.excluded_paths`, errors)
    if (typeof task?.parallel_safe !== 'boolean') errors.push(`${prefix}.parallel_safe must be boolean`)
    if (typeof task?.human_confirmation_required !== 'boolean') errors.push(`${prefix}.human_confirmation_required must be boolean`)
    list(task?.finding_ids).forEach((findingId) => {
      if (!findingIds.has(findingId)) errors.push(`${prefix} references unknown finding ${findingId}`)
      mappedFindings.add(findingId)
    })
  })

  findings.filter((finding) => isOpen(finding) && (finding.release_blocker || ['CRITICAL', 'HIGH'].includes(finding.severity)))
    .forEach((finding) => {
      if (!mappedFindings.has(finding.id)) errors.push(`${finding.id} must map to a remediation task`)
    })

  const blockers = findings.filter(isBlocking)
  if (audit.verdict === 'GO' && blockers.length > 0) errors.push('GO cannot contain open release blockers')
  if (audit.verdict === 'GO' && coverage.some((item) => !['REVIEWED', 'NOT_APPLICABLE'].includes(item.status))) {
    errors.push('GO requires every mandatory coverage domain to be REVIEWED or NOT_APPLICABLE')
  }
  if (audit.verdict === 'CONDITIONAL_GO' && findings.some((finding) => isOpen(finding) && finding.severity === 'CRITICAL')) {
    errors.push('CONDITIONAL_GO cannot contain an open CRITICAL finding')
  }
  if (audit.verdict === 'NO_GO' && blockers.length === 0 && list(audit.limitations).length === 0) {
    errors.push('NO_GO requires an open release blocker or a documented critical evidence limitation')
  }
  if (errors.length > 0) throw new Error(`audit contract validation failed:\n- ${errors.join('\n- ')}`)
}

const severityCounts = (findings) => Object.fromEntries(SEVERITIES.map((severity) => [
  severity,
  findings.filter((finding) => finding.severity === severity && !['FALSE_POSITIVE'].includes(finding.status)).length,
]))

const locationsText = (finding) => list(finding.locations).map((location) => {
  const suffix = location.line ? `:${location.line}` : ''
  const symbol = location.symbol ? ` (${location.symbol})` : ''
  return `${location.path}${suffix}${symbol}`
}).join('; ')

const markdownTable = (headers, rows) => {
  const output = [
    `| ${headers.map(md).join(' | ')} |`,
    `| ${headers.map(() => '---').join(' | ')} |`,
  ]
  for (const row of rows) output.push(`| ${row.map(md).join(' | ')} |`)
  return output.join('\n')
}

const bulletList = (values, empty = '无') => list(values).length > 0
  ? list(values).map((value) => `- ${value}`).join('\n')
  : `- ${empty}`

const renderMarkdownReport = (data) => {
  const audit = data.audit
  const findings = data.findings
  const counts = severityCounts(findings)
  const blockers = findings.filter(isBlocking)
  const lines = [
    `# ${audit.title}`,
    '',
    markdownTable(['字段', '值'], [
      ['仓库', audit.repo], ['分支', audit.branch], ['提交', audit.commit], ['工作树', audit.working_tree],
      ['生成时间', audit.generated_at], ['审查范围', audit.scope], ['授权测试级别', audit.authorized_test_level],
      ['上线结论', audit.verdict], ['整体风险', audit.overall_risk],
    ]),
    '',
    '> 本报告仅反映指定提交、已提供配置和已执行验证。未发现不等于不存在漏洞。',
    '',
    '## 执行摘要',
    '',
    bulletList(data.executive_summary),
    '',
    `发现统计：Critical ${counts.CRITICAL} / High ${counts.HIGH} / Medium ${counts.MEDIUM} / Low ${counts.LOW} / Info ${counts.INFO}；未关闭发布阻断项 ${blockers.length}。`,
    '',
    '## 标准与限制',
    '',
    '### 适用标准', '', bulletList(audit.standards), '',
    '### 审查限制', '', bulletList(audit.limitations), '',
    '## 攻击面与信任边界', '',
    markdownTable(['攻击面', '资产', '入口', '信任边界', '说明'], list(data.attack_surface).map((item) => [
      item.surface, list(item.assets).join(', '), list(item.entry_points).join(', '), list(item.trust_boundaries).join(', '), item.notes,
    ])),
    '',
    '## 覆盖矩阵', '',
    markdownTable(['领域', '状态', '证据', '限制'], list(data.coverage).map((item) => [
      item.domain, item.status, list(item.evidence).join('; '), list(item.limitations).join('; '),
    ])),
    '',
    '## 安全检查与测试', '',
    markdownTable(['ID', '领域', '命令/检查', '状态', '结果', '证据'], list(data.checks).map((item) => [
      item.id, item.domain, item.command, item.status, item.result, list(item.evidence).join('; '),
    ])),
    '',
    '## 已验证正向控制', '',
    markdownTable(['控制', '证据', '边界/说明'], list(data.positive_controls).map((item) => [
      item.title, list(item.evidence).join('; '), item.notes,
    ])),
    '',
    '## 安全发现', '',
  ]

  if (findings.length === 0) lines.push('未记录正式安全发现。', '')
  for (const finding of findings) {
    lines.push(
      `### ${finding.id} · ${finding.title}`,
      '',
      markdownTable(['字段', '值'], [
        ['严重度', finding.severity], ['置信度', finding.confidence], ['状态', finding.status],
        ['发布阻断', finding.release_blocker ? '是' : '否'], ['类别', finding.category],
        ['可能性', finding.likelihood], ['CWE', list(finding.cwe).join(', ')],
        ['标准映射', list(finding.standards).join(', ')], ['受影响资产', list(finding.affected_assets).join(', ')],
        ['位置', locationsText(finding)],
      ]),
      '',
      '**证据**', '', bulletList(finding.evidence), '',
      '**攻击场景**', '', finding.attack_scenario, '',
      '**影响**', '', finding.impact, '',
      '**修复建议**', '', finding.recommendation, '',
      '**复测条件**', '', bulletList(finding.verification), '',
    )
  }

  lines.push(
    '## 修复优先级概览', '',
    bulletList(data.remediation.strategy), '',
    markdownTable(['任务', '优先级', '关联发现', '标题', '状态'], list(data.remediation.tasks).map((task) => [
      task.id, task.priority, list(task.finding_ids).join(', '), task.title, task.status,
    ])),
    '',
    '## 残余风险', '',
    bulletList(data.residual_risks), '',
    '## 上线结论', '',
    `**${audit.verdict}** — 整体风险：**${audit.overall_risk}**；未关闭发布阻断项：**${blockers.length}**。`, '',
  )
  return `${lines.join('\n')}\n`
}

const htmlList = (values, empty = '无') => `<ul>${(list(values).length ? list(values) : [empty]).map((value) => `<li>${esc(value)}</li>`).join('')}</ul>`

const htmlTable = (headers, rows) => `<div class="table-wrap"><table><thead><tr>${headers.map((header) => `<th>${esc(header)}</th>`).join('')}</tr></thead><tbody>${rows.map((row) => `<tr>${row.map((cell) => `<td>${esc(cell)}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`

const renderHtmlReport = (data) => {
  const audit = data.audit
  const findings = data.findings
  const counts = severityCounts(findings)
  const blockers = findings.filter(isBlocking)
  const findingHtml = findings.length === 0 ? '<p>未记录正式安全发现。</p>' : findings.map((finding) => `
    <article class="finding severity-${finding.severity.toLowerCase()}">
      <h3>${esc(finding.id)} · ${esc(finding.title)}</h3>
      <div class="badges"><span>${esc(finding.severity)}</span><span>${esc(finding.confidence)}</span><span>${esc(finding.status)}</span>${finding.release_blocker ? '<span class="blocker">RELEASE BLOCKER</span>' : ''}</div>
      ${htmlTable(['字段', '值'], [
        ['类别', finding.category], ['可能性', finding.likelihood], ['CWE', list(finding.cwe).join(', ')],
        ['标准映射', list(finding.standards).join(', ')], ['受影响资产', list(finding.affected_assets).join(', ')],
        ['位置', locationsText(finding)],
      ])}
      <h4>证据</h4>${htmlList(finding.evidence)}
      <h4>攻击场景</h4><p>${esc(finding.attack_scenario)}</p>
      <h4>影响</h4><p>${esc(finding.impact)}</p>
      <h4>修复建议</h4><p>${esc(finding.recommendation)}</p>
      <h4>复测条件</h4>${htmlList(finding.verification)}
    </article>`).join('')

  return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(audit.title)}</title>
<style>
:root{--bg:#f4f7fb;--card:#fff;--ink:#172033;--muted:#5b6579;--line:#dde3ee;--critical:#8b1e2d;--high:#c43c35;--medium:#b36b00;--low:#2d6a4f;--accent:#234eaa}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}.page{max-width:1320px;margin:auto;padding:28px}.hero{background:linear-gradient(135deg,#14213d,#234eaa);color:#fff;padding:30px;border-radius:14px;box-shadow:0 12px 30px #17203322}.hero h1{margin:0 0 12px;font-size:28px}.hero p{margin:4px 0;color:#e7eeff}.verdict{display:inline-block;margin-top:14px;padding:8px 14px;border:1px solid #ffffff66;border-radius:999px;font-weight:800}.grid{display:grid;grid-template-columns:repeat(5,1fr);gap:12px;margin:18px 0}.metric,.section,.finding{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px}.metric strong{display:block;font-size:24px}.metric span{color:var(--muted)}.section{margin:16px 0}.section h2{margin:0 0 12px;font-size:20px}.finding{margin:12px 0;border-left-width:6px}.severity-critical{border-left-color:var(--critical)}.severity-high{border-left-color:var(--high)}.severity-medium{border-left-color:var(--medium)}.severity-low,.severity-info{border-left-color:var(--low)}h3{margin:0 0 8px}h4{margin:14px 0 4px}.badges{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:10px}.badges span{background:#edf1f7;border-radius:999px;padding:3px 9px;font-size:12px;font-weight:700}.badges .blocker{background:#ffe0e0;color:#8b1e2d}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%;margin:8px 0}th,td{border-bottom:1px solid var(--line);padding:9px 10px;text-align:left;vertical-align:top;word-break:break-word}th{background:#f7f9fc}code{background:#eef2f7;padding:2px 5px;border-radius:4px}ul{margin:6px 0;padding-left:22px}.notice{border-left:4px solid var(--medium);padding:10px 14px;background:#fff6df}.footer{color:var(--muted);text-align:center;padding:20px}@media(max-width:800px){.page{padding:12px}.grid{grid-template-columns:repeat(2,1fr)}.hero h1{font-size:22px}}
</style>
</head>
<body><main class="page">
<header class="hero"><h1>${esc(audit.title)}</h1><p>仓库：${esc(audit.repo)}　分支：${esc(audit.branch)}　提交：${esc(audit.commit)}</p><p>范围：${esc(audit.scope)}　授权级别：${esc(audit.authorized_test_level)}　生成时间：${esc(audit.generated_at)}</p><div class="verdict">${esc(audit.verdict)} · ${esc(audit.overall_risk)}</div></header>
<section class="grid"><div class="metric"><strong>${counts.CRITICAL}</strong><span>Critical</span></div><div class="metric"><strong>${counts.HIGH}</strong><span>High</span></div><div class="metric"><strong>${counts.MEDIUM}</strong><span>Medium</span></div><div class="metric"><strong>${counts.LOW}</strong><span>Low</span></div><div class="metric"><strong>${blockers.length}</strong><span>Release blockers</span></div></section>
<section class="section"><h2>执行摘要</h2>${htmlList(data.executive_summary)}<p class="notice">本报告仅反映指定提交、已提供配置和已执行验证。未发现不等于不存在漏洞。</p></section>
<section class="section"><h2>标准与限制</h2><h4>适用标准</h4>${htmlList(audit.standards)}<h4>审查限制</h4>${htmlList(audit.limitations)}</section>
<section class="section"><h2>攻击面与信任边界</h2>${htmlTable(['攻击面','资产','入口','信任边界','说明'],list(data.attack_surface).map((item)=>[item.surface,list(item.assets).join(', '),list(item.entry_points).join(', '),list(item.trust_boundaries).join(', '),item.notes]))}</section>
<section class="section"><h2>覆盖矩阵</h2>${htmlTable(['领域','状态','证据','限制'],list(data.coverage).map((item)=>[item.domain,item.status,list(item.evidence).join('; '),list(item.limitations).join('; ')]))}</section>
<section class="section"><h2>安全检查与测试</h2>${htmlTable(['ID','领域','命令/检查','状态','结果','证据'],list(data.checks).map((item)=>[item.id,item.domain,item.command,item.status,item.result,list(item.evidence).join('; ')]))}</section>
<section class="section"><h2>已验证正向控制</h2>${htmlTable(['控制','证据','边界/说明'],list(data.positive_controls).map((item)=>[item.title,list(item.evidence).join('; '),item.notes]))}</section>
<section class="section"><h2>安全发现</h2>${findingHtml}</section>
<section class="section"><h2>修复优先级概览</h2>${htmlList(data.remediation.strategy)}${htmlTable(['任务','优先级','关联发现','标题','状态'],list(data.remediation.tasks).map((task)=>[task.id,task.priority,list(task.finding_ids).join(', '),task.title,task.status]))}</section>
<section class="section"><h2>残余风险</h2>${htmlList(data.residual_risks)}</section>
<section class="section"><h2>上线结论</h2><p><strong>${esc(audit.verdict)}</strong> — 整体风险：<strong>${esc(audit.overall_risk)}</strong>；未关闭发布阻断项：<strong>${blockers.length}</strong>。</p></section>
<footer class="footer">Generated from one validated audit JSON source. HTML is self-contained and performs no network requests.</footer>
</main></body></html>`
}

const renderFixPlan = (data) => {
  const audit = data.audit
  const tasks = data.remediation.tasks
  const lines = [
    '# Smart Recruit 安全修复执行计划', '',
    markdownTable(['字段', '值'], [
      ['来源审查', audit.title], ['仓库/分支', `${audit.repo} / ${audit.branch}`], ['审查提交', audit.commit],
      ['审查结论', audit.verdict], ['整体风险', audit.overall_risk], ['生成时间', audit.generated_at],
    ]), '',
    '> 本计划面向 Codex 等实现 Agent。它不授权访问或修改生产系统，也不自动扩大文件范围。执行前必须重新读取 `AGENTS.md` 和适用知识。', '',
    '## 执行规则', '',
    '- 默认连续执行全部任务；按依赖和 P0、P1、P2、P3 优先级串行处理。',
    '- 每次只实施一个边界清晰的任务；通过后自动进入下一任务，并保留用户已有工作树变更。',
    '- 不打印、复制或提交真实密钥和个人数据。',
    '- 涉及公共 API、Proto、Schema、共享鉴权策略、生产配置、密钥轮换或外部系统时，先取得明确确认。',
    '- 不以隐藏路径、吞掉错误或关闭安全检查作为修复。',
    '- 每项修复必须增加或更新负向回归测试，并在完成后复核原安全发现。', '',
    '## 修复策略', '', bulletList(data.remediation.strategy), '',
    '## 有序任务总览', '',
    markdownTable(['顺序', '任务', '优先级', '关联发现', '依赖', '可并行', '需确认', '状态'], tasks.map((task, index) => [
      index + 1, task.id, task.priority, list(task.finding_ids).join(', '), list(task.dependencies).join(', '),
      task.parallel_safe ? '是' : '否', task.human_confirmation_required ? '是' : '否', task.status,
    ])), '',
  ]

  if (tasks.length === 0) lines.push('当前没有修复任务。', '')
  for (const task of tasks) {
    lines.push(
      `## ${task.id} · ${task.title}`, '',
      markdownTable(['字段', '值'], [
        ['优先级', task.priority], ['关联发现', list(task.finding_ids).join(', ')], ['状态', task.status],
        ['建议负责人', task.owner_hint], ['依赖', list(task.dependencies).join(', ') || '无'],
        ['可并行', task.parallel_safe ? '是' : '否'], ['需要人工确认', task.human_confirmation_required ? '是' : '否'],
      ]), '',
      '### 目标', '', task.objective, '',
      '### 安全理由', '', task.rationale, '',
      '### 允许范围', '', bulletList(task.scope.allowed_paths), '',
      '### 排除范围', '', bulletList(task.scope.excluded_paths), '',
      '### 实施步骤', '', list(task.steps).map((step, index) => `${index + 1}. ${step}`).join('\n') || '1. 未提供', '',
      '### 验收标准', '', bulletList(task.acceptance_criteria), '',
      '### 验证命令/测试', '', bulletList(task.tests), '',
      '### 回滚或安全遏制', '', task.rollback, '',
      '### 安全注意事项', '', bulletList(task.security_notes), '',
      '### 必须交付', '', bulletList(task.deliverables), '',
    )
  }

  const traceRows = data.findings.map((finding) => [
    finding.id, finding.severity, finding.release_blocker ? '是' : '否',
    tasks.filter((task) => list(task.finding_ids).includes(finding.id)).map((task) => task.id).join(', ') || '未映射',
  ])
  lines.push(
    '## 发现到任务追踪矩阵', '',
    markdownTable(['发现', '严重度', '发布阻断', '修复任务'], traceRows), '',
    '## 最终累计验证门禁', '',
    '- 所有 P0/P1 完成；任何例外均由有权责任人书面接受并注明到期日。',
    '- 受影响的 Go 测试、前端测试、类型检查和专项负向安全测试通过。',
    '- 按影响范围重新执行 Secret、依赖、SAST、镜像与 IaC 扫描。',
    '- 在经授权的预发布环境复测租户隔离、身份授权、AI/MCP、支付和部署边界。',
    '- 更新审查 JSON，并重新生成配套 Markdown、HTML 和本修复计划。',
    '- 重新作出 `GO`、`CONDITIONAL_GO` 或 `NO_GO` 结论。', '',
  )
  const manifest = {
    schema_version: 1,
    artifact_type: 'prelaunch-security-remediation-plan',
    source_format: 'embedded-manifest',
    audit: data.audit,
    findings: data.findings.map((finding) => ({
      id: finding.id,
      title: finding.title,
      severity: finding.severity,
      confidence: finding.confidence,
      release_blocker: finding.release_blocker,
      status: finding.status,
      locations: finding.locations,
      verification: finding.verification,
    })),
    remediation: data.remediation,
  }
  lines.push(
    '## Machine-readable remediation manifest', '',
    '> 该区块由审查渲染器生成，供 `$prelaunch-security-fix` 确定性解析。不要手工编辑；应修改审查 JSON 后重新生成全部产物。', '',
    '```prelaunch-security-remediation-manifest',
    JSON.stringify(manifest, null, 2),
    '```', '',
  )
  return `${lines.join('\n')}\n`
}

const safeBasename = (value) => {
  if (!/^[a-zA-Z0-9._-]+$/.test(value)) throw new Error('basename may contain only letters, digits, dot, underscore, and hyphen')
  return value
}

const timestamp = () => new Date().toISOString().replace(/[-:]/g, '').replace('T', '-').slice(0, 15)

const writeNew = (file, content) => {
  if (fs.existsSync(file)) throw new Error(`refusing to overwrite existing file: ${file}`)
  fs.writeFileSync(file, content, { encoding: 'utf8', flag: 'wx' })
}

const main = () => {
  const args = parseArgs(process.argv)
  if (args.help) {
    process.stdout.write(usage())
    return
  }
  if (!args.sample && !args.input) throw new Error(`--input is required unless --sample is used\n${usage()}`)
  const data = args.sample ? samplePayload() : JSON.parse(fs.readFileSync(args.input, 'utf8'))
  validate(data)

  const outputDir = path.resolve(args.outputDir)
  fs.mkdirSync(outputDir, { recursive: true })
  const basename = safeBasename(args.basename || `prelaunch-security-audit-${timestamp()}`)
  const markdownPath = path.join(outputDir, `${basename}-security-report.md`)
  const htmlPath = path.join(outputDir, `${basename}-security-report.html`)
  const remediationPath = path.join(outputDir, `${basename}-remediation-plan.md`)

  writeNew(markdownPath, renderMarkdownReport(data))
  writeNew(htmlPath, renderHtmlReport(data))
  writeNew(remediationPath, renderFixPlan(data))

  process.stdout.write(`${JSON.stringify({
    verdict: data.audit.verdict,
    security_report_markdown: markdownPath,
    security_report_html: htmlPath,
    remediation_plan_markdown: remediationPath,
  }, null, 2)}\n`)
}

try {
  main()
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
