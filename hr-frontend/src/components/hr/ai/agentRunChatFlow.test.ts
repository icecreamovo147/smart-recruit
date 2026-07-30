import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'
import type { AgentRunEvent, AgentRunSnapshot } from '@shared/types/agentRun'
import { createInitialHrAgentRunState, reduceAgentRunEvent } from '@/utils/hrAgentRunReducer'
import { useHrAgentRun } from '@/composables/useHrAgentRun'
import {
  bindRunStateToChatUi,
  cancelPendingAgentRun,
  createClientRequestId,
  executeConfirmChatRun,
  executeCreateChatRun,
  friendlyDurableRunErrorMessage,
  hasPendingRunConfirmation,
  insufficientCreditsMessage,
  isAgentSkillConfirmationExpired,
  isMCPConfirmationExpired,
  modelFallbackMessage,
  parseCandidateOptionsFromMeta,
  streamTimeoutMessage,
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
  const resultMetadata: unknown[] = []
  const binder: DurableChatUiBinder = {
    onAssistantDelta: (d) => deltas.push(d),
    onAssistantSnapshot: (t) => snapshots.push(t),
    onProcessDelta: (d) => processDeltas.push(d),
    onProcessSnapshot: () => {},
    onCandidateOptions: (o) => options.push(o),
    onResultMetadata: (meta) => resultMetadata.push(meta),
  }
  return { binder, deltas, processDeltas, snapshots, options, resultMetadata }
}

describe('agentRunChatFlow helpers', () => {
  it('createClientRequestId returns non-empty id', () => {
    expect(createClientRequestId().length).toBeGreaterThan(8)
  })

  it('maps persisted asynchronous credit errors to the billing guidance', () => {
    expect(friendlyDurableRunErrorMessage(
      'provider',
      'rpc error: code = ResourceExhausted desc = insufficient_credits',
    )).toBe(insufficientCreditsMessage)
    expect(friendlyDurableRunErrorMessage('insufficient_credits', '')).toBe(insufficientCreditsMessage)
  })

  it('uses dedicated timeout and parameterized model-fallback messages', () => {
    expect(streamTimeoutMessage()).toBe('AI 服务响应超时，请稍后重试')
    expect(modelFallbackMessage('GPT-4.1')).toBe('请求的模型不可用，已自动切换至 GPT-4.1')
  })

  it('maps exact Package v2 candidates without legacy aliases', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      reason: 'pick',
      agent_skill_confirmation_id: 'skill-confirm-7',
      recommended_agent_skill_version_ids: [18, 17],
      agent_skill_confirmation_expires_at: '2026-07-28T12:10:00Z',
      candidates: [
        {
          skill_id: 7,
          version_id: 17,
          version: '2.0.0',
          compiled_hash: 'abcdef0123456789',
          name: 'resume_review',
          display_name: '简历复核',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 320,
          recommended: true,
        },
        {
          skill_id: 8,
          version_id: 18,
          version: '1.0.0',
          compiled_hash: 'fedcba9876543210',
          name: 'interview_rubric',
          display_name: '面试评估',
          composition_role: 'supporting',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 180,
          recommended: true,
        },
      ],
    })
    expect(payload?.candidates[0]?.version_id).toBe(17)
    expect(payload?.candidates[0]?.risk).toBe('high')
    expect(payload?.recommended_agent_skill_version_ids).toEqual([18, 17])
    expect(payload?.confirmation_id).toBe('skill-confirm-7')
    expect(payload?.expires_at).toBe('2026-07-28T12:10:00Z')
    expect(payload).not.toHaveProperty('recommended_agent_skill_ids')
    expect(payload).not.toHaveProperty('user_message_id')
  })

  it('detects expired MCP confirmations from the bound approval payload', () => {
    const payload = JSON.stringify({
      type: 'mcp_tool',
      expires_at: '2026-07-28T12:10:00Z',
    })
    expect(isMCPConfirmationExpired(payload, Date.parse('2026-07-28T12:09:59Z'))).toBe(false)
    expect(isMCPConfirmationExpired(payload, Date.parse('2026-07-28T12:10:00Z'))).toBe(true)
    expect(isMCPConfirmationExpired('not-json', Date.now())).toBe(false)
  })

  it('detects exact Agent Skill confirmation expiry', () => {
    const expiresAt = '2026-07-28T12:10:00Z'
    expect(isAgentSkillConfirmationExpired(expiresAt, Date.parse('2026-07-28T12:09:59Z'))).toBe(false)
    expect(isAgentSkillConfirmationExpired(expiresAt, Date.parse(expiresAt))).toBe(true)
    expect(isAgentSkillConfirmationExpired(undefined, Date.now())).toBe(false)
  })

  it('blocks a second run while durable confirmation is pending', () => {
    expect(hasPendingRunConfirmation('waiting_confirmation', [])).toBe(true)
    expect(hasPendingRunConfirmation('running', [{
      agentSkillSelection: { required: true },
      skillSelectionRequest: { runId: 35 },
    }])).toBe(true)
    expect(hasPendingRunConfirmation('succeeded', [])).toBe(false)
  })

  it('preserves a cancel-only card when refreshed confirmation data is incomplete', () => {
    expect(toAgentSkillSelectionPayload({
      required: true,
      reason: 'confirmation payload incomplete',
      agent_skill_confirmation_id: 'skill-confirm-incomplete',
      recommended_agent_skill_version_ids: [18, 17],
    })).toEqual({
      required: true,
      reason: 'confirmation payload incomplete',
      candidates: [],
      confirmation_kind: 'agent_skill',
      confirmation_id: 'skill-confirm-incomplete',
      recommended_agent_skill_version_ids: [],
    })
  })

  it('preserves a cancel-only card when typed candidates are missing a confirmation ID', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      candidates: [{
        skill_id: 7,
        version_id: 17,
        version: '2.0.0',
        compiled_hash: 'abcdef0123456789',
        name: 'resume_review',
        display_name: '简历复核',
        composition_role: 'primary',
        risk: 'high',
        activation_policy: 'confirm',
        core_estimated_tokens: 320,
        recommended: true,
      }],
    })

    expect(payload).toEqual({
      required: true,
      reason: 'Agent Skill 确认信息不完整',
      candidates: [],
      confirmation_kind: 'agent_skill',
      recommended_agent_skill_version_ids: [],
    })
  })

  it('preserves a cancel-only card when typed candidates are invalid', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      reason: 'restore failed',
      agent_skill_confirmation_id: 'skill-confirm-7',
      candidates: [{
        skill_id: 7,
        version_id: 0,
        version: '',
        compiled_hash: '',
        name: '',
        display_name: '',
        composition_role: 'primary',
        risk: 'high',
        activation_policy: 'confirm',
        core_estimated_tokens: 320,
        recommended: true,
      }],
    })

    expect(payload).toEqual({
      required: true,
      reason: 'restore failed',
      candidates: [],
      confirmation_kind: 'agent_skill',
      confirmation_id: 'skill-confirm-7',
      recommended_agent_skill_version_ids: [],
    })
  })

  it('fails closed instead of filtering a partially invalid candidate snapshot', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      reason: 'restore failed',
      agent_skill_confirmation_id: 'skill-confirm-partial',
      recommended_agent_skill_version_ids: [18, 17],
      candidates: [
        {
          skill_id: 8,
          version_id: 18,
          version: '1.0.0',
          compiled_hash: 'hash-18',
          name: 'interview_rubric',
          display_name: '面试评估',
          composition_role: 'supporting',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 180,
          recommended: true,
        },
        {
          skill_id: 7,
          version_id: 0,
          version: '2.0.0',
          compiled_hash: 'hash-17',
          name: 'resume_review',
          display_name: '简历复核',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 320,
          recommended: true,
        },
      ],
    })

    expect(payload?.candidates).toEqual([])
    expect(payload?.recommended_agent_skill_version_ids).toEqual([])
  })

  it.each([
    ['string coercion', ['18', 17] as unknown as number[]],
    ['fractional ID', [18.5, 17]],
    ['zero ID', [0, 17]],
    ['negative ID', [-18, 17]],
    ['duplicate ID', [18, 18]],
  ])('fails closed for %s in the exact recommendation snapshot', (_name, recommendedIDs) => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      agent_skill_confirmation_id: 'skill-confirm-invalid-ids',
      recommended_agent_skill_version_ids: recommendedIDs,
      candidates: [
        {
          skill_id: 8,
          version_id: 18,
          version: '1.0.0',
          compiled_hash: 'hash-18',
          name: 'interview_rubric',
          display_name: '面试评估',
          composition_role: 'supporting',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 180,
          recommended: true,
        },
        {
          skill_id: 7,
          version_id: 17,
          version: '2.0.0',
          compiled_hash: 'hash-17',
          name: 'resume_review',
          display_name: '简历复核',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 320,
          recommended: true,
        },
      ],
    })

    expect(payload?.candidates).toEqual([])
    expect(payload?.recommended_agent_skill_version_ids).toEqual([])
  })

  it.each([
    ['missing candidate', [18]],
    ['extra candidate', [18, 17, 16]],
  ])('fails closed when the exact recommendation has a %s', (_name, recommendedIDs) => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      agent_skill_confirmation_id: 'skill-confirm-membership',
      recommended_agent_skill_version_ids: recommendedIDs,
      candidates: [
        {
          skill_id: 8,
          version_id: 18,
          version: '1.0.0',
          compiled_hash: 'hash-18',
          name: 'interview_rubric',
          display_name: '面试评估',
          composition_role: 'supporting',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 180,
          recommended: true,
        },
        {
          skill_id: 7,
          version_id: 17,
          version: '2.0.0',
          compiled_hash: 'hash-17',
          name: 'resume_review',
          display_name: '简历复核',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 320,
          recommended: true,
        },
      ],
    })

    expect(payload?.candidates).toEqual([])
    expect(payload?.recommended_agent_skill_version_ids).toEqual([])
  })

  it('fails closed when candidate version IDs are duplicated', () => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      agent_skill_confirmation_id: 'skill-confirm-duplicate-candidates',
      recommended_agent_skill_version_ids: [18, 17],
      candidates: [
        {
          skill_id: 8,
          version_id: 18,
          version: '1.0.0',
          compiled_hash: 'hash-18-a',
          name: 'interview_rubric',
          display_name: '面试评估',
          composition_role: 'supporting',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 180,
          recommended: true,
        },
        {
          skill_id: 9,
          version_id: 18,
          version: '1.0.1',
          compiled_hash: 'hash-18-b',
          name: 'interview_rubric_copy',
          display_name: '面试评估副本',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 200,
          recommended: true,
        },
      ],
    })

    expect(payload?.candidates).toEqual([])
    expect(payload?.recommended_agent_skill_version_ids).toEqual([])
  })

  it.each([
    ['string version ID', { version_id: '18' as unknown as number }],
    ['string skill ID', { skill_id: '8' as unknown as number }],
    ['missing version', { version: '' }],
    ['missing compiled hash', { compiled_hash: '' }],
    ['missing display identity', { name: '', display_name: '' }],
    ['invalid risk', { risk: 'unknown' as 'high' }],
    ['invalid activation policy', { activation_policy: 'sometimes' as 'confirm' }],
    ['invalid composition role', { composition_role: 'secondary' as 'primary' }],
    ['invalid token estimate', { core_estimated_tokens: Number.NaN }],
    ['not recommended', { recommended: false }],
  ])('fails closed for an incomplete candidate: %s', (_name, override) => {
    const payload = toAgentSkillSelectionPayload({
      required: true,
      agent_skill_confirmation_id: 'skill-confirm-invalid-candidate',
      recommended_agent_skill_version_ids: [18],
      candidates: [{
        skill_id: 8,
        version_id: 18,
        version: '1.0.0',
        compiled_hash: 'hash-18',
        name: 'interview_rubric',
        display_name: '面试评估',
        composition_role: 'supporting',
        risk: 'high',
        activation_policy: 'confirm',
        core_estimated_tokens: 180,
        recommended: true,
        ...override,
      }],
    })

    expect(payload?.candidates).toEqual([])
    expect(payload?.recommended_agent_skill_version_ids).toEqual([])
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
            suggested_questions: ['查看候选人详情', '分析岗位匹配度', '准备面试问题'],
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
    const { binder, deltas, options, resultMetadata } = collectBinder()

    const result = await executeCreateChatRun(
      api,
      {
        session_id: 3,
        message: 'hi',
        client_request_id: 'req-submit-1',
        agent_skill_version_ids: [701],
      },
      binder,
      { isAborted: () => false },
    )

    expect(createAgentRun).toHaveBeenCalledWith(
      expect.objectContaining({
        session_id: 3,
        message: 'hi',
        client_request_id: 'req-submit-1',
        agent_skill_version_ids: [701],
      }),
    )
    expect(createAgentRun.mock.calls[0]?.[0]).not.toHaveProperty('agent_skill_ids')
    expect(result.outcome).toBe('terminal')
    expect(result.state.assistantText).toBe('Hello world')
    expect(deltas.join('')).toBe('Hello world')
    expect(options).toHaveLength(1)
    expect(resultMetadata).toEqual([
      expect.objectContaining({ suggested_questions: ['查看候选人详情', '分析岗位匹配度', '准备面试问题'] }),
    ])
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
        agent_skill_confirmation_id: 'skill-confirm-1',
        recommended_agent_skill_version_ids: [11],
        candidates: [{
          skill_id: 1,
          version_id: 11,
          version: '2.0.0',
          compiled_hash: 'hash-11',
          name: 'skill_a',
          display_name: 'Skill A',
          composition_role: 'primary',
          risk: 'high',
          activation_policy: 'confirm',
          core_estimated_tokens: 200,
        }],
      },
    })

    const { binder, deltas } = collectBinder()
    const result = await executeConfirmChatRun(
      api,
      {
        client_request_id: 'req-confirm-1',
        agent_skill_confirmation_id: 'skill-confirm-1',
        agent_skill_confirmation_decision: 'approve',
        selected_agent_skill_version_ids: [11],
      },
      binder,
      { isAborted: () => false },
    )

    expect(confirmAgentRun).toHaveBeenCalledWith(
      33,
      expect.objectContaining({
        agent_skill_confirmation_id: 'skill-confirm-1',
        agent_skill_confirmation_decision: 'approve',
        selected_agent_skill_version_ids: [11],
        client_request_id: 'req-confirm-1',
      }),
    )
    expect(createAgentRun).not.toHaveBeenCalled()
    expect(result.outcome).toBe('terminal')
    expect(result.state.assistantText).toContain('confirmed reply')
    expect(deltas.join('')).toContain('confirmed reply')

    wrapper.unmount()
  })

  it('MCP confirm path forwards the bound payload without granting Skill confirmation', async () => {
    const rawConfirmation = JSON.stringify({
      type: 'mcp_tool',
      confirmation_id: 'confirm-1',
      capability_key: '7:search',
      arguments_hash: 'args-hash',
      expires_at: '2026-07-28T12:10:00Z',
    })
    const selection = toAgentSkillSelectionPayload({
      required: true,
      reason: 'confirmation_required',
      raw_json: rawConfirmation,
    })
    expect(selection).toEqual(expect.objectContaining({
      confirmation_kind: 'mcp_tool',
      candidates: [],
    }))
    expect(JSON.parse(selection?.confirmation_payload_json || '{}')).toEqual(expect.objectContaining({
      type: 'mcp_tool',
      confirmation_id: 'confirm-1',
      capability_key: '7:search',
      arguments_hash: 'args-hash',
      expires_at: '2026-07-28T12:10:00Z',
      approved: true,
    }))

    confirmAgentRun.mockResolvedValue({
      run: snapshot({ run_id: 34, session_id: 4, status: 'running', last_event_seq: 5 }),
    })
    subscribeAgentRunEvents.mockImplementation(
      async (
        runId: number,
        _after: number,
        handlers: { onEvent?: (e: AgentRunEvent) => void; onDone?: () => void },
      ) => {
        handlers.onEvent?.({ run_id: runId, seq: 6, event_type: 'run.completed', status: 'succeeded' })
        handlers.onDone?.()
      },
    )
    const { api, wrapper } = mountRuntime()
    api.state.value = createInitialHrAgentRunState({
      runId: 34,
      sessionId: 4,
      status: 'waiting_confirmation',
      lastEventSeq: 5,
    })

    await executeConfirmChatRun(
      api,
      {
        confirmation_payload_json: selection?.confirmation_payload_json,
      },
      collectBinder().binder,
      { isAborted: () => false },
    )

    const mcpRequest = confirmAgentRun.mock.calls[0]?.[1]
    expect(mcpRequest).toEqual({
      confirmation_payload_json: selection?.confirmation_payload_json,
      client_request_id: expect.any(String),
    })
    expect(mcpRequest).not.toHaveProperty('agent_skill_confirmation_id')
    expect(mcpRequest).not.toHaveProperty('agent_skill_confirmation_decision')
    expect(mcpRequest).not.toHaveProperty('selected_agent_skill_version_ids')
    wrapper.unmount()
  })

  it('reject path cancels the exact waiting MCP run and reaches terminal state', async () => {
    cancelAgentRun.mockResolvedValue({
      run: snapshot({
        run_id: 35,
        session_id: 4,
        status: 'canceled',
        last_event_seq: 6,
      }),
    })
    const { api, wrapper } = mountRuntime()
    api.state.value = createInitialHrAgentRunState({
      runId: 35,
      sessionId: 4,
      status: 'waiting_confirmation',
      lastEventSeq: 5,
    })

    const canceled = await cancelPendingAgentRun(api, 35, 'req-reject-mcp')

    expect(cancelAgentRun).toHaveBeenCalledWith(35, {
      client_request_id: 'req-reject-mcp',
    })
    expect(canceled.status).toBe('canceled')
    expect(canceled.isTerminal).toBe(true)
    wrapper.unmount()
  })

  it('reject path restores the expected run before canceling stale local state', async () => {
    getAgentRun.mockResolvedValue({
      run: snapshot({
        run_id: 36,
        session_id: 4,
        status: 'waiting_confirmation',
        last_event_seq: 5,
      }),
    })
    cancelAgentRun.mockResolvedValue({
      run: snapshot({
        run_id: 36,
        session_id: 4,
        status: 'canceled',
        last_event_seq: 6,
      }),
    })
    const { api, wrapper } = mountRuntime()

    const canceled = await cancelPendingAgentRun(api, 36, 'req-reject-restored')

    expect(getAgentRun).toHaveBeenCalledWith(36)
    expect(cancelAgentRun).toHaveBeenCalledWith(36, {
      client_request_id: 'req-reject-restored',
    })
    expect(canceled.status).toBe('canceled')
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
            agent_skill_confirmation_id: 'skill-confirm-2',
            recommended_agent_skill_version_ids: [22],
            candidates: [{
              skill_id: 2,
              version_id: 22,
              version: '2.0.0',
              compiled_hash: 'hash-22',
              name: 's2',
              display_name: 'S2',
              composition_role: 'primary',
              risk: 'critical',
              activation_policy: 'manual_only',
              core_estimated_tokens: 300,
              recommended: true,
            }],
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
    expect(selection?.candidates[0]?.version_id).toBe(22)
    expect(selection?.candidates[0]?.risk).toBe('critical')

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
