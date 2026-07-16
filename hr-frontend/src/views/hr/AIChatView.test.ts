/**
 * Focused recovery tests for AIChatView refresh restore helpers.
 * Full view mount is heavy (router, many APIs); composable tests cover reconnect/cancel.
 * Here we exercise message-slot seeding rules used by restoreActiveRunForSession.
 */
import { describe, expect, it } from 'vitest'
import {
  buildApplicationAnalysisMessage,
  buildApplicationAnalysisRunRequest,
  resolveApplicationAnalysisMessage,
} from './AIChatView.vue'
import { sanitizeAssistantProcessText } from '@/utils/hrAssistantProcess'

interface MessageItem {
  role: string
  content: string
  pending?: boolean
  failed?: boolean
  processContent?: string
  process_content?: string
  waitingText?: string
  model_name?: string
  agentSkillSelection?: unknown
}

/**
 * Mirror of ensureRestoreAssistantSlot in AIChatView.vue — kept in sync for unit coverage.
 * If the view helper drifts, update both places (or extract later when scope allows).
 */
function ensureRestoreAssistantSlot(
  messages: MessageItem[],
  session: { application_id?: number },
  run: { assistantText: string; processText: string; modelName: string; isTerminal: boolean },
): number {
  const lastIndex = messages.length - 1
  const last = lastIndex >= 0 ? messages[lastIndex] : null
  if (
    last?.role === 'assistant' &&
    !last.agentSkillSelection &&
    !last.failed &&
    (
      last.pending ||
      !last.content ||
      (run.assistantText && run.assistantText.startsWith(last.content || ''))
    )
  ) {
    messages[lastIndex] = {
      ...last,
      content: run.assistantText || last.content || '',
      processContent: run.processText || last.processContent || last.process_content || '',
      pending: !run.isTerminal && !(run.assistantText && !last.pending && last.content === run.assistantText),
      waitingText: last.waitingText || (session.application_id ? '分析中' : '响应中'),
      model_name: run.modelName || last.model_name,
    }
    return lastIndex
  }

  messages.push({
    role: 'assistant',
    content: run.assistantText || '',
    processContent: run.processText || '',
    pending: true,
    waitingText: session.application_id ? '分析中' : '响应中',
    ...(run.modelName ? { model_name: run.modelName } : {}),
  })
  return messages.length - 1
}

describe('AIChatView restore assistant slot seeding', () => {
  it('reuses pending assistant bubble and seeds snapshot text without duplication', () => {
    const messages: MessageItem[] = [
      { role: 'user', content: 'hello' },
      { role: 'assistant', content: '', pending: true, waitingText: '响应中' },
    ]
    const idx = ensureRestoreAssistantSlot(messages, {}, {
      assistantText: 'partial reply',
      processText: 'tool:search',
      modelName: 'gpt-test',
      isTerminal: false,
    })
    expect(idx).toBe(1)
    expect(messages).toHaveLength(2)
    expect(messages[1].content).toBe('partial reply')
    expect(messages[1].processContent).toBe('tool:search')
    expect(messages[1].model_name).toBe('gpt-test')
    expect(messages[1].pending).toBe(true)
  })

  it('appends a new assistant bubble when history ends with a completed answer', () => {
    const messages: MessageItem[] = [
      { role: 'user', content: 'q1' },
      { role: 'assistant', content: 'done already', pending: false },
      { role: 'user', content: 'q2' },
    ]
    const idx = ensureRestoreAssistantSlot(messages, { application_id: 9 }, {
      assistantText: 'new run text',
      processText: '',
      modelName: '',
      isTerminal: false,
    })
    expect(idx).toBe(3)
    expect(messages).toHaveLength(4)
    expect(messages[3].content).toBe('new run text')
    expect(messages[3].waitingText).toBe('分析中')
    expect(messages[3].pending).toBe(true)
  })

  it('extends last assistant when snapshot is a prefix continuation of stored content', () => {
    const messages: MessageItem[] = [
      { role: 'assistant', content: 'Hel', pending: false },
    ]
    const idx = ensureRestoreAssistantSlot(messages, {}, {
      assistantText: 'Hello',
      processText: '',
      modelName: '',
      isTerminal: false,
    })
    expect(idx).toBe(0)
    expect(messages[0].content).toBe('Hello')
  })
})

describe('AIChatView assistant process formatting', () => {
  it('renders persisted ADK runtime JSON as display summary text', () => {
    const text = sanitizeAssistantProcessText(JSON.stringify({
      runtime: 'adk',
      display_summary: ['已完成：读取当前岗位列表。'],
      fallback_used: false,
      tool_count: 1,
      tool_results: [{ tool_name: 'get_job_list', status: 'success' }],
    }))

    expect(text).toBe('已完成：读取当前岗位列表。')
    expect(text).not.toContain('"runtime"')
    expect(text).not.toContain('tool_results')
  })
})

describe('AIChatView application analysis message', () => {
  it('builds a non-empty planner-recognizable message for route fallback', () => {
    const message = buildApplicationAnalysisMessage('张三', '后端工程师')
    expect(message).toContain('张三')
    expect(message).toContain('后端工程师')
    expect(message).toContain('匹配度')
    expect(message.trim()).not.toBe('')
  })

  it('uses stable placeholders when route labels are missing', () => {
    expect(buildApplicationAnalysisMessage()).toContain('该候选人')
    expect(buildApplicationAnalysisMessage()).toContain('该岗位')
  })

  it('route entry prefers the backend canonical message in the submitted payload', () => {
    const message = resolveApplicationAnalysisMessage(
      [{ role: 'user', content: '  后端返回的匹配评估指令  ' }],
      '张三',
      '后端工程师',
    )
    const payload = buildApplicationAnalysisRunRequest({
      sessionId: 11,
      message,
      applicationId: 22,
      clientRequestId: 'route-request',
      modelId: 33,
      skillCapabilityKeys: ['candidate.match'],
    })
    expect(payload.message).toBe('后端返回的匹配评估指令')
    expect(payload.action_type).toBe('analyze_application')
    expect(payload.application_id).toBe(22)
    expect(payload.skill_capability_keys).toEqual(['candidate.match'])
  })

  it('in-chat analysis submits the planner-recognizable fallback for legacy empty messages', () => {
    const message = resolveApplicationAnalysisMessage(
      [{ role: 'assistant', content: 'ignored' }, { role: 'user', content: '   ' }],
      '李四',
      '产品经理',
    )
    const payload = buildApplicationAnalysisRunRequest({
      sessionId: 44,
      message,
      applicationId: 55,
      clientRequestId: 'option-request',
    })
    expect(payload.message ?? '').toContain('李四')
    expect(payload.message ?? '').toContain('产品经理')
    expect(payload.message ?? '').toContain('匹配度')
    expect((payload.message ?? '').trim()).not.toBe('')
    expect(payload.action_type).toBe('analyze_application')
  })
})
