#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { asList, parseCli, SUCCESS_STATES } from './plan_utils.mjs'
import { stateSummary, validateState } from './state_utils.mjs'

const md = (value) => String(value ?? '').replaceAll('|', '\\|').replaceAll('\n', '<br>')
const table = (headers, rows) => [
  `| ${headers.map(md).join(' | ')} |`,
  `| ${headers.map(() => '---').join(' | ')} |`,
  ...rows.map((row) => `| ${row.map(md).join(' | ')} |`),
].join('\n')

const atomicWriteText = (file, value) => {
  const temp = `${file}.${process.pid}.tmp`
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(temp, value, 'utf8')
  fs.renameSync(temp, file)
}

try {
  const args = parseCli(process.argv, new Set(['help', 'force']))
  if (args.help || !args.state || !args.output) {
    process.stdout.write('Usage: render_fix_summary.mjs --state <fix-state.json> --output <summary.md> [--force]\n')
    process.exitCode = args.help ? 0 : 2
  } else {
    const statePath = path.resolve(args.state)
    const output = path.resolve(args.output)
    if (fs.existsSync(output) && !args.force) throw new Error(`refusing to overwrite existing summary: ${output}`)
    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'))
    const schedule = validateState(state, statePath)
    const unresolved = state.tasks.filter((task) => !SUCCESS_STATES.has(task.status))
    const lines = [
      '# Smart Recruit 知识库修复执行总结', '',
      table(['字段', '值'], [
        ['Run ID', state.run_id], ['修复计划', state.plan.path], ['计划哈希', state.plan.sha256],
        ['审计提交', state.plan.audit_commit], ['审计结论', state.plan.audit_verdict],
        ['执行分支', state.repository.branch], ['初始提交', state.repository.initial_head],
        ['选择模式', `${state.selection.mode}: ${state.selection.values.join(', ') || 'all'}`],
        ['最终执行状态', state.run_status], ['生成时间', new Date().toISOString()],
      ]), '',
      '## 任务结果', '',
      table(['顺序', '任务', '优先级', '目标/级别', '状态', '关联发现', '确认门禁', '修改路径', '检查'], state.tasks.map((task) => [
        task.order + 1, task.id, task.priority, `${task.resolution_target}/${task.knowledge_level}`, task.status,
        asList(task.finding_ids).join(', '), task.effective_confirmation_reasons.join(', ') || '无',
        asList(task.modified_paths).join(', ') || '无', asList(task.checks).map((check) => `${check.name}:${check.status}`).join(', ') || '无',
      ])), '',
      '## 调度与未完成项', '',
      `- 下一任务：${schedule.next_task || '无'}`,
      `- Ready：${schedule.ready_tasks.join(', ') || '无'}`,
      `- 等待确认：${schedule.awaiting_confirmation.join(', ') || '无'}`,
      `- 依赖阻断：${schedule.dependency_blocked.join(', ') || '无'}`,
      `- 未解决任务：${unresolved.map((task) => `${task.id}(${task.status})`).join(', ') || '无'}`, '',
      '## 预先存在的工作树重叠', '',
      ...state.tasks.map((task) => `- ${task.id}：allowed=${task.preexisting_allowed_overlap.join(', ') || '无'}；excluded=${task.preexisting_excluded_overlap.join(', ') || '无'}`), '',
      '## 结论', '', `**${state.run_status}**`, '',
    ]
    if (state.run_status === 'REMEDIATION_COMPLETE_PENDING_REAUDIT') {
      lines.push('全部计划任务达到成功终态。使用 `$knowledge-current-state-audit` 对当前工作树重新执行完整审计；新审计结论才是知识现状的权威结果。', '')
    } else lines.push('修复尚未全部完成。解决上面的确认、依赖、范围或验证条件后，使用 `$knowledge-current-state-fix` 恢复本次运行。', '')
    atomicWriteText(output, `${lines.join('\n')}\n`)
    process.stdout.write(`${output}\n`)
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`)
  process.exitCode = 1
}
