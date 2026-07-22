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

const renderMermaid = (source) => [
  '<div class="mermaid-container">',
  `<pre class="mermaid-source" style="display:none">${esc(source)}</pre>`,
  '<p>加载图表中...</p>',
  '</div>',
].join('')

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
})

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
<script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
<script>
mermaid.initialize({ startOnLoad:false, securityLevel:'loose', theme:'default' });
async function renderAll() {
  const els = document.querySelectorAll('.mermaid-source');
  for (let i = 0; i < els.length; i++) {
    const src = els[i].textContent || '';
    const container = els[i].closest('.mermaid-container');
    try {
      const out = await mermaid.render('mermaid-' + i, src);
      container.innerHTML = out.svg;
      container.classList.add('rendered');
    } catch (e) {
      container.innerHTML = '<pre>' + src.replace(/[&<>]/g, s => ({'&':'&amp;','<':'&lt;','>':'&gt;'}[s])) + '</pre>';
    }
  }
}
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
renderAll();
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
  const args = { sample: false }
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--sample') args.sample = true
    else if (arg === '--input') args.input = argv[++i]
    else if (arg === '--output') args.output = argv[++i]
    else if (arg === '--markdown-output') args.markdownOutput = argv[++i]
    else if (arg === '--help' || arg === '-h') {
      console.log('Usage: node render_review_report.mjs --input report.json [--output report.html] [--markdown-output report.md]')
      console.log('       node render_review_report.mjs --sample [--output sample.html] [--markdown-output sample.md]')
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
fs.mkdirSync(path.dirname(args.output), { recursive: true })
fs.writeFileSync(args.output, htmlDocument(data), 'utf8')
console.log(args.output)
fs.mkdirSync(path.dirname(args.markdownOutput), { recursive: true })
fs.writeFileSync(args.markdownOutput, markdownDocument(data), 'utf8')
console.log(args.markdownOutput)
