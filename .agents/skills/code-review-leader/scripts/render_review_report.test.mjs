import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'render_review_report.mjs')
const parserPath = path.resolve(path.dirname(scriptPath), '../../code-review-sdd/scripts/parse_code_review_sdd.mjs')

const render = (sources) => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'code-review-leader-renderer-'))
  const input = path.join(directory, 'report.json')
  const output = path.join(directory, 'report.html')
  const markdown = path.join(directory, 'report.md')
  const payload = {
    title: 'Renderer test',
    sections: [{
      title: 'Diagrams',
      blocks: sources.map((source) => ({ type: 'mermaid', source })),
    }],
  }
  fs.writeFileSync(input, JSON.stringify(payload))
  const result = spawnSync(process.execPath, [
    scriptPath,
    '--input', input,
    '--output', output,
    '--markdown-output', markdown,
    '--no-sdd',
  ], { encoding: 'utf8' })
  const html = result.status === 0 ? fs.readFileSync(output, 'utf8') : ''
  fs.rmSync(directory, { recursive: true, force: true })
  return { ...result, html }
}

const renderSdd = (payload, staleCanonical = '') => {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'code-review-leader-sdd-'))
  const input = path.join(directory, 'report.json')
  const output = path.join(directory, 'report.html')
  const markdown = path.join(directory, 'report.md')
  const sdd = path.join(directory, 'code-review-sdd.md')
  fs.writeFileSync(input, JSON.stringify(payload))
  if (staleCanonical) fs.writeFileSync(sdd, staleCanonical)
  const result = spawnSync(process.execPath, [
    scriptPath,
    '--input', input,
    '--output', output,
    '--markdown-output', markdown,
    '--sdd-output', sdd,
  ], { encoding: 'utf8' })
  const sddBody = result.status === 0 ? fs.readFileSync(sdd, 'utf8') : ''
  const parsed = result.status === 0
    ? spawnSync(process.execPath, [parserPath, '--input', sdd], { encoding: 'utf8' })
    : null
  fs.rmSync(directory, { recursive: true, force: true })
  return { ...result, sddBody, parsed }
}

test('preserves explicit labels across later bare branch and merge references', () => {
  const result = render([
    'flowchart TD\nA[业务入口] --> B[用户确认]\nA --> C[用户拒绝]\nB --> D[流程结束]\nC --> D',
  ])
  assert.equal(result.status, 0, result.stderr)
  for (const label of ['业务入口', '用户确认', '用户拒绝', '流程结束']) {
    assert.match(result.html, new RegExp(`>${label}</text>`))
  }
  for (const id of ['A', 'B', 'C', 'D']) {
    assert.doesNotMatch(result.html, new RegExp(`>${id}</text>`))
  }
})

test('allows a later explicit declaration to complete an earlier bare node', () => {
  const result = render([
    'flowchart LR\nA --> B[第一步]\nA[补全入口] --> C[第二步]',
  ])
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.html, />补全入口<\/text>/)
  assert.doesNotMatch(result.html, />A<\/text>/)
})

test('keeps unsupported Mermaid visible without external resources', () => {
  const result = render([
    'sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hello',
  ])
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.html, /当前 Mermaid 语法超出离线渲染子集/)
  assert.match(result.html, /sequenceDiagram/)
  assert.doesNotMatch(result.html, /<(?:script|link|img|source|video|audio|iframe)\b[^>]*(?:src|href)=["']https?:\/\//i)
})

test('clean review atomically replaces a stale canonical plan with no-action state', () => {
  const result = renderSdd({
    title: 'Clean review',
    verdict: '可以合入',
    risk_level: '低风险',
    sections: [],
  }, 'STALE-PLAN\n')
  assert.equal(result.status, 0, result.stderr)
  assert.doesNotMatch(result.sddBody, /STALE-PLAN/)
  assert.match(result.sddBody, /"no_action": true/)
  assert.equal(result.parsed?.status, 0, result.parsed?.stderr)
  const parsed = JSON.parse(result.parsed.stdout)
  assert.equal(parsed.no_action, true)
  assert.equal(parsed.task_count, 0)
})

test('actionable review still generates and parses repair tasks', () => {
  const finding = {
    id: 'F-001',
    priority: 'P1',
    title: 'Actionable finding',
    file: 'src/a.ts',
    line: 1,
    risk: 'high',
    detail: 'Failure evidence',
    recommendation: 'Repair it',
  }
  const result = renderSdd({
    title: 'Actionable review',
    verdict: '需修复后合入',
    risk_level: '高风险',
    sections: [{
      title: 'Findings',
      blocks: [{ type: 'findings', items: [finding] }],
    }],
    remediation: {
      strategy: ['repair'],
      tasks: [{
        id: 'CR-001',
        finding_ids: ['F-001'],
        priority: 'P1',
        title: 'Repair finding',
        objective: 'Failure no longer reproduces',
        rationale: 'Failure evidence',
        status: 'PENDING',
        dependencies: [],
        parallel_safe: false,
        human_confirmation_required: false,
        scope: { allowed_paths: ['src/a.ts'], excluded_paths: [] },
        source_evidence: ['src/a.ts:1'],
        steps: ['repair'],
        acceptance_criteria: ['passes'],
        tests: [],
        rollback: 'restore file',
        deliverables: ['repair'],
      }],
    },
  })
  assert.equal(result.status, 0, result.stderr)
  assert.doesNotMatch(result.sddBody, /"no_action": true/)
  assert.equal(result.parsed?.status, 0, result.parsed?.stderr)
  const parsed = JSON.parse(result.parsed.stdout)
  assert.equal(parsed.no_action, false)
  assert.equal(parsed.task_count, 1)
  assert.equal(parsed.tasks[0].id, 'CR-001')
})
