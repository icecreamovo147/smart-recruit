/**
 * Shared durable-run chat flow used by AIChatView entry points.
 * Keeps create / confirm paths on the same useHrAgentRun runtime.
 */

import { watch, type ShallowRef, type WatchStopHandle } from 'vue'
import type { UseHrAgentRunReturn } from '@/composables/useHrAgentRun'
import type {
  AgentRunConfirmation,
  AgentRunResultMetadata,
  AgentRunSkillCandidate,
  ConfirmAgentRunRequest,
  CreateAgentRunRequest,
} from '@/types/agentRun'
import type { AgentSkillSelectionCandidate, AgentSkillSelectionPayload, CandidateOption, ContextUsageInfo } from '@/types/ai'
import type { HrAgentRunState } from '@/utils/hrAgentRunReducer'

export type DurableChatOutcome = 'terminal' | 'waiting_confirmation' | 'aborted' | 'failed'

export interface DurableChatFlowResult {
  outcome: DurableChatOutcome
  state: HrAgentRunState
  error?: Error
}

export interface DurableChatUiBinder {
  onAssistantDelta: (delta: string) => void
  onAssistantSnapshot: (text: string) => void
  onProcessDelta: (delta: string) => void
  onProcessSnapshot: (text: string) => void
  onModelName?: (name: string) => void
  onContextUsage?: (usage: ContextUsageInfo, sessionId?: number | null) => void
  onCandidateOptions?: (options: CandidateOption[]) => void
  onResultMetadata?: (meta: AgentRunResultMetadata | null) => void
  onRunError?: (errorType: string, errorMessage: string) => void
}

export function createClientRequestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `hars-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`
}

function asSkillCandidates(list: AgentRunSkillCandidate[] | undefined): AgentSkillSelectionCandidate[] {
  if (!Array.isArray(list)) return []
  return list.map((candidate) => ({
    id: Number(candidate.id) || 0,
    name: candidate.name || '',
    display_name: candidate.display_name || candidate.name || '',
    reason: candidate.reason || '',
    score: Number(candidate.score) || 0,
    priority: Number(candidate.priority) || 0,
    category: candidate.category || '',
    scenario: candidate.scenario || '',
    risk_level: candidate.risk_level || '',
    recommended: Boolean(candidate.recommended),
    vector_score: Number(candidate.vector_score) || 0,
    lexical_score: Number(candidate.lexical_score) || 0,
    metadata_score: Number(candidate.metadata_score) || 0,
    relevance_score: Number(candidate.relevance_score) || 0,
    business_boost: Number(candidate.business_boost) || 0,
    final_rank_score: Number(candidate.final_rank_score) || 0,
    relevance_mode: candidate.relevance_mode || '',
    pool_rank: Number(candidate.pool_rank) || 0,
    ranking_confidence: candidate.ranking_confidence || '',
  }))
}

/**
 * Normalize durable confirmation payloads into the existing skill-selection UI shape.
 * Backend may nest selection under `agent_skill_selection` inside raw JSON.
 */
export function toAgentSkillSelectionPayload(
  confirmation: AgentRunConfirmation | null | undefined,
): AgentSkillSelectionPayload | null {
  if (!confirmation) return null

  const fromNested = (raw: unknown): AgentSkillSelectionPayload | null => {
    if (!raw || typeof raw !== 'object') return null
    const obj = raw as Record<string, unknown>
    const nested = (obj.agent_skill_selection || obj) as Record<string, unknown>
    const candidates = asSkillCandidates(
      (nested.candidates || obj.candidates) as AgentRunSkillCandidate[] | undefined,
    )
    if (candidates.length === 0 && !obj.required && !nested.required) {
      return null
    }
    const recommended =
      (nested.recommended_agent_skill_ids as number[] | undefined) ||
      (obj.recommended_agent_skill_ids as number[] | undefined) ||
      []
    const userMessageId =
      Number(nested.user_message_id ?? obj.user_message_id) || undefined
    return {
      required: Boolean(nested.required ?? obj.required ?? true),
      reason: String(nested.reason || obj.reason || '请确认本次要调用的 Skill'),
      candidates,
      recommended_agent_skill_ids: recommended.map((id) => Number(id)).filter((id) => Number.isFinite(id)),
      ...(userMessageId ? { user_message_id: userMessageId } : {}),
    }
  }

  if (confirmation.candidates && confirmation.candidates.length > 0) {
    return {
      required: confirmation.required ?? true,
      reason: confirmation.reason || '请确认本次要调用的 Skill',
      candidates: asSkillCandidates(confirmation.candidates),
      recommended_agent_skill_ids: (confirmation.recommended_agent_skill_ids || [])
        .map((id) => Number(id))
        .filter((id) => Number.isFinite(id)),
      ...(confirmation.user_message_id
        ? { user_message_id: Number(confirmation.user_message_id) }
        : {}),
    }
  }

  const loose = confirmation as AgentRunConfirmation & {
    agent_skill_selection?: unknown
  }
  if (loose.agent_skill_selection) {
    const mapped = fromNested({
      ...loose,
      agent_skill_selection: loose.agent_skill_selection,
    })
    if (mapped) return mapped
  }

  if (confirmation.raw_json) {
    try {
      const parsed = JSON.parse(confirmation.raw_json) as unknown
      const mapped = fromNested(parsed)
      if (mapped) return mapped
    } catch {
      // fall through
    }
  }

  // Minimal parking signal without candidate cards.
  if (confirmation.required || confirmation.reason) {
    return {
      required: confirmation.required ?? true,
      reason: confirmation.reason || '请确认本次要调用的 Skill',
      candidates: [],
      recommended_agent_skill_ids: (confirmation.recommended_agent_skill_ids || [])
        .map((id) => Number(id))
        .filter((id) => Number.isFinite(id)),
      ...(confirmation.user_message_id
        ? { user_message_id: Number(confirmation.user_message_id) }
        : {}),
    }
  }

  return null
}

export function parseCandidateOptionsFromMeta(
  meta: AgentRunResultMetadata | null | undefined,
): CandidateOption[] {
  if (!meta?.candidate_options) return []
  const value = meta.candidate_options
  if (Array.isArray(value)) return value as unknown as CandidateOption[]
  try {
    return JSON.parse(value) as CandidateOption[]
  } catch {
    return []
  }
}

export function bindRunStateToChatUi(
  stateRef: ShallowRef<HrAgentRunState>,
  binder: DurableChatUiBinder,
): WatchStopHandle {
  let lastAssistant = ''
  let lastProcess = ''
  let lastModelName = ''
  let lastResultKey = ''
  let lastErrorKey = ''

  const seed = stateRef.value
  lastAssistant = seed.assistantText || ''
  lastProcess = seed.processText || ''
  if (lastAssistant) {
    binder.onAssistantSnapshot(lastAssistant)
  }
  if (lastProcess) {
    binder.onProcessSnapshot(lastProcess)
  }

  return watch(
    stateRef,
    (s) => {
      const assistant = s.assistantText || ''
      if (assistant !== lastAssistant) {
        if (assistant.startsWith(lastAssistant)) {
          const delta = assistant.slice(lastAssistant.length)
          if (delta) binder.onAssistantDelta(delta)
        } else {
          binder.onAssistantSnapshot(assistant)
        }
        lastAssistant = assistant
      }

      const process = s.processText || ''
      if (process !== lastProcess) {
        if (!process) {
          binder.onProcessSnapshot('')
        } else if (process.startsWith(lastProcess)) {
          const delta = process.slice(lastProcess.length)
          if (delta) binder.onProcessDelta(delta)
        } else {
          binder.onProcessSnapshot(process)
        }
        lastProcess = process
      }

      if (s.modelName && s.modelName !== lastModelName) {
        lastModelName = s.modelName
        binder.onModelName?.(s.modelName)
      }

      if (s.resultMetadata) {
        const key = JSON.stringify(s.resultMetadata)
        if (key !== lastResultKey) {
          lastResultKey = key
          binder.onResultMetadata?.(s.resultMetadata)
          const options = parseCandidateOptionsFromMeta(s.resultMetadata)
          if (options.length > 0) {
            binder.onCandidateOptions?.(options)
          }
          const usage = s.resultMetadata.context_usage as ContextUsageInfo | null | undefined
          if (usage) {
            binder.onContextUsage?.(usage, s.sessionId)
          }
        }
      }

      if (s.errorType || s.errorMessage) {
        const errKey = `${s.errorType}|${s.errorMessage}`
        if (errKey !== lastErrorKey && s.isTerminal && (s.status === 'failed' || s.errorMessage)) {
          lastErrorKey = errKey
          binder.onRunError?.(s.errorType || 'run_error', s.errorMessage || '运行失败')
        }
      }
    },
    { deep: true, flush: 'sync' },
  )
}

function mapSettlement(
  settlement: 'terminal' | 'waiting_confirmation' | 'aborted',
  state: HrAgentRunState,
): DurableChatFlowResult {
  if (settlement === 'aborted') {
    return { outcome: 'aborted', state }
  }
  if (settlement === 'waiting_confirmation') {
    return { outcome: 'waiting_confirmation', state }
  }
  if (state.status === 'failed' && (state.errorMessage || state.errorType)) {
    return {
      outcome: 'failed',
      state,
      error: new Error(state.errorMessage || state.errorType || '运行失败'),
    }
  }
  return { outcome: 'terminal', state }
}

/**
 * Create a durable run, subscribe, and drive chat UI until settled.
 * UI binder is attached before startRun so synchronous SSE events are not missed.
 */
export async function executeCreateChatRun(
  runtime: UseHrAgentRunReturn,
  payload: CreateAgentRunRequest,
  binder: DurableChatUiBinder,
  opts: { isAborted: () => boolean },
): Promise<DurableChatFlowResult> {
  const stop = bindRunStateToChatUi(runtime.state, binder)
  try {
    const request: CreateAgentRunRequest = {
      ...payload,
      client_request_id: payload.client_request_id || createClientRequestId(),
    }
    const started = await runtime.startRun(request)
    if (opts.isAborted()) {
      return { outcome: 'aborted', state: runtime.state.value }
    }
    const settlement = await runtime.waitUntilSettled({
      runId: started.runId,
      shouldAbort: opts.isAborted,
    })
    return mapSettlement(settlement, runtime.state.value)
  } catch (error: unknown) {
    return {
      outcome: 'failed',
      state: runtime.state.value,
      error: error instanceof Error ? error : new Error('创建 Agent 运行失败'),
    }
  } finally {
    stop()
  }
}

/**
 * Confirm skill selection on the current durable run and drive chat UI until settled.
 */
export async function executeConfirmChatRun(
  runtime: UseHrAgentRunReturn,
  payload: ConfirmAgentRunRequest,
  binder: DurableChatUiBinder,
  opts: { isAborted: () => boolean },
): Promise<DurableChatFlowResult> {
  const stop = bindRunStateToChatUi(runtime.state, binder)
  try {
    const request: ConfirmAgentRunRequest = {
      ...payload,
      client_request_id: payload.client_request_id || createClientRequestId(),
      agent_skill_selection_confirmed:
        payload.agent_skill_selection_confirmed !== false,
    }
    const confirmed = await runtime.confirm(request)
    if (opts.isAborted()) {
      return { outcome: 'aborted', state: runtime.state.value }
    }
    const settlement = await runtime.waitUntilSettled({
      runId: confirmed.runId,
      shouldAbort: opts.isAborted,
    })
    return mapSettlement(settlement, runtime.state.value)
  } catch (error: unknown) {
    return {
      outcome: 'failed',
      state: runtime.state.value,
      error: error instanceof Error ? error : new Error('确认 Agent 运行失败'),
    }
  } finally {
    stop()
  }
}
