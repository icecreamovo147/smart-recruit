/**
 * Focused recovery tests for AIChatView refresh restore helpers.
 * Full view mount is heavy (router, many APIs); composable tests cover reconnect/cancel.
 * Here we exercise message-slot seeding rules used by restoreActiveRunForSession.
 */
import { describe, expect, it } from 'vitest'

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
