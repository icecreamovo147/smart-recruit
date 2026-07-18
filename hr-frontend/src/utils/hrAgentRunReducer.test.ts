import { describe, expect, it } from 'vitest'
import type { AgentRunEvent, AgentRunSnapshot } from '@/types/agentRun'
import {
  applyAgentRunEvents,
  createInitialHrAgentRunState,
  hydrateFromSnapshot,
  reduceAgentRunEvent,
} from './hrAgentRunReducer'

const baseEvent = (partial: Partial<AgentRunEvent> & Pick<AgentRunEvent, 'event_type' | 'seq'>): AgentRunEvent => ({
  run_id: 42,
  ...partial,
})

describe('hrAgentRunReducer', () => {
  it('creates empty initial state', () => {
    const state = createInitialHrAgentRunState()
    expect(state.runId).toBeNull()
    expect(state.assistantText).toBe('')
    expect(state.lastEventSeq).toBe(0)
    expect(state.isTerminal).toBe(false)
  })

  it('hydrates from snapshot without duplicating later deltas incorrectly', () => {
    const snapshot: AgentRunSnapshot = {
      run_id: 7,
      session_id: 3,
      status: 'running',
      assistant_text: 'Hello',
      process_text: 'thinking',
      last_event_seq: 5,
      confirmation_request: { required: true, reason: 'pick skills' },
      model_name: 'gpt',
    }
    const state = hydrateFromSnapshot(snapshot)
    expect(state.runId).toBe(7)
    expect(state.sessionId).toBe(3)
    expect(state.assistantText).toBe('Hello')
    expect(state.processText).toBe('thinking')
    expect(state.lastEventSeq).toBe(5)
    expect(state.confirmation?.reason).toBe('pick skills')
    expect(state.isTerminal).toBe(false)
  })

  it('appends assistant and process deltas', () => {
    let state = createInitialHrAgentRunState({ runId: 42, status: 'running' })
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 1, event_type: 'assistant.delta', delta: 'Hel' }),
    )
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 2, event_type: 'assistant.delta', delta: 'lo' }),
    )
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 3, event_type: 'process.delta', delta: 'p1' }),
    )
    expect(state.assistantText).toBe('Hello')
    expect(state.processText).toBe('p1')
    expect(state.lastEventSeq).toBe(3)
  })

  it('uses event_message as visible process text for durable stream events', () => {
    const state = applyAgentRunEvents(createInitialHrAgentRunState({ runId: 42 }), [
      baseEvent({ seq: 1, event_type: 'process.delta', event_message: 'planning HR recruiting context' }),
      baseEvent({ seq: 2, event_type: 'tool.started', tool_name: 'get_candidate_detail', event_message: 'querying get_candidate_detail' }),
      baseEvent({ seq: 3, event_type: 'tool.finished', tool_name: 'get_candidate_detail', event_message: 'get_candidate_detail finished' }),
    ])

    expect(state.processText).toBe('我正在判断问题意图，并规划需要读取哪些招聘数据。\n我正在查询实时招聘数据。\n已获取一项实时招聘数据。')
    expect(state.lastToolName).toBe('get_candidate_detail')
  })

  it('keeps process context usage metadata without changing event type', () => {
    const state = reduceAgentRunEvent(
      createInitialHrAgentRunState({ runId: 42, modelName: '默认模型' }),
      baseEvent({
        seq: 1,
        event_type: 'process.delta',
        event_message: 'context usage estimated',
        result_metadata: {
          context_usage: {
            model_id: 8,
            model_name: 'deepseek-v4-flash',
            prompt_tokens_estimated: 240,
          },
        },
      }),
    )
    expect(state.processText).toBe('我已确认上下文容量，准备整理工具结果。')
    expect(state.resultMetadata?.context_usage?.model_id).toBe(8)
    expect(state.modelId).toBe(8)
    expect(state.modelName).toBe('deepseek-v4-flash')
  })

  it('updates model name from run result metadata during live streaming', () => {
    const state = reduceAgentRunEvent(
      createInitialHrAgentRunState({ runId: 42, modelName: '默认模型' }),
      baseEvent({
        seq: 1,
        event_type: 'run.result',
        result_metadata: {
          context_usage: {
            model_id: 1,
            model_name: 'deepseek-v4-flash',
          },
        },
      }),
    )

    expect(state.modelId).toBe(1)
    expect(state.modelName).toBe('deepseek-v4-flash')
  })

  it('prefers backend display_message over local fallback wording', () => {
    const state = reduceAgentRunEvent(
      createInitialHrAgentRunState({ runId: 42 }),
      baseEvent({
        seq: 1,
        event_type: 'tool.started',
        tool_name: 'get_candidate_detail',
        event_message: 'querying get_candidate_detail',
        display_message: '我正在读取当前投递和候选人上下文。',
      }),
    )
    expect(state.processText).toBe('我正在读取当前投递和候选人上下文。')
  })

  it('snapshots replace text buffers', () => {
    let state = createInitialHrAgentRunState({
      runId: 42,
      assistantText: 'old',
      processText: 'old-p',
      lastEventSeq: 1,
    })
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 2, event_type: 'assistant.snapshot', snapshot_text: 'fresh answer' }),
    )
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 3, event_type: 'process.snapshot', snapshot_text: 'fresh process' }),
    )
    expect(state.assistantText).toBe('fresh answer')
    expect(state.processText).toBe('fresh process')
  })

  it('ignores duplicate and stale sequence numbers', () => {
    let state = createInitialHrAgentRunState({ runId: 42, lastEventSeq: 2, assistantText: 'ab' })
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 2, event_type: 'assistant.delta', delta: 'XX' }),
    )
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 1, event_type: 'assistant.delta', delta: 'YY' }),
    )
    expect(state.assistantText).toBe('ab')
    expect(state.lastEventSeq).toBe(2)
  })

  it('refresh hydrate + replay after last_event_seq does not duplicate assistant text', () => {
    // Simulate page refresh: snapshot already includes text through seq 5.
    const hydrated = hydrateFromSnapshot({
      run_id: 42,
      session_id: 1,
      status: 'running',
      assistant_text: 'Hello world',
      process_text: 'step-1',
      last_event_seq: 5,
    })
    // Replayed events with seq <= 5 must be ignored; only seq 6+ append.
    const next = applyAgentRunEvents(hydrated, [
      baseEvent({ seq: 4, event_type: 'assistant.delta', delta: 'Hello' }),
      baseEvent({ seq: 5, event_type: 'assistant.delta', delta: ' world' }),
      baseEvent({ seq: 6, event_type: 'assistant.delta', delta: '!' }),
      baseEvent({ seq: 7, event_type: 'process.delta', delta: '→step-2' }),
    ])
    expect(next.assistantText).toBe('Hello world!')
    expect(next.processText).toBe('step-1→step-2')
    expect(next.lastEventSeq).toBe(7)
  })

  it('ignores events for a different run id', () => {
    let state = createInitialHrAgentRunState({ runId: 42, assistantText: 'keep' })
    state = reduceAgentRunEvent(state, {
      run_id: 99,
      seq: 10,
      event_type: 'assistant.delta',
      delta: 'stale-run',
    })
    expect(state.assistantText).toBe('keep')
    expect(state.lastEventSeq).toBe(0)
  })

  it('handles status, confirmation, result, error, cancel, and completion', () => {
    let state = createInitialHrAgentRunState({ runId: 42 })

    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 1, event_type: 'run.created', status: 'queued' }),
    )
    expect(state.status).toBe('queued')

    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 2, event_type: 'run.status_changed', status: 'running' }),
    )
    expect(state.status).toBe('running')

    state = reduceAgentRunEvent(
      state,
      baseEvent({
        seq: 3,
        event_type: 'confirmation.required',
        confirmation: { required: true, reason: 'skills' },
      }),
    )
    expect(state.status).toBe('waiting_confirmation')
    expect(state.confirmation?.reason).toBe('skills')

    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 4, event_type: 'confirmation.accepted', status: 'running' }),
    )
    expect(state.status).toBe('running')

    state = reduceAgentRunEvent(
      state,
      baseEvent({
        seq: 5,
        event_type: 'run.result',
        result_metadata: { action: 'analyze', candidate_name: 'Ada' },
      }),
    )
    expect(state.resultMetadata?.candidate_name).toBe('Ada')

    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 6, event_type: 'run.completed', status: 'succeeded' }),
    )
    expect(state.status).toBe('succeeded')
    expect(state.isTerminal).toBe(true)

    // Terminal cancel path on a fresh run
    let canceled = createInitialHrAgentRunState({ runId: 42, status: 'running' })
    canceled = reduceAgentRunEvent(
      canceled,
      baseEvent({ seq: 1, event_type: 'run.canceled' }),
    )
    expect(canceled.status).toBe('canceled')
    expect(canceled.isTerminal).toBe(true)

    let failed = createInitialHrAgentRunState({ runId: 42 })
    failed = reduceAgentRunEvent(
      failed,
      baseEvent({
        seq: 1,
        event_type: 'run.error',
        error_type: 'provider',
        error_message: 'boom',
      }),
    )
    expect(failed.status).toBe('failed')
    expect(failed.errorType).toBe('provider')
    expect(failed.errorMessage).toBe('boom')
    expect(failed.isTerminal).toBe(true)
  })

  it('advances seq on heartbeat without changing text', () => {
    let state = createInitialHrAgentRunState({ runId: 42, assistantText: 'x', lastEventSeq: 1 })
    state = reduceAgentRunEvent(
      state,
      baseEvent({ seq: 2, event_type: 'run.heartbeat' }),
    )
    expect(state.lastEventSeq).toBe(2)
    expect(state.assistantText).toBe('x')
  })

  it('applies ordered batches and records tool events', () => {
    const state = applyAgentRunEvents(createInitialHrAgentRunState({ runId: 42 }), [
      baseEvent({ seq: 1, event_type: 'tool.started', tool_name: 'search' }),
      baseEvent({ seq: 2, event_type: 'tool.finished', tool_name: 'search' }),
      baseEvent({ seq: 3, event_type: 'assistant.delta', delta: 'ok' }),
    ])
    expect(state.lastToolName).toBe('search')
    expect(state.assistantText).toBe('ok')
  })

  it('parses confirmation from payload_json when confirmation field missing', () => {
    const state = reduceAgentRunEvent(
      createInitialHrAgentRunState({ runId: 42 }),
      baseEvent({
        seq: 1,
        event_type: 'confirmation.required',
        payload_json: JSON.stringify({ required: true, reason: 'from-json' }),
      }),
    )
    expect(state.confirmation?.reason).toBe('from-json')
    expect(state.status).toBe('waiting_confirmation')
  })
})
