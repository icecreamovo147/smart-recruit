import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'
import type { AgentRunEvent, AgentRunSnapshot } from '@shared/types/agentRun'
import { createInitialHrAgentRunState, reduceAgentRunEvent } from '@/utils/hrAgentRunReducer'
import { useHrAgentRun } from '@/composables/useHrAgentRun'
import {
  bindRunStateToChatUi,
  createClientRequestId,
  executeConfirmChatRun,
  executeCreateChatRun,
  parseCandidateOptionsFromMeta,
  toAgentSkillSelectionPayload,
  type DurableChatUiBinder,
} from './agentRunChatFlow'

const {
  createAgentRun,
  getActiveAgentRun,
  getAgentRun,
  cancelAgentRun,
  confirmAgentRun,
  subscribeAgentRunEvents,
} = vi.hoisted(() => ({
  createAgentRun: vi.fn(),
  getActiveAgentRun: vi.fn(),
  getAgentRun: vi.fn(),
  cancelAgentRun: vi.fn(),
  confirmAgentRun: vi.fn(),
  subscribeAgentRunEvents: vi.fn(),
}))

vi.mock('@/api/agentRun', () => ({
  createAgentRun,
  getActiveAgentRun,
  getAgentRun,
  cancelAgentRun,
  confirmAgentRun,
  subscribeAgentRunEvents,
}))

const snapshot = (partial: Partial<AgentRunSnapshot> = {}): AgentRunSnapshot => ({
  run_id: 100,
  session_id: 9,
  status: 'queued',
  assistant_text: '',
  process_text: '',
  last_event_seq: 0,
  ...partial,
})

function mountRuntime() {
  let api: ReturnType<typeof useHrAgentRun> | undefined
  const Comp = defineComponent({
    setup() {
      api = useHrAgentRun()
      return () => null
    },
  })
  const wrapper = mount(Comp)
  if (!api) throw new Error('composable not initialized')
  return { api, wrapper }
}

function collectBinder() {
  const deltas: string[] = []
  const processDeltas: string[] = []
  const snapshots: string[] = []
  const options: unknown[] = []
  const binder: DurableChatUiBinder = {
    onAssistantDelta: (d) => deltas.push(d),
    onAssistantSnapshot: (t) => snapshots.push(t),
    onProcessDelta: (d) => processDeltas.push(d),
    onProcessSnapshot: () => {},
    onCandidateOptions: (o) => options.push(o),
  }
  return { binder, deltas, processDeltas, snapshots, options }
}

describe('agentRunChatFlow helpers', () => {
  it('createClientRequestId returns non-empty id', () => {
    expect(createClientRequestId().length).toBeGreaterThan(8)
  })

  it('toAgentSkillSelectionPayload maps nested agent_skill_selection', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      reason: 'pick',
      raw_json: JSON.stringify({
        required: true,
        reason: 'pick',
        agent_skill_selection: {
          candidates: [
            {
              id: 7,
              name: 'resume_review',
              display_name: '简历复核',
              recommended: true,
            },
          ],
          recommended_agent_skill_ids: [7],
          user_message_id: 42,
        },
      }),
    })
    expect(payload?.candidates[0]?.id).toBe(7)
    expect(payload?.recommended_agent_skill_ids).toEqual([7])
    expect(payload?.user_message_id).toBe(42)
  })

  it('parseCandidateOptionsFromMeta parses JSON string', () => {
    const options = parseCandidateOptionsFromMeta({
      candidate_options: JSON.stringify([
        { application_id: 1, candidate_name: 'A', job_title: 'FE', masked_phone: '***', round_no: 1 },
      ]),
    })
    expect(options).toHaveLength(1)
    expect(options[0].application_id).toBe(1)
  })

  it('pushes realtime SSE context usage into the chat binder', async () => {
    const state = shallowRef(createInitialHrAgentRunState({ sessionId: 9 }))
    const usages: Array<{ tokens?: number; sessionId?: number | null }> = []
    const stop = bindRunStateToChatUi(state, {
      onAssistantDelta: () => {},
      onAssistantSnapshot: () => {},
      onProcessDelta: () => {},
      onProcessSnapshot: () => {},
      onContextUsage: (value, sessionId) => usages.push({
        tokens: value.prompt_tokens_actual,
        sessionId,
      }),
    })
    state.value = {
      ...state.value,
      resultMetadata: {
        context_usage: {
          prompt_tokens_actual: 851,
          input_budget_tokens: 6_758,
        },
      },
    }
    await nextTick()
    expect(usages).toEqual([{ tokens: 851, sessionId: 9 }])
    stop()
  })
})

describe('agentRunChatFlow entry paths (submit + skill confirm)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    subscribeAgentRunEvents.mockImplementation(async () => {})
  })

  it('submit path: create run reduces assistant deltas then completes', async () => {
    createAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 55, session_id: 3, status: 'running', last_event_seq: 0 }),
    })

    subscribeAgentRunEvents.mockImplementation(
      async (
        _runId: number,
        _after: number,
        handlers: { onEvent?: (e: AgentRunEvent) => void; onDone?: () => void },
      ) => {
        handlers.onEvent?.({
          run_id: 55,
          seq: 1,
          event_type: 'assistant.delta',
          delta: 'Hello ',
        })
        handlers.onEvent?.({
          run_id: 55,
          seq: 2,
          event_type: 'assistant.delta',
          delta: 'world',
        })
        handlers.onEvent?.({
          run_id: 55,
          seq: 3,
          event_type: 'run.result',
          result_metadata: {
            candidate_options: JSON.stringify([
              {
                application_id: 9,
                candidate_name: 'Zhang',
                job_title: 'Backend',
                masked_phone: '***',
                round_no: 1,
              },
            ]),
          },
        })
        handlers.onEvent?.({
          run_id: 55,
          seq: 4,
          event_type: 'run.completed',
          status: 'succeeded',
        })
        handlers.onDone?.()
      },
    )

    const { api, wrapper } = mountRuntime()
    const { binder, deltas, options } = collectBinder()

    const result = await executeCreateChatRun(
      api,
      { session_id: 3, message: 'hi', client_request_id: 'req-submit-1' },
      binder,
      { isAborted: () => false },
    )

    expect(createAgentRun).toHaveBeenCalledWith(
      expect.objectContaining({
        session_id: 3,
        message: 'hi',
        client_request_id: 'req-submit-1',
      }),
    )
    expect(result.outcome).toBe('terminal')
    expect(result.state.assistantText).toBe('Hello world')
    expect(deltas.join('')).toBe('Hello world')
    expect(options).toHaveLength(1)
    expect(api.state.value.isTerminal).toBe(true)

    wrapper.unmount()
  })

  it('skill confirm path: confirm resumes same run and reaches terminal', async () => {
    confirmAgentRun.mockResolvedValue({
      run: snapshot({
        run_id: 33,
        session_id: 4,
        status: 'running',
        last_event_seq: 5,
        assistant_text: '',
      }),
    })

    subscribeAgentRunEvents.mockImplementation(
      async (
        runId: number,
        _after: number,
        handlers: { onEvent?: (e: AgentRunEvent) => void; onDone?: () => void },
      ) => {
        handlers.onEvent?.({
          run_id: runId,
          seq: 6,
          event_type: 'assistant.delta',
          delta: 'confirmed reply',
        })
        handlers.onEvent?.({
          run_id: runId,
          seq: 7,
          event_type: 'run.completed',
          status: 'succeeded',
        })
        handlers.onDone?.()
      },
    )

    const { api, wrapper } = mountRuntime()
    api.state.value = createInitialHrAgentRunState({
      runId: 33,
      sessionId: 4,
      status: 'waiting_confirmation',
      lastEventSeq: 5,
      confirmation: {
        required: true,
        reason: 'pick skills',
        candidates: [{ id: 1, name: 'skill_a', display_name: 'Skill A' }],
        recommended_agent_skill_ids: [1],
      },
    })

    const { binder, deltas } = collectBinder()
    const result = await executeConfirmChatRun(
      api,
      {
        client_request_id: 'req-confirm-1',
        agent_skill_ids: [1],
        agent_skill_selection_confirmed: true,
      },
      binder,
      { isAborted: () => false },
    )

    expect(confirmAgentRun).toHaveBeenCalledWith(
      33,
      expect.objectContaining({
        agent_skill_ids: [1],
        agent_skill_selection_confirmed: true,
        client_request_id: 'req-confirm-1',
      }),
    )
    expect(createAgentRun).not.toHaveBeenCalled()
    expect(result.outcome).toBe('terminal')
    expect(result.state.assistantText).toContain('confirmed reply')
    expect(deltas.join('')).toContain('confirmed reply')

    wrapper.unmount()
  })

  it('submit path parks at waiting_confirmation for skill selection', async () => {
    createAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 70, session_id: 2, status: 'running', last_event_seq: 0 }),
    })
    subscribeAgentRunEvents.mockImplementation(
      async (
        _runId: number,
        _after: number,
        handlers: { onEvent?: (e: AgentRunEvent) => void; onDone?: () => void },
      ) => {
        handlers.onEvent?.({
          run_id: 70,
          seq: 1,
          event_type: 'confirmation.required',
          status: 'waiting_confirmation',
          payload_json: JSON.stringify({
            required: true,
            reason: 'select',
            candidates: [{ id: 2, name: 's2', display_name: 'S2' }],
            recommended_agent_skill_ids: [2],
          }),
        })
        handlers.onDone?.()
      },
    )

    const { api, wrapper } = mountRuntime()
    const { binder } = collectBinder()
    const result = await executeCreateChatRun(
      api,
      { session_id: 2, message: 'need skills', client_request_id: 'req-skill-1' },
      binder,
      { isAborted: () => false },
    )

    expect(result.outcome).toBe('waiting_confirmation')
    expect(result.state.status).toBe('waiting_confirmation')
    const selection = toAgentSkillSelectionPayload(result.state.confirmation)
    expect(selection?.candidates[0]?.id).toBe(2)

    wrapper.unmount()
  })
})

describe('useHrAgentRun waitUntilSettled', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    subscribeAgentRunEvents.mockImplementation(async () => {})
  })

  it('resolves waiting_confirmation without requiring terminal', async () => {
    const { api, wrapper } = mountRuntime()
    api.state.value = createInitialHrAgentRunState({
      runId: 1,
      status: 'running',
    })

    const pending = api.waitUntilSettled({ runId: 1 })
    await nextTick()
    api.applyEvent({
      run_id: 1,
      seq: 1,
      event_type: 'confirmation.required',
      status: 'waiting_confirmation',
      confirmation: { required: true, reason: 'x' },
    })
    await expect(pending).resolves.toBe('waiting_confirmation')
    wrapper.unmount()
  })

  it('reduceAgentRunEvent keeps confirm path independent of create', () => {
    let state = createInitialHrAgentRunState({ runId: 9, status: 'waiting_confirmation' })
    state = reduceAgentRunEvent(state, {
      run_id: 9,
      seq: 2,
      event_type: 'confirmation.accepted',
      status: 'running',
    })
    state = reduceAgentRunEvent(state, {
      run_id: 9,
      seq: 3,
      event_type: 'assistant.delta',
      delta: 'ok',
    })
    state = reduceAgentRunEvent(state, {
      run_id: 9,
      seq: 4,
      event_type: 'run.completed',
      status: 'succeeded',
    })
    expect(state.assistantText).toBe('ok')
    expect(state.isTerminal).toBe(true)
  })
})
