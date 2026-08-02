#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'

const DEFAULT_TITLE = 'Code Review Leader — 变更可视化报告'
const DEFAULT_OUTPUT_DIR = '.code-review-sdd'

const esc = (value) => String(value ?? '')
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')
  .replaceAll("'", '&#39;')

const mdEsc = (value) => String(value ?? '').replaceAll('|', '\\|')

const stripHtml = (value) => String(value ?? '')
  .replace(/<br\s*\/?>/gi, '\n')
  .replace(/<\/p\s*>/gi, '\n\n')
  .replace(/<\/li\s*>/gi, '\n')
  .replace(/<li\s*>/gi, '- ')
  .replace(/<\/?(ul|ol|div|span|strong|b|em|code|h\d)[^>]*>/gi, '')
  .replace(/<[^>]+>/g, '')
  .replaceAll('&lt;', '<')
  .replaceAll('&gt;', '>')
  .replaceAll('&amp;', '&')
  .replaceAll('&quot;', '"')
  .replaceAll('&#39;', "'")
  .trim()

const riskClass = (value) => {
  const text = String(value ?? '').toLowerCase()
  if (['high', 'p0', 'p1', '高', '高风险'].includes(text)) return 'risk-high'
  if (['medium', 'p2', '中', '中风险', 'warning'].includes(text)) return 'risk-medium'
  return 'risk-low'
}

const priorityBadge = (priority) => {
  const p = String(priority || 'P3').toUpperCase()
  const cls = ['P0', 'P1'].includes(p) ? 'badge-violation' : p === 'P2' ? 'badge-attention' : 'badge-compliant'
  return `<span class="badge ${cls}">${esc(p)}</span>`
}

const renderSummary = (summary) => {
  const items = summary?.length ? summary : [
    { label: '🔍 审查范围', content_html: '<p>未提供审查范围。</p>' },
    { label: '⚡ 整体评估', risk: true, content_html: '<p>未提供整体评估。</p>' },
  ]
  const parts = ['<div class="summary-bar" id="section-summary"><h2>整体说明</h2>']
  for (const item of items) {
    const cls = item.risk ? 'summary-item summary-item--risk' : 'summary-item'
    parts.push(`<div class="${cls}">`)
    parts.push(`<div class="summary-item-label">${esc(item.label || '摘要')}</div>`)
    parts.push(`<div class="summary-item-content">${item.content_html || esc(item.text || '')}</div>`)
    parts.push('</div>')
  }
  parts.push('</div>')
  return parts.join('\n')
}

const renderTable = (block) => {
  const headers = block.headers || []
  const rows = block.rows || []
  const sortable = block.sortable !== false
  const ths = headers.map((header) => {
    const attr = sortable ? ' data-sortable' : ''
    return `<th${attr}>${esc(header)}${sortable ? '<span class="sort-icon">⇅</span>' : ''}</th>`
  }).join('')
  const bodyRows = rows.map((row) => {
    const cells = row.map((cell) => {
      if (cell && typeof cell === 'object' && 'html' in cell) return `<td>${cell.html}</td>`
      return `<td>${esc(cell)}</td>`
    }).join('')
    return `<tr>${cells}</tr>`
  }).join('')
  return `<table><thead><tr>${ths}</tr></thead><tbody>${bodyRows}</tbody></table>`
}

const renderList = (block) => {
  const tag = block.ordered ? 'ol' : 'ul'
  const items = (block.items || []).map((item) => `<li>${esc(item)}</li>`).join('')
  return `<${tag}>${items}</${tag}>`
}

let diagramSequence = 0

const parseMermaidNode = (token) => {
  const match = String(token || '').trim().match(/^([A-Za-z][A-Za-z0-9_-]*)(?:\[(.*)\]|\((.*)\)|\{(.*)\})?$/)
  if (!match) return null
  const rawLabel = match[2] ?? match[3] ?? match[4] ?? match[1]
  return {
    id: match[1],
    label: String(rawLabel).trim().replace(/^["']|["']$/g, ''),
  }
}

const parseOfflineFlowchart = (source) => {
  const lines = String(source || '')
    .split(/\r?\n/)
    .map((line) => line.trim().replace(/;$/, ''))
    .filter(Boolean)
  const header = lines.shift()?.match(/^(?:flowchart|graph)\s+(TD|TB|LR|RL)$/i)
  if (!header) return null

  const direction = header[1].toUpperCase()
  const nodes = new Map()
  const edges = []
  const mergeNode = (node) => {
    const current = nodes.get(node.id)
    if (!current || (current.label === current.id && node.label !== node.id)) {
      nodes.set(node.id, node)
    }
  }
  for (const line of lines) {
    const edge = line.match(/^(.+?)\s*(-->|==>|-\.->)\s*(.+)$/)
    if (!edge) return null
    const from = parseMermaidNode(edge[1])
    const to = parseMermaidNode(edge[3])
    if (!from || !to) return null
    mergeNode(from)
    mergeNode(to)
    edges.push({ from: from.id, to: to.id })
  }
  if (!nodes.size || !edges.length) return null
  return { direction, nodes, edges }
}

const renderOfflineFlowchart = (source) => {
  const graph = parseOfflineFlowchart(source)
  if (!graph) return null

  const outgoing = new Map([...graph.nodes.keys()].map((id) => [id, []]))
  const indegree = new Map([...graph.nodes.keys()].map((id) => [id, 0]))
  for (const edge of graph.edges) {
    outgoing.get(edge.from).push(edge.to)
    indegree.set(edge.to, (indegree.get(edge.to) || 0) + 1)
  }

  const layers = new Map()
  const queue = [...graph.nodes.keys()].filter((id) => indegree.get(id) === 0)
  if (!queue.length) return null
  for (const id of queue) layers.set(id, 0)
  for (let index = 0; index < queue.length; index += 1) {
    const id = queue[index]
    for (const target of outgoing.get(id) || []) {
      layers.set(target, Math.max(layers.get(target) || 0, (layers.get(id) || 0) + 1))
      indegree.set(target, (indegree.get(target) || 0) - 1)
      if (indegree.get(target) === 0) queue.push(target)
    }
  }
  if (layers.size !== graph.nodes.size) return null

  const grouped = []
  for (const id of graph.nodes.keys()) {
    const layer = layers.get(id) || 0
    grouped[layer] ||= []
    grouped[layer].push(id)
  }

  const nodeWidth = 190
  const nodeHeight = 54
  const layerGap = 72
  const nodeGap = 42
  const padding = 28
  const horizontal = graph.direction === 'LR' || graph.direction === 'RL'
  const maxLayerSize = Math.max(...grouped.map((items) => items.length))
  const primarySize = padding * 2 + grouped.length * (horizontal ? nodeWidth : nodeHeight) + Math.max(0, grouped.length - 1) * layerGap
  const secondarySize = padding * 2 + maxLayerSize * (horizontal ? nodeHeight : nodeWidth) + Math.max(0, maxLayerSize - 1) * nodeGap
  const width = horizontal ? primarySize : secondarySize
  const height = horizontal ? secondarySize : primarySize
  const positions = new Map()

  grouped.forEach((ids, layer) => {
    const layerSpan = ids.length * (horizontal ? nodeHeight : nodeWidth) + Math.max(0, ids.length - 1) * nodeGap
    ids.forEach((id, index) => {
      const primary = padding + layer * ((horizontal ? nodeWidth : nodeHeight) + layerGap)
      const secondary = (secondarySize - layerSpan) / 2 + index * ((horizontal ? nodeHeight : nodeWidth) + nodeGap)
      positions.set(id, horizontal
        ? { x: primary, y: secondary }
        : { x: secondary, y: primary })
    })
  })

  const reverse = graph.direction === 'RL'
  if (reverse) {
    for (const position of positions.values()) {
      position.x = width - padding - nodeWidth - (position.x - padding)
    }
  }

  const markerId = `offline-flow-arrow-${++diagramSequence}`
  const edgeSVG = graph.edges.map((edge) => {
    const from = positions.get(edge.from)
    const to = positions.get(edge.to)
    const coords = horizontal
      ? [from.x + nodeWidth, from.y + nodeHeight / 2, to.x, to.y + nodeHeight / 2]
      : [from.x + nodeWidth / 2, from.y + nodeHeight, to.x + nodeWidth / 2, to.y]
    return `<line x1="${coords[0]}" y1="${coords[1]}" x2="${coords[2]}" y2="${coords[3]}" marker-end="url(#${markerId})"/>`
  }).join('')
  const nodeSVG = [...graph.nodes.values()].map((node) => {
    const position = positions.get(node.id)
    const label = node.label.length > 26 ? `${node.label.slice(0, 25)}…` : node.label
    return `<g><rect x="${position.x}" y="${position.y}" width="${nodeWidth}" height="${nodeHeight}" rx="8"/>`
      + `<text x="${position.x + nodeWidth / 2}" y="${position.y + nodeHeight / 2}" dominant-baseline="middle" text-anchor="middle">${esc(label)}</text></g>`
  }).join('')

  return `<svg class="offline-flowchart" viewBox="0 0 ${width} ${height}" role="img" aria-label="离线流程图">`
    + `<defs><marker id="${markerId}" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z"/></marker></defs>`
    + `<g class="offline-flowchart__edges">${edgeSVG}</g><g class="offline-flowchart__nodes">${nodeSVG}</g></svg>`
}

const renderMermaid = (source) => {
  const diagram = renderOfflineFlowchart(source)
  const status = diagram
    ? '<p class="diagram-note">离线静态流程图</p>'
    : '<p class="diagram-note diagram-note--fallback">当前 Mermaid 语法超出离线渲染子集，以下源码保持完整可读。</p>'
  return [
    '<figure class="mermaid-container">',
    status,
    diagram || '',
    '<details class="mermaid-source-details">',
    '<summary>查看 Mermaid 源码</summary>',
    `<pre class="mermaid-source">${esc(source)}</pre>`,
    '</details>',
    '</figure>',
  ].join('')
}

const renderFindings = (items) => {
  if (!items?.length) return '<p><span class="badge badge-compliant">未发现阻塞性问题</span></p>'
  const rows = items.map((item) => {
    let loc = esc(item.file || '')
    if (item.line) loc += `:${esc(item.line)}`
    return '<tr>'
      + `<td>${priorityBadge(item.priority)}</td>`
      + `<td><strong>${esc(item.title || '问题')}</strong></td>`
      + `<td><code>${loc}</code></td>`
      + `<td class="${riskClass(item.risk)}">${esc(item.risk || 'medium')}</td>`
      + `<td>${esc(item.detail || '')}</td>`
      + `<td>${esc(item.recommendation || '')}</td>`
      + '</tr>'
  }).join('')
  return '<table><thead><tr>'
    + '<th data-sortable>优先级<span class="sort-icon">⇅</span></th>'
    + '<th>问题</th><th>位置</th><th data-sortable>风险<span class="sort-icon">⇅</span></th>'
    + '<th>影响</th><th>建议</th>'
    + `</tr></thead><tbody>${rows}</tbody></table>`
}

const renderBlock = (block) => {
  const kind = block.type || 'paragraph'
  if (kind === 'html') return String(block.html || '')
  if (kind === 'paragraph') return `<p>${esc(block.text || '')}</p>`
  if (kind === 'list') return renderList(block)
  if (kind === 'table') return renderTable(block)
  if (kind === 'mermaid') return renderMermaid(String(block.source || ''))
  if (kind === 'findings') return renderFindings(block.items || [])
  return `<p>${esc(JSON.stringify(block))}</p>`
}

const renderSections = (sections) => (sections || []).map((section) => {
  const parts = [
    `<div class="section"><div class="section-header"><h2>${esc(section.title || '未命名章节')}</h2><span class="toggle">▼</span></div><div class="section-body">`,
  ]
  if (section.content_html) parts.push(String(section.content_html))
  if (section.text) parts.push(`<p>${esc(section.text)}</p>`)
  if (section.mermaid) parts.push(renderMermaid(String(section.mermaid)))
  for (const block of section.blocks || []) parts.push(renderBlock(block))
  parts.push('</div></div>')
  return parts.join('\n')
}).join('\n')

const samplePayload = () => ({
  title: DEFAULT_TITLE,
  repo: 'sample-repo',
  branch: 'feature/example',
  generated_at: new Date().toISOString().slice(0, 16).replace('T', ' '),
  scope: 'commit abc1234 (abc1234^..abc1234)',
  scope_type: 'commit',
  reviewed_commit: 'abc1234',
  verdict: '需修复后合入',
  risk_level: '需重点关注',
  summary: [
    { label: '🔍 审查范围', content_html: '<p>当前工作区未提交变更。</p>' },
    { label: '📦 变更规模', content_html: '<ul><li>3 个文件，120 insertions / 8 deletions</li></ul>' },
    { label: '⚡ 整体评估', risk: true, content_html: '<p>存在 1 个 P1 问题，建议修复后合入。</p>' },
  ],
  sections: [
    {
      title: '🔗 D1: 变更逻辑链',
      blocks: [
        { type: 'paragraph', text: '本次变更新增 API 层校验并调整前端调用。' },
        { type: 'mermaid', source: 'flowchart TD\nA[前端] --> B[API]\nB --> C[数据库]' },
      ],
    },
    {
      title: '🎯 D5: 风险热力矩阵',
      blocks: [{
        type: 'findings',
        items: [{
          id: 'F-001',
          priority: 'P1',
          title: '缺少权限校验',
          file: 'server/api.ts',
          line: 42,
          risk: 'high',
          detail: '未登录用户可触发写入路径。',
          recommendation: '在 handler 入口增加认证中间件。',
        }],
      }],
    },
  ],
  remediation: {
    strategy: [
      '先修复阻塞合入的权限校验缺口',
      '补充负向回归测试后重新审查',
    ],
    tasks: [{
      id: 'CR-001',
      finding_ids: ['F-001'],
      priority: 'P1',
      title: '为写入 API 增加认证中间件',
      objective: '未登录请求不能触发写入路径。',
      rationale: '审查确认 handler 缺少认证，存在未授权写入风险。',
      status: 'PENDING',
      dependencies: [],
      parallel_safe: false,
      human_confirmation_required: false,
      scope: {
        allowed_paths: ['server/api.ts', 'server/api.test.ts'],
        excluded_paths: [],
      },
      source_evidence: ['server/api.ts:42'],
      steps: [
        '复现未登录可写入的问题路径',
        '在 handler 入口增加认证中间件',
        '补充负向回归测试',
        '运行相关测试确认通过',
      ],
      acceptance_criteria: [
        '未登录请求被拒绝',
        '已登录合法请求行为不变',
        '相关测试通过',
      ],
      tests: ['node --test server/api.test.ts'],
      rollback: '通过 git 恢复本任务引入的变更',
      deliverables: ['代码修复', '回归测试', '验证证据'],
    }],
  },
})

const PRIORITY_RANK = { P0: 0, P1: 1, P2: 2, P3: 3 }

const asList = (value) => (Array.isArray(value) ? value : [])

const collectFindings = (data) => {
  const items = []
  for (const section of asList(data?.sections)) {
    for (const block of asList(section?.blocks)) {
      if (block?.type !== 'findings') continue
      for (const item of asList(block.items)) {
        if (item && (item.title || item.detail || item.file || item.recommendation)) items.push(item)
      }
    }
  }
  return items
}

const normalizeFinding = (item, index) => {
  const id = String(item.id || item.finding_id || `F-${String(index + 1).padStart(3, '0')}`)
  return {
    id,
    title: String(item.title || '未命名问题'),
    priority: String(item.priority || 'P2').toUpperCase(),
    risk: String(item.risk || 'medium'),
    file: item.file ? String(item.file) : '',
    line: item.line == null || item.line === '' ? null : Number(item.line) || String(item.line),
    detail: String(item.detail || ''),
    recommendation: String(item.recommendation || ''),
    status: String(item.status || 'OPEN'),
  }
}

const findingToTask = (finding, index) => {
  const loc = finding.file ? `${finding.file}${finding.line != null ? `:${finding.line}` : ''}` : ''
  const priority = ['P0', 'P1', 'P2', 'P3'].includes(finding.priority) ? finding.priority : 'P2'
  return {
    id: `CR-${String(index + 1).padStart(3, '0')}`,
    finding_ids: [finding.id],
    priority,
    title: finding.title,
    objective: finding.recommendation || finding.detail || finding.title,
    rationale: finding.detail || finding.title,
    status: 'PENDING',
    dependencies: [],
    parallel_safe: false,
    human_confirmation_required: priority === 'P0',
    scope: {
      allowed_paths: finding.file ? [finding.file] : [],
      excluded_paths: [],
    },
    source_evidence: loc ? [loc] : [],
    steps: [
      `复现并确认问题：${finding.title}`,
      finding.recommendation || '按审查建议完成根因修复',
      '补充或更新回归测试',
      '运行相关验证并确认原问题不再复现',
    ],
    acceptance_criteria: [
      '原问题在当前证据下不再复现',
      '相关测试或检查通过',
      '未引入超出任务范围的无关改动',
    ],
    tests: [],
    rollback: '通过 git 恢复本任务引入的变更',
    deliverables: ['代码修复', '验证证据'],
  }
}

const normalizeTask = (task, index) => {
  const id = String(task.id || `CR-${String(index + 1).padStart(3, '0')}`)
  const priority = String(task.priority || 'P2').toUpperCase()
  return {
    id,
    finding_ids: asList(task.finding_ids).map(String),
    priority: ['P0', 'P1', 'P2', 'P3'].includes(priority) ? priority : 'P2',
    title: String(task.title || '未命名修复任务'),
    objective: String(task.objective || ''),
    rationale: String(task.rationale || ''),
    status: String(task.status || 'PENDING'),
    dependencies: asList(task.dependencies).map(String),
    parallel_safe: Boolean(task.parallel_safe),
    human_confirmation_required: Boolean(task.human_confirmation_required),
    scope: {
      allowed_paths: asList(task.scope?.allowed_paths).map(String),
      excluded_paths: asList(task.scope?.excluded_paths).map(String),
    },
    source_evidence: asList(task.source_evidence).map(String),
    steps: asList(task.steps).map(String),
    acceptance_criteria: asList(task.acceptance_criteria).map(String),
    tests: asList(task.tests).map(String),
    rollback: String(task.rollback || '通过 git 恢复本任务引入的变更'),
    deliverables: asList(task.deliverables).map(String),
  }
}

const orderTasks = (tasks) => [...tasks].sort((left, right) => {
  const rank = (PRIORITY_RANK[left.priority] ?? 99) - (PRIORITY_RANK[right.priority] ?? 99)
  if (rank !== 0) return rank
  return String(left.id).localeCompare(String(right.id))
})

const buildRemediation = (data) => {
  const findings = collectFindings(data).map(normalizeFinding)
  const providedTasks = asList(data?.remediation?.tasks).map(normalizeTask)
  const tasks = providedTasks.length
    ? providedTasks
    : findings.map((finding, index) => findingToTask(finding, index))
  if (!tasks.length) return null
  const findingById = new Map(findings.map((item) => [item.id, item]))
  for (const task of tasks) {
    for (const findingId of task.finding_ids) {
      if (!findingById.has(findingId)) {
        findingById.set(findingId, {
          id: findingId,
          title: task.title,
          priority: task.priority,
          risk: 'medium',
          file: '',
          line: null,
          detail: task.rationale,
          recommendation: task.objective,
          status: 'OPEN',
        })
      }
    }
  }
  return {
    strategy: asList(data?.remediation?.strategy).map(String),
    findings: [...findingById.values()],
    tasks: orderTasks(tasks),
  }
}

const joinText = (values) => (asList(values).length ? asList(values).join('、') : '无')

const code = (value) => `\`${String(value ?? '')}\``

const reviewManifest = (data, reportPaths = {}) => ({
  title: data.title || DEFAULT_TITLE,
  repo: data.repo || '',
  branch: data.branch || '',
  scope: data.scope || '',
  scope_type: data.scope_type || '',
  reviewed_commit: data.reviewed_commit || '',
  verdict: data.verdict || '',
  risk_level: data.risk_level || '',
  generated_at: data.generated_at || '',
  html_report: reportPaths.html || '',
  markdown_report: reportPaths.markdown || '',
})

const sddDocument = (data, remediation, reportPaths = {}) => {
  const tasks = remediation.tasks
  const findings = remediation.findings
  const manifest = {
    schema_version: 1,
    review: reviewManifest(data, reportPaths),
    findings: findings.map((finding) => ({
      id: finding.id,
      title: finding.title,
      priority: finding.priority,
      risk: finding.risk,
      status: finding.status,
      file: finding.file,
      line: finding.line,
    })),
    strategy: remediation.strategy,
    tasks,
  }
  const lines = [
    '# Code Review SDD — 审查修复计划',
    '',
    '> 本计划由 Code Review Leader 在发现可行动问题时生成，供 `code-review-sdd` skill 串行修复。它不自动授权扩大范围、修改公共契约、提交或推送。',
    '',
    '## 审查关联',
    '',
    '| 项目 | 值 |',
    '|---|---|',
    `| 合入建议 | ${mdEsc(data.verdict || '')} |`,
    `| 整体风险 | ${mdEsc(data.risk_level || '')} |`,
    `| 仓库 | ${mdEsc(data.repo || '')} |`,
    `| 分支 | ${mdEsc(data.branch || '')} |`,
    `| 范围 | ${mdEsc(data.scope || '')} |`,
    `| 范围类型 | ${mdEsc(data.scope_type || '')} |`,
    `| 审查 commit | ${mdEsc(data.reviewed_commit || '')} |`,
    `| 生成时间 | ${mdEsc(data.generated_at || '')} |`,
    `| HTML 报告 | ${mdEsc(reportPaths.html || '')} |`,
    `| Markdown 报告 | ${mdEsc(reportPaths.markdown || '')} |`,
    '',
    '## 执行规则与非目标',
    '',
    '- 按依赖顺序，再按 P0 → P1 → P2 → P3 串行推进；仅在任务明确标记 `parallel_safe: true` 时并行。',
    '- 每个任务开始前重新核对源证据与当前代码；保留用户已有工作树修改。',
    '- 只修复本计划中的已确认问题；不要顺手重构无关代码。',
    '- 公共 API / schema / 全局配置 / 锁文件变更，若任务未授权则暂停并请求确认。',
    '- 不要提交、推送、开 PR，除非用户另行要求。',
    '- 本计划不要求也不自动启动 `spec-harness`。',
    '',
    '## 修复策略',
    '',
    ...(remediation.strategy.length ? remediation.strategy.map((item) => `- ${item}`) : ['- 按优先级修复审查发现的问题，并补齐回归验证']),
    '',
    '## 任务顺序',
    '',
    '| 任务 | 优先级 | Findings | 依赖 | 人工确认 | 可并行 |',
    '|---|---|---|---|---|---|',
    ...tasks.map((task) => `| ${mdEsc(task.id)} ${mdEsc(task.title)} | ${mdEsc(task.priority)} | ${mdEsc(joinText(task.finding_ids))} | ${mdEsc(joinText(task.dependencies))} | ${task.human_confirmation_required ? '是' : '否'} | ${task.parallel_safe ? '是' : '否'} |`),
    '',
  ]
  for (const task of tasks) {
    lines.push(
      `## ${task.id} — ${task.title}`,
      '',
      `- 优先级/状态：${code(task.priority)} / ${code(task.status)}`,
      `- Findings：${joinText(task.finding_ids)}`,
      `- 依赖：${joinText(task.dependencies)}`,
      `- 可并行：${task.parallel_safe ? '是' : '否'}`,
      `- 需要人工确认：${task.human_confirmation_required ? '是' : '否'}`,
      '',
      '### 目标',
      '',
      task.objective || '无',
      '',
      '### 理由',
      '',
      task.rationale || '无',
      '',
      '### 允许范围',
      '',
      ...(task.scope.allowed_paths.length ? task.scope.allowed_paths.map((item) => `- ${code(item)}`) : ['- 无（修复前需根据证据补齐允许路径）']),
      '',
      '### 排除范围',
      '',
      ...(task.scope.excluded_paths.length ? task.scope.excluded_paths.map((item) => `- ${code(item)}`) : ['- 无']),
      '',
      '### 源证据',
      '',
      ...(task.source_evidence.length ? task.source_evidence.map((item) => `- ${code(item)}`) : ['- 无']),
      '',
      '### 执行步骤',
      '',
      ...(task.steps.length ? task.steps.map((item, index) => `${index + 1}. ${item}`) : ['1. 根据审查建议完成根因修复']),
      '',
      '### 验收标准',
      '',
      ...(task.acceptance_criteria.length ? task.acceptance_criteria.map((item) => `- [ ] ${item}`) : ['- [ ] 原问题不再复现']),
      '',
      '### 验证命令',
      '',
      ...(task.tests.length ? task.tests.map((item) => `- ${code(item)}`) : ['- （未指定；按改动模块运行对应测试/typecheck）']),
      '',
      '### 回滚/撤销',
      '',
      task.rollback,
      '',
      '### 交付物',
      '',
      ...(task.deliverables.length ? task.deliverables.map((item) => `- ${item}`) : ['- 代码修复', '- 验证证据']),
      '',
    )
  }
  const traceability = findings.map((finding) => ({
    finding,
    tasks: tasks.filter((task) => task.finding_ids.includes(finding.id)),
  }))
  lines.push(
    '## Finding—任务追踪矩阵',
    '',
    '| Finding | 状态 | 优先级 | 风险 | 任务 |',
    '|---|---|---|---|---|',
    ...traceability.map(({ finding, tasks: linked }) => `| ${mdEsc(finding.id)} ${mdEsc(finding.title)} | ${mdEsc(finding.status)} | ${mdEsc(finding.priority)} | ${mdEsc(finding.risk)} | ${mdEsc(linked.map((task) => task.id).join('、') || '无')} |`),
    '',
    '## 最终累计验证门禁',
    '',
    '- [ ] 所有 P0/P1 任务已完成，或由用户明确接受残留风险',
    '- [ ] 每个已修复 finding 在当前证据下不再复现',
    '- [ ] 相关单元/集成/类型检查通过',
    '- [ ] 改动未超出各任务允许范围',
    '- [ ] 建议重新运行 `code-review-leader` 做复核',
    '',
    '## Agent 执行清单',
    '',
    '```code-review-sdd-manifest',
    JSON.stringify(manifest, null, 2),
    '```',
    '',
  )
  return `${lines.join('\n').trimEnd()}\n`
}

const noActionSddDocument = (data, reportPaths = {}) => {
  const manifest = {
    schema_version: 1,
    no_action: true,
    review: reviewManifest(data, reportPaths),
    findings: [],
    strategy: [],
    tasks: [],
  }
  return [
    '# Code Review SDD — 无待修复事项',
    '',
    '> 最新 Code Review Leader 审查未发现可行动问题。此 canonical marker 会使旧修复计划失效，不得回退执行历史 timestamped SDD。',
    '',
    `- 合入建议：${data.verdict || ''}`,
    `- 整体风险：${data.risk_level || ''}`,
    `- HTML 报告：${reportPaths.html || ''}`,
    `- Markdown 报告：${reportPaths.markdown || ''}`,
    '',
    '```code-review-sdd-manifest',
    JSON.stringify(manifest, null, 2),
    '```',
    '',
  ].join('\n')
}

const writeFileAtomic = (target, content) => {
  fs.mkdirSync(path.dirname(target), { recursive: true })
  const temporary = path.join(
    path.dirname(target),
    `.${path.basename(target)}.${process.pid}.${Date.now()}.tmp`,
  )
  try {
    fs.writeFileSync(temporary, content, 'utf8')
    fs.renameSync(temporary, target)
  } finally {
    if (fs.existsSync(temporary)) fs.rmSync(temporary, { force: true })
  }
}

const htmlDocument = (data) => {
  const title = esc(data.title || DEFAULT_TITLE)
  const metadata = [
    data.generated_at ? `报告时间：${esc(data.generated_at)}` : '',
    data.repo ? `仓库：${esc(data.repo)}` : '',
    data.branch ? `分支：${esc(data.branch)}` : '',
    data.scope ? `范围：${esc(data.scope)}` : '',
  ].filter(Boolean).join(' | ')
  const verdict = esc(data.verdict || '')
  const risk = esc(data.risk_level || '')
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>${title}</title>
<style>
:root {
  --kd-primary:#4E83FD; --kd-success:#73D13D; --kd-danger:#FF7875; --kd-warning:#FFC069;
  --kd-info:#69C0FF; --bg:#f8f9fa; --card-bg:#fff; --text:#1d2129;
  --text-secondary:#4e5969; --border:#e5e6eb;
}
* { margin:0; padding:0; box-sizing:border-box; }
body {
  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif;
  background:var(--bg); color:var(--text); line-height:1.6; padding:24px; max-width:1440px; margin:0 auto;
}
h1 { font-size:24px; margin-bottom:8px; text-align:center; }
.meta { color:var(--text-secondary); font-size:13px; margin:-4px 0 16px; text-align:center; }
.summary-bar { background:#D6E4FF; padding:20px 24px; border-radius:8px; margin-bottom:24px; border:1px solid #ADC6FF; }
.summary-bar h2 { font-size:15px; margin-bottom:8px; font-weight:700; }
.summary-item { margin:10px 0; padding:10px 14px; background:rgba(255,255,255,.6); border-radius:6px; border-left:3px solid var(--kd-primary); }
.summary-item--risk { border-left-color:var(--kd-danger); background:rgba(255,241,240,.5); }
.summary-item-label { font-weight:700; font-size:14px; margin-bottom:4px; }
.summary-item-content { font-size:14px; }
.section { background:var(--card-bg); border:1px solid var(--border); border-radius:8px; margin-bottom:16px; overflow:hidden; }
.section-header { display:flex; align-items:center; padding:16px 20px; cursor:pointer; user-select:none; border-bottom:1px solid var(--border); }
.section-header:hover { background:#f2f3f5; }
.section-header h2 { font-size:16px; flex:1; margin:0; }
.toggle { font-size:12px; color:var(--text-secondary); transition:transform .2s; }
.toggle.collapsed { transform:rotate(-90deg); }
.section-body { padding:20px; overflow-x:auto; }
.section-body.hidden { display:none; }
.mermaid-container { background:#fafbfc; border:1px solid var(--border); border-radius:6px; padding:16px; margin:12px 0; text-align:center; position:relative; }
.mermaid-container svg { max-width:100%; }
.diagram-note { color:var(--text-secondary); font-size:12px; margin:0 0 10px; }
.diagram-note--fallback { color:#D48806; text-align:left; }
.offline-flowchart { width:100%; min-height:120px; }
.offline-flowchart__edges line { stroke:#86909c; stroke-width:1.6; }
.offline-flowchart__edges marker path { fill:#86909c; }
.offline-flowchart__nodes rect { fill:#fff; stroke:#4E83FD; stroke-width:1.5; }
.offline-flowchart__nodes text { fill:#1d2129; font-size:13px; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif; }
.mermaid-source-details { margin-top:10px; text-align:left; }
.mermaid-source-details summary { cursor:pointer; color:var(--text-secondary); font-size:12px; }
.mermaid-source { margin-top:8px; padding:12px; overflow:auto; background:#f2f3f5; border-radius:4px; font:12px/1.5 Consolas,monospace; white-space:pre; }
table { width:100%; border-collapse:collapse; margin:12px 0; font-size:14px; }
th,td { padding:10px 12px; text-align:left; border-bottom:1px solid var(--border); word-break:break-word; overflow-wrap:break-word; }
td:first-child { max-width:280px; word-break:break-all; }
th { background:#f7f8fa; font-weight:600; user-select:none; white-space:nowrap; }
th[data-sortable] { cursor:pointer; }
tr:hover td { background:#f7f8fa; }
.risk-high { color:var(--kd-danger); font-weight:700; }
.risk-medium { color:#D48806; font-weight:700; }
.risk-low { color:#389E0D; }
.badge { display:inline-block; padding:2px 8px; border-radius:4px; font-size:12px; font-weight:500; }
.badge-violation { background:#FFF1F0; color:var(--kd-danger); }
.badge-attention { background:#FFF7E6; color:#D48806; }
.badge-compliant { background:#F6FFED; color:#389E0D; }
p { margin:8px 0; } ul,ol { margin:8px 0 8px 20px; } li { margin:4px 0; }
code { background:#f2f3f5; padding:2px 6px; border-radius:3px; font-size:13px; font-family:Consolas,monospace; }
.nav-toc { position:fixed; left:16px; top:24px; width:210px; background:var(--card-bg); border:1px solid var(--border); border-radius:8px; padding:16px 14px; font-size:14px; z-index:100; max-height:calc(100vh - 48px); overflow-y:auto; }
.nav-toc-title { font-weight:600; font-size:12px; color:var(--text-secondary); text-transform:uppercase; letter-spacing:.5px; margin-bottom:8px; padding-bottom:6px; border-bottom:1px solid var(--border); }
.nav-toc a { display:block; padding:6px 0 6px 10px; color:var(--text-secondary); text-decoration:none; font-size:13px; line-height:1.5; border-left:2px solid transparent; }
.nav-toc a:hover,.nav-toc a.active { color:var(--kd-primary); border-left-color:var(--kd-primary); font-weight:600; }
.back-to-top { position:fixed; right:24px; bottom:24px; width:40px; height:40px; border-radius:50%; background:var(--kd-primary); color:#fff; border:0; font-size:18px; cursor:pointer; display:flex; align-items:center; justify-content:center; opacity:0; transition:opacity .2s; z-index:100; }
.back-to-top.visible { opacity:1; }
@media (max-width:1440px) { .nav-toc { display:none; } body { padding-left:24px; } }
@media (min-width:1441px) { body { padding-left:250px; } }
</style>
</head>
<body>
<nav class="nav-toc" id="nav-toc"><div class="nav-toc-title">目录</div></nav>
<div id="report-content">
<h1>${title}</h1>
<p class="meta">${metadata}</p>
<div class="summary-item summary-item--risk"><div class="summary-item-label">结论</div><div class="summary-item-content"><p>合入建议：${verdict || '未填写'}；整体风险：${risk || '未填写'}</p></div></div>
${renderSummary(data.summary || [])}
${renderSections(data.sections || [])}
</div>
<button class="back-to-top" id="back-to-top" title="回到顶部">↑</button>
<script>
document.querySelectorAll('.section-header').forEach(function(header, idx) {
  const body = header.nextElementSibling;
  const toggle = header.querySelector('.toggle');
  const id = 'section-' + (idx + 1);
  header.id = id;
  header.addEventListener('click', function() {
    body.classList.toggle('hidden');
    toggle.classList.toggle('collapsed');
  });
});
(function initToc() {
  const toc = document.getElementById('nav-toc');
  document.querySelectorAll('.section-header h2').forEach(function(h2) {
    const a = document.createElement('a');
    a.href = '#' + h2.parentElement.id;
    a.textContent = h2.textContent;
    toc.appendChild(a);
  });
})();
window.addEventListener('scroll', function() {
  document.getElementById('back-to-top').classList.toggle('visible', window.scrollY > 300);
});
document.getElementById('back-to-top').addEventListener('click', function() { window.scrollTo({top:0, behavior:'smooth'}); });
document.querySelectorAll('th[data-sortable]').forEach(function(th) {
  th.addEventListener('click', function() {
    const table = th.closest('table');
    const tbody = table.querySelector('tbody');
    const idx = Array.from(th.parentElement.children).indexOf(th);
    const asc = th.dataset.asc !== 'true';
    Array.from(tbody.querySelectorAll('tr')).sort(function(a,b) {
      return asc
        ? a.children[idx].textContent.localeCompare(b.children[idx].textContent, 'zh-CN')
        : b.children[idx].textContent.localeCompare(a.children[idx].textContent, 'zh-CN');
    }).forEach(function(row) { tbody.appendChild(row); });
    th.dataset.asc = String(asc);
  });
});
</script>
</body>
</html>
`
}

const markdownTable = (headers, rows) => {
  if (!headers?.length) return ''
  const header = `| ${headers.map(mdEsc).join(' | ')} |`
  const divider = `| ${headers.map(() => '---').join(' | ')} |`
  const body = (rows || []).map((row) => `| ${row.map((cell) => {
    if (cell && typeof cell === 'object' && 'html' in cell) return mdEsc(stripHtml(cell.html))
    return mdEsc(cell)
  }).join(' | ')} |`)
  return [header, divider, ...body].join('\n')
}

const markdownFindings = (items) => {
  if (!items?.length) return '未发现阻塞性问题。\n'
  const rows = items.map((item) => {
    let loc = String(item.file || '')
    if (item.line) loc += `:${item.line}`
    return [item.priority || 'P3', item.title || '问题', loc, item.risk || 'medium', item.detail || '', item.recommendation || '']
  })
  return markdownTable(['优先级', '问题', '位置', '风险', '影响', '建议'], rows)
}

const markdownBlock = (block) => {
  const kind = block.type || 'paragraph'
  if (kind === 'html') return stripHtml(block.html || '')
  if (kind === 'paragraph') return String(block.text || '')
  if (kind === 'list') {
    const items = block.items || []
    return block.ordered
      ? items.map((item, idx) => `${idx + 1}. ${item}`).join('\n')
      : items.map((item) => `- ${item}`).join('\n')
  }
  if (kind === 'table') return markdownTable(block.headers || [], block.rows || [])
  if (kind === 'mermaid') return `\`\`\`mermaid\n${String(block.source || '')}\n\`\`\``
  if (kind === 'findings') return markdownFindings(block.items || [])
  return JSON.stringify(block)
}

const markdownDocument = (data) => {
  const lines = [`# ${data.title || DEFAULT_TITLE}`, '']
  const metadata = [
    ['报告时间', data.generated_at],
    ['仓库', data.repo],
    ['分支', data.branch],
    ['范围', data.scope],
    ['范围类型', data.scope_type],
    ['审查 commit', data.reviewed_commit],
    ['合入建议', data.verdict],
    ['整体风险', data.risk_level],
  ]
  for (const [label, value] of metadata) {
    if (value) lines.push(`- **${label}**：${value}`)
  }
  lines.push('', '## 整体说明', '')
  for (const item of data.summary || []) {
    lines.push(`### ${stripHtml(item.label || '摘要')}`)
    lines.push(item.content_html ? stripHtml(item.content_html) : String(item.text || ''))
    lines.push('')
  }
  for (const section of data.sections || []) {
    lines.push(`## ${stripHtml(section.title || '未命名章节')}`, '')
    if (section.content_html) lines.push(stripHtml(section.content_html), '')
    if (section.text) lines.push(String(section.text), '')
    if (section.mermaid) lines.push(`\`\`\`mermaid\n${String(section.mermaid)}\n\`\`\``, '')
    for (const block of section.blocks || []) {
      const rendered = markdownBlock(block).trim()
      if (rendered) lines.push(rendered, '')
    }
  }
  return `${lines.join('\n').trimEnd()}\n`
}

const parseArgs = (argv) => {
  const args = { sample: false, noSdd: false }
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--sample') args.sample = true
    else if (arg === '--no-sdd') args.noSdd = true
    else if (arg === '--input') args.input = argv[++i]
    else if (arg === '--output') args.output = argv[++i]
    else if (arg === '--markdown-output') args.markdownOutput = argv[++i]
    else if (arg === '--sdd-output') args.sddOutput = argv[++i]
    else if (arg === '--help' || arg === '-h') {
      console.log('Usage: node render_review_report.mjs --input report.json [--output report.html] [--markdown-output report.md] [--sdd-output code-review-sdd.md]')
      console.log('       node render_review_report.mjs --sample [--output sample.html] [--markdown-output sample.md] [--sdd-output sample-sdd.md]')
      console.log('Flags: --no-sdd  skip writing the repair SDD even when findings exist')
      process.exit(0)
    } else {
      throw new Error(`Unknown argument: ${arg}`)
    }
  }
  if (!args.sample && !args.input) throw new Error('--input is required unless --sample is used')
  return args
}

const args = parseArgs(process.argv.slice(2))
const data = args.sample ? samplePayload() : JSON.parse(fs.readFileSync(args.input, 'utf8'))
const timestamp = new Date().toISOString().replace(/[-:]/g, '').replace(/\..+$/, '').replace('T', '-')
args.output ||= path.join(DEFAULT_OUTPUT_DIR, `code-review-leader-${timestamp}.html`)
args.markdownOutput ||= args.output.replace(/\.html?$/i, '.md')
const html = htmlDocument(data)
if (/<(?:script|link|img|source|video|audio|iframe)\b[^>]*(?:src|href)=["']https?:\/\//i.test(html)) {
  throw new Error('self-contained HTML must not reference external resources')
}
fs.mkdirSync(path.dirname(args.output), { recursive: true })
fs.writeFileSync(args.output, html, 'utf8')
console.log(`html_report: ${args.output}`)
fs.mkdirSync(path.dirname(args.markdownOutput), { recursive: true })
fs.writeFileSync(args.markdownOutput, markdownDocument(data), 'utf8')
console.log(`markdown_report: ${args.markdownOutput}`)

const remediation = buildRemediation(data)
if (!args.noSdd && remediation) {
  const archiveSdd = args.output.replace(/\.html?$/i, '-sdd.md')
  args.sddOutput ||= path.join(DEFAULT_OUTPUT_DIR, 'code-review-sdd.md')
  const reportPaths = { html: args.output, markdown: args.markdownOutput }
  const sddBody = sddDocument(data, remediation, reportPaths)
  for (const target of [...new Set([args.sddOutput, archiveSdd])]) {
    writeFileAtomic(target, sddBody)
    console.log(`code_review_sdd: ${target}`)
  }
  console.log(`code_review_sdd_tasks: ${remediation.tasks.length}`)
} else if (!args.noSdd) {
  args.sddOutput ||= path.join(DEFAULT_OUTPUT_DIR, 'code-review-sdd.md')
  const reportPaths = { html: args.output, markdown: args.markdownOutput }
  writeFileAtomic(args.sddOutput, noActionSddDocument(data, reportPaths))
  console.log(`code_review_sdd: ${args.sddOutput}`)
  console.log('code_review_sdd_tasks: 0 (no_action)')
}
