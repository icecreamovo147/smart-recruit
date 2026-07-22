#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { asList, parseCli, SUCCESS_STATES } from './plan_utils.mjs'
import { validateState } from './state_utils.mjs'

const md = (value) => String(value ?? '').replaceAll('|', '\\|').replaceAll('\n', '<br>')
const table = (headers, rows) => [
  `| ${headers.map(md).join(' | ')} |`,
  `| ${headers.map(() => '---').join(' | ')} |`,
  ...rows.map((row) => `| ${row.map(md).join(' | ')} |`),
].join('\n')

try {
  const args = parseCli(process.argv, new Set(['help']))
  if (args.help || !args.state || !args.output) {
    process.stdout.write('Usage: render_fix_summary.mjs --state <fix-state.json> --output <summary.md>\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const output = path.resolve(args.output)
    if (fs.existsSync(output)) throw new Error(`refusing to overwrite existing summary: ${output}`)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    const schedule = validateState(state)
    const unresolved = state.tasks.filter((task) => !SUCCESS_STATES.has(task.status))
    const lines = [
      '# Smart Recruit 安全修复执行总结', '',
      table(['字段', '值'], [
        ['Run ID', state.run_id], ['修复计划', state.plan.path], ['计划哈希', state.plan.sha256],
        ['审查提交', state.plan.audit_commit], ['执行分支', state.repository.branch],
        ['初始提交', state.repository.initial_head], ['最终执行状态', state.run_status],
        ['生成时间', new Date().toISOString()],
      ]), '',
      '## 任务结果', '',
      table(['顺序', '任务', '优先级', '状态', '关联发现', '修改路径', '检查'], state.tasks.map((task) => [
        task.order + 1, task.id, task.priority, task.status, asList(task.finding_ids).join(', '),
        asList(task.modified_paths).join(', '), asList(task.checks).map((check) => `${check.name}:${check.status}`).join(', '),
      ])), '',
      '## 调度与未完成项', '',
      `- 下一任务：${schedule.next_task || '无'}`,
      `- Ready：${schedule.ready_tasks.join(', ') || '无'}`,
      `- 依赖阻断：${schedule.dependency_blocked.join(', ') || '无'}`,
      `- 未解决任务：${unresolved.map((task) => `${task.id}(${task.status})`).join(', ') || '无'}`, '',
      '## 结论', '',
      `**${state.run_status}**`, '',
    ]
    if (state.run_status === 'REMEDIATION_COMPLETE_PENDING_REAUDIT') {
      lines.push('所有选定任务已达到成功终态。请使用 `$prelaunch-security-audit` 对修复后的当前提交/工作树重新执行完整上线前安全审查。', '')
    } else lines.push('修复尚未全部完成。解决上述阻断条件后，使用 `$prelaunch-security-fix` 恢复本次运行。', '')
    fs.mkdirSync(path.dirname(output), { recursive: true })
    fs.writeFileSync(output, `${lines.join('\n')}\n`, { encoding: 'utf8', flag: 'wx' })
    process.stdout.write(`${output}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
