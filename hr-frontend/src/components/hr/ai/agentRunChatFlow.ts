import { t } from '@shared/i18n'

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
} from '@shared/types/agentRun'
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

export const insufficientCreditsMessage = 'AI 套餐额度不足，请购买套餐或加量包后重试'

export function streamTimeoutMessage(): string {
  return t('ai.stream_timeout')
}

export function modelFallbackMessage(effectiveModelName: string): string {
  return t('ai.model_fallback', { model: effectiveModelName.trim() })
}

export function hasPendingRunConfirmation(
  runStatus: string,
  messages: Array<{
    agentSkillSelection?: { required?: boolean }
    skillSelectionRequest?: unknown
  }>,
): boolean {
  return runStatus === 'waiting_confirmation'
    || messages.some((message) =>
      Boolean(message.agentSkillSelection?.required && message.skillSelectionRequest),
    )
}

export function isMCPConfirmationExpired(
  confirmationPayloadJSON: string | undefined,
  nowMs = Date.now(),
): boolean {
  if (!confirmationPayloadJSON) return false
  try {
    const payload = JSON.parse(confirmationPayloadJSON) as Record<string, unknown>
    if (payload.type !== 'mcp_tool' || typeof payload.expires_at !== 'string') return false
    const expiresAt = Date.parse(payload.expires_at)
    return Number.isFinite(expiresAt) && expiresAt <= nowMs
  } catch {
    return false
  }
}

export function isAgentSkillConfirmationExpired(
  expiresAt: string | undefined,
  nowMs = Date.now(),
): boolean {
  if (!expiresAt) return false
  const parsed = Date.parse(expiresAt)
  return Number.isFinite(parsed) && parsed <= nowMs
}

export async function cancelPendingAgentRun(
  runtime: UseHrAgentRunReturn,
  expectedRunId: number,
  clientRequestId = createClientRequestId(),
): Promise<HrAgentRunState> {
  if (!Number.isFinite(expectedRunId) || expectedRunId <= 0) {
    throw new Error(t('common.invalid_request'))
  }
  if (runtime.state.value.runId !== expectedRunId) {
    await runtime.hydrateFromRunId(expectedRunId, { autoSubscribe: false })
  }
  return runtime.cancel({ client_request_id: clientRequestId })
}

export function friendlyDurableRunErrorMessage(errorType = '', errorMessage = ''): string {
  const combined = `${errorType} ${errorMessage}`.toLowerCase()
  if (combined.includes('insufficient_credits')) {
    return insufficientCreditsMessage
  }
  return errorMessage || errorType || '运行失败'
}

export function createClientRequestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `hars-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`
}

const skillRiskLevels = new Set(['low', 'medium', 'high', 'critical'])
const skillActivationPolicies = new Set(['auto', 'confirm', 'manual_only'])
const skillCompositionRoles = new Set(['primary', 'supporting'])

function positiveInteger(value: unknown): number {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 0
}

function safeNumberList(value: unknown): number[] {
  if (!Array.isArray(value)) return []
  return [...new Set(value.map(positiveInteger).filter((id) => id > 0))]
}

function asSkillCandidates(list: AgentRunSkillCandidate[] | undefined): AgentSkillSelectionCandidate[] {
  if (!Array.isArray(list)) return []
  return list
    .map((candidate) => {
      const skillId = positiveInteger(candidate.skill_id)
      const versionId = positiveInteger(candidate.version_id)
      if (!skillId || !versionId) return null
      const risk = skillRiskLevels.has(candidate.risk) ? candidate.risk : 'low'
      const activationPolicy = skillActivationPolicies.has(candidate.activation_policy)
        ? candidate.activation_policy
        : risk === 'critical'
          ? 'manual_only'
          : risk === 'high'
            ? 'confirm'
            : 'auto'
      return {
        skill_id: skillId,
        version_id: versionId,
        version: String(candidate.version || ''),
        compiled_hash: String(candidate.compiled_hash || ''),
        name: String(candidate.name || ''),
        display_name: String(candidate.display_name || candidate.name || ''),
        reason: String(candidate.reason || ''),
        score: Number(candidate.score) || 0,
        priority: Number(candidate.priority) || 0,
        category: String(candidate.category || ''),
        scenario: String(candidate.scenario || ''),
        composition_role: skillCompositionRoles.has(candidate.composition_role)
          ? candidate.composition_role
          : 'primary',
        risk,
        activation_policy: activationPolicy,
        core_estimated_tokens: Math.max(0, Number(candidate.core_estimated_tokens) || 0),
        recommended: Boolean(candidate.recommended),
        vector_score: Number(candidate.vector_score) || 0,
        lexical_score: Number(candidate.lexical_score) || 0,
        metadata_score: Number(candidate.metadata_score) || 0,
        relevance_score: Number(candidate.relevance_score) || 0,
        business_boost: Number(candidate.business_boost) || 0,
        final_rank_score: Number(candidate.final_rank_score) || 0,
        relevance_mode: String(candidate.relevance_mode || ''),
        pool_rank: Number(candidate.pool_rank) || 0,
        ranking_confidence: String(candidate.ranking_confidence || ''),
      } as AgentSkillSelectionCandidate
    })
    .filter((candidate): candidate is AgentSkillSelectionCandidate => Boolean(candidate))
}

/**
 * Normalize durable confirmation payloads into the governed Package v2 UI shape.
 * Agent Skill confirmation uses typed fields. MCP remains an independent opaque payload.
 */
export function toAgentSkillSelectionPayload(
  confirmation: AgentRunConfirmation | null | undefined,
): AgentSkillSelectionPayload | null {
  if (!confirmation) return null

  const cancelOnlyAgentSkillPayload = (): AgentSkillSelectionPayload => {
    const confirmationId = String(confirmation.agent_skill_confirmation_id || '').trim()
    return {
      required: confirmation.required ?? true,
      reason: confirmation.reason || 'Agent Skill 确认信息不完整',
      candidates: [],
      confirmation_kind: 'agent_skill',
      recommended_agent_skill_version_ids: safeNumberList(
        confirmation.recommended_agent_skill_version_ids,
      ),
      ...(confirmationId ? { confirmation_id: confirmationId } : {}),
      ...(confirmation.agent_skill_confirmation_expires_at
        ? { expires_at: confirmation.agent_skill_confirmation_expires_at }
        : {}),
    }
  }

  if (confirmation.candidates && confirmation.candidates.length > 0) {
    const candidates = asSkillCandidates(confirmation.candidates)
    const confirmationId = String(confirmation.agent_skill_confirmation_id || '').trim()
    if (candidates.length === 0 || !confirmationId) return cancelOnlyAgentSkillPayload()
    const candidateVersionIDs = new Set(candidates.map((candidate) => candidate.version_id))
    const recommendedVersionIDs = safeNumberList(
      confirmation.recommended_agent_skill_version_ids,
    ).filter((versionID) => candidateVersionIDs.has(versionID))
    return {
      required: confirmation.required ?? true,
      reason: confirmation.reason || '请确认启用本次 Agent Skill',
      candidates,
      confirmation_kind: 'agent_skill',
      confirmation_id: confirmationId,
      recommended_agent_skill_version_ids: recommendedVersionIDs,
      ...(confirmation.agent_skill_confirmation_expires_at
        ? { expires_at: confirmation.agent_skill_confirmation_expires_at }
        : {}),
    }
  }

  if (confirmation.raw_json) {
    try {
      const parsed = JSON.parse(confirmation.raw_json) as unknown
      if (parsed && typeof parsed === 'object') {
        const mcp = parsed as Record<string, unknown>
        if (
          mcp.type === 'mcp_tool' &&
          typeof mcp.confirmation_id === 'string' &&
          typeof mcp.capability_key === 'string' &&
          typeof mcp.arguments_hash === 'string'
        ) {
          return {
            required: true,
            reason: confirmation.reason || `请确认执行 MCP 能力 ${mcp.capability_key}`,
            candidates: [],
            confirmation_kind: 'mcp_tool',
            recommended_agent_skill_version_ids: [],
            confirmation_payload_json: JSON.stringify({
              type: 'mcp_tool',
              confirmation_id: mcp.confirmation_id,
              capability_key: mcp.capability_key,
              arguments_hash: mcp.arguments_hash,
              ...(typeof mcp.expires_at === 'string' ? { expires_at: mcp.expires_at } : {}),
              approved: true,
            }),
          }
        }
      }
    } catch {
      // fall through
    }
  }

  // Preserve a cancel path for a malformed or partially restored confirmation.
  if (confirmation.required || confirmation.reason) {
    return cancelOnlyAgentSkillPayload()
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
  settlement: 'terminal' | 'waiting_confirmation' | 'timed_out' | 'aborted',
  state: HrAgentRunState,
): DurableChatFlowResult {
  if (settlement === 'aborted') {
    return { outcome: 'aborted', state }
  }
  if (settlement === 'waiting_confirmation') {
    return { outcome: 'waiting_confirmation', state }
  }
  if (settlement === 'timed_out') {
    return {
      outcome: 'failed',
      state,
      error: new Error(t('ai.stream_timeout')),
    }
  }
  if (state.status === 'failed' && (state.errorMessage || state.errorType)) {
    return {
      outcome: 'failed',
      state,
      error: new Error(friendlyDurableRunErrorMessage(state.errorType, state.errorMessage)),
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
      error: error instanceof Error ? error : new Error(t('ai.stream_failed')),
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
      error: error instanceof Error ? error : new Error(t('ai.stream_failed')),
    }
  } finally {
    stop()
  }
}
