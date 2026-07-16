import { describe, expect, it } from 'vitest'
import { normalizeAgentRunEvent } from './agentRun'
import type { AgentRunEvent } from '@/types/agentRun'

describe('normalizeAgentRunEvent', () => {
  it('keeps process metadata events as process events and extracts payload fields', () => {
    const event: AgentRunEvent = {
      run_id: 1,
      seq: 2,
      event_type: 'process.delta',
      payload_json: JSON.stringify({
        status: 'running',
        source_event: 'context_usage',
        event_message: 'context usage estimated',
        display_message: '我已确认上下文容量，准备整理工具结果。',
        step_key: 'compose_answer',
        result_metadata: {
          context_usage: {
            model_id: 7,
            model_name: 'deepseek-chat',
            prompt_tokens_estimated: 120,
          },
        },
      }),
    }

    const normalized = normalizeAgentRunEvent(event)

    expect(normalized.event_type).toBe('process.delta')
    expect(normalized.status).toBe('running')
    expect(normalized.event_message).toBe('context usage estimated')
    expect(normalized.display_message).toBe('我已确认上下文容量，准备整理工具结果。')
    expect(normalized.step_key).toBe('compose_answer')
    expect(normalized.result_metadata?.context_usage?.model_id).toBe(7)
  })

  it('keeps process events without result metadata unchanged', () => {
    const event: AgentRunEvent = {
      run_id: 1,
      seq: 3,
      event_type: 'process.delta',
      delta: 'planning',
    }

    expect(normalizeAgentRunEvent(event)).toEqual(event)
  })

  it('keeps gateway-shaped process events with top-level metadata unchanged', () => {
    const event: AgentRunEvent = {
      run_id: 1,
      seq: 4,
      event_type: 'process.delta',
      payload_json: JSON.stringify({
        status: 'running',
        source_event: 'context_usage',
        event_message: 'context usage estimated',
      }),
      result_metadata: {
        context_usage: {
          model_id: 8,
          model_name: 'qwen-plus',
          prompt_tokens_estimated: 240,
        },
      },
    }

    const normalized = normalizeAgentRunEvent(event)

    expect(normalized.event_type).toBe('process.delta')
    expect(normalized.event_message).toBe('context usage estimated')
    expect(normalized.result_metadata?.context_usage?.model_id).toBe(8)
  })
})
