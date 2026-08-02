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
  return typeof value === 'number' && Number.isInteger(value) && value > 0 ? value : 0
}

function exactPositiveIntegerList(value: unknown): number[] | null {
  if (!Array.isArray(value) || value.length === 0) return null
  const exact: number[] = []
  const seen = new Set<number>()
  for (const item of value) {
    if (typeof item !== 'number' || !Number.isInteger(item) || item <= 0 || seen.has(item)) {
      return null
    }
    seen.add(item)
    exact.push(item)
  }
  return exact
}

function asSkillCandidates(list: AgentRunSkillCandidate[] | undefined): AgentSkillSelectionCandidate[] {
  if (!Array.isArray(list)) return []
  return list
    .map((candidate) => {
      const skillId = positiveInteger(candidate.skill_id)
      const versionId = positiveInteger(candidate.version_id)
      const version = typeof candidate.version === 'string' ? candidate.version.trim() : ''
      const compiledHash = typeof candidate.compiled_hash === 'string' ? candidate.compiled_hash.trim() : ''
      const name = typeof candidate.name === 'string' ? candidate.name.trim() : ''
      const displayName = typeof candidate.display_name === 'string' ? candidate.display_name.trim() : ''
      const risk = candidate.risk
      const activationPolicy = candidate.activation_policy
      const compositionRole = candidate.composition_role
      const coreEstimatedTokens = candidate.core_estimated_tokens
      if (
        !skillId
        || !versionId
        || !version
        || !compiledHash
        || (!name && !displayName)
        || !skillRiskLevels.has(risk)
        || !skillActivationPolicies.has(activationPolicy)
        || !skillCompositionRoles.has(compositionRole)
        || typeof coreEstimatedTokens !== 'number'
        || !Number.isFinite(coreEstimatedTokens)
        || coreEstimatedTokens < 0
        || candidate.recommended !== true
      ) {
        return null
      }
      return {
        skill_id: skillId,
        version_id: versionId,
        version,
        compiled_hash: compiledHash,
        name,
        display_name: displayName || name,
        reason: String(candidate.reason || ''),
        score: Number(candidate.score) || 0,
        priority: Number(candidate.priority) || 0,
        category: String(candidate.category || ''),
        scenario: String(candidate.scenario || ''),
        composition_role: compositionRole,
        risk,
        activation_policy: activationPolicy,
        core_estimated_tokens: coreEstimatedTokens,
        recommended: true,
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
      recommended_agent_skill_version_ids: [],
      ...(confirmationId ? { confirmation_id: confirmationId } : {}),
      ...(confirmation.agent_skill_confirmation_expires_at
        ? { expires_at: confirmation.agent_skill_confirmation_expires_at }
        : {}),
    }
  }

  if (confirmation.candidates && confirmation.candidates.length > 0) {
    const candidates = asSkillCandidates(confirmation.candidates)
    const confirmationId = String(confirmation.agent_skill_confirmation_id || '').trim()
    const recommendedVersionIDs = exactPositiveIntegerList(
      confirmation.recommended_agent_skill_version_ids,
    )
    const candidateVersionIDs = candidates.map((candidate) => candidate.version_id)
    const candidateVersionIDSet = new Set(candidateVersionIDs)
    if (
      !confirmationId
      || !recommendedVersionIDs
      || candidates.length !== confirmation.candidates.length
      || candidateVersionIDSet.size !== candidateVersionIDs.length
      || candidateVersionIDs.length !== recommendedVersionIDs.length
      || recommendedVersionIDs.some((versionID) => !candidateVersionIDSet.has(versionID))
    ) {
      return cancelOnlyAgentSkillPayload()
    }
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
