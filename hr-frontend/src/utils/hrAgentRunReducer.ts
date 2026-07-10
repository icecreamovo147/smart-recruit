/**
 * Framework-independent pure reducer for durable HR Agent run events.
 * Used by Vue composables; not a global store and not Redux.
 */

import type {
  AgentRunConfirmation,
  AgentRunEvent,
  AgentRunResultMetadata,
  AgentRunSnapshot,
  AgentRunStatus,
} from '@/types/agentRun'
import { isTerminalAgentRunStatus } from '@/types/agentRun'

export interface HrAgentRunState {
  runId: number | null
  sessionId: number | null
  status: AgentRunStatus | string
  assistantText: string
  processText: string
  lastEventSeq: number
  confirmation: AgentRunConfirmation | null
  resultMetadata: AgentRunResultMetadata | null
  errorType: string
  errorMessage: string
  isTerminal: boolean
  clientRequestId: string
  messageId: number | null
  historyId: number | null
  modelId: number | null
  modelName: string
  agentType: string
  agentId: number | null
  agentName: string
  lastToolName: string
  optionContextJson: string
  startedAt: string
  completedAt: string
  updatedAt: string
}

export function createInitialHrAgentRunState(
  partial: Partial<HrAgentRunState> = {},
): HrAgentRunState {
  return {
    runId: null,
    sessionId: null,
    status: '',
    assistantText: '',
    processText: '',
    lastEventSeq: 0,
    confirmation: null,
    resultMetadata: null,
    errorType: '',
    errorMessage: '',
    isTerminal: false,
    clientRequestId: '',
    messageId: null,
    historyId: null,
    modelId: null,
    modelName: '',
    agentType: '',
    agentId: null,
    agentName: '',
    lastToolName: '',
    optionContextJson: '',
    startedAt: '',
    completedAt: '',
    updatedAt: '',
    ...partial,
  }
}

export function hydrateFromSnapshot(snapshot: AgentRunSnapshot | null | undefined): HrAgentRunState {
  if (!snapshot || !snapshot.run_id) {
    return createInitialHrAgentRunState()
  }
  const status = snapshot.status || ''
  return createInitialHrAgentRunState({
    runId: snapshot.run_id,
    sessionId: snapshot.session_id ?? null,
    status,
    assistantText: snapshot.assistant_text || '',
    processText: snapshot.process_text || '',
    lastEventSeq: Number(snapshot.last_event_seq) || 0,
    confirmation: snapshot.confirmation_request ?? null,
    resultMetadata: snapshot.result_metadata ?? null,
    errorType: snapshot.error_type || '',
    errorMessage: snapshot.error_message || '',
    isTerminal: isTerminalAgentRunStatus(status),
    clientRequestId: snapshot.client_request_id || '',
    messageId: snapshot.message_id ?? null,
    historyId: snapshot.history_id ?? null,
    modelId: snapshot.model_id ?? null,
    modelName: snapshot.model_name || '',
    agentType: snapshot.agent_type || '',
    agentId: snapshot.agent_id ?? null,
    agentName: snapshot.agent_name || '',
    optionContextJson: snapshot.option_context_json || '',
    startedAt: snapshot.started_at || '',
    completedAt: snapshot.completed_at || '',
    updatedAt: snapshot.updated_at || '',
  })
}

function parseConfirmationFromPayload(event: AgentRunEvent): AgentRunConfirmation | null {
  if (event.confirmation) {
    return event.confirmation
  }
  const raw = event.payload_json
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as AgentRunConfirmation
    if (parsed && typeof parsed === 'object') {
      return parsed
    }
  } catch {
    return { required: true, raw_json: raw }
  }
  return null
}

function parseResultMetadata(event: AgentRunEvent): AgentRunResultMetadata | null {
  if (event.result_metadata) {
    return event.result_metadata
  }
  const raw = event.payload_json
  if (!raw) return null
  try {
    return JSON.parse(raw) as AgentRunResultMetadata
  } catch {
    return { raw_json: raw }
  }
}

function withStatus(state: HrAgentRunState, status: string): HrAgentRunState {
  if (!status) return state
  return {
    ...state,
    status,
    isTerminal: isTerminalAgentRunStatus(status),
  }
}

/**
 * Apply a single durable run event.
 * Ignores events for a different run_id and duplicate/stale seq values.
 */
export function reduceAgentRunEvent(
  state: HrAgentRunState,
  event: AgentRunEvent | null | undefined,
): HrAgentRunState {
  if (!event || !event.event_type) {
    return state
  }

  // Transport-level error envelopes are not durable seq events.
  if (event.code && event.code !== 0 && !event.event_type.startsWith('run.')) {
    return state
  }

  const eventRunId = Number(event.run_id)
  if (state.runId != null && Number.isFinite(eventRunId) && eventRunId > 0 && eventRunId !== state.runId) {
    return state
  }

  const seq = Number(event.seq)
  if (Number.isFinite(seq) && seq > 0 && seq <= state.lastEventSeq) {
    return state
  }

  let next: HrAgentRunState = {
    ...state,
    runId: state.runId ?? (Number.isFinite(eventRunId) && eventRunId > 0 ? eventRunId : state.runId),
    lastEventSeq:
      Number.isFinite(seq) && seq > state.lastEventSeq ? seq : state.lastEventSeq,
  }

  if (event.created_at) {
    next = { ...next, updatedAt: event.created_at }
  }

  switch (event.event_type) {
    case 'run.created': {
      if (event.status) next = withStatus(next, event.status)
      else if (!next.status) next = withStatus(next, 'queued')
      return next
    }
    case 'run.status_changed': {
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'assistant.delta': {
      const delta = event.delta ?? ''
      if (delta) {
        next = { ...next, assistantText: next.assistantText + delta }
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'assistant.snapshot': {
      const text = event.snapshot_text ?? ''
      next = { ...next, assistantText: text }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'process.delta': {
      const delta = event.delta ?? ''
      if (delta) {
        next = { ...next, processText: next.processText + delta }
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'process.snapshot': {
      const text = event.snapshot_text ?? ''
      next = { ...next, processText: text }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'tool.started': {
      next = {
        ...next,
        lastToolName: event.tool_name || next.lastToolName,
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'tool.finished': {
      next = {
        ...next,
        lastToolName: event.tool_name || next.lastToolName,
      }
      if (event.error_type || event.error_message) {
        // Tool-level errors do not necessarily fail the run; record lightly.
        next = {
          ...next,
          errorType: event.error_type || next.errorType,
          errorMessage: event.error_message || next.errorMessage,
        }
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'confirmation.required': {
      next = {
        ...next,
        confirmation: parseConfirmationFromPayload(event),
      }
      next = withStatus(next, event.status || 'waiting_confirmation')
      return next
    }
    case 'confirmation.accepted': {
      next = {
        ...next,
        confirmation: event.confirmation ?? next.confirmation,
      }
      if (event.status) next = withStatus(next, event.status)
      else if (next.status === 'waiting_confirmation') {
        next = withStatus(next, 'running')
      }
      return next
    }
    case 'run.result': {
      next = {
        ...next,
        resultMetadata: parseResultMetadata(event) ?? next.resultMetadata,
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'run.error': {
      next = {
        ...next,
        errorType: event.error_type || next.errorType,
        errorMessage: event.error_message || next.errorMessage,
      }
      next = withStatus(next, event.status || 'failed')
      return next
    }
    case 'run.completed': {
      const terminalStatus = event.status || (next.status && isTerminalAgentRunStatus(next.status) ? next.status : 'succeeded')
      next = withStatus(next, terminalStatus)
      if (event.result_metadata) {
        next = { ...next, resultMetadata: event.result_metadata }
      }
      if (event.error_type || event.error_message) {
        next = {
          ...next,
          errorType: event.error_type || next.errorType,
          errorMessage: event.error_message || next.errorMessage,
        }
      }
      return next
    }
    case 'run.canceled': {
      next = withStatus(next, event.status || 'canceled')
      if (event.error_type || event.error_message) {
        next = {
          ...next,
          errorType: event.error_type || next.errorType,
          errorMessage: event.error_message || next.errorMessage,
        }
      }
      return next
    }
    case 'run.heartbeat': {
      // Keep seq / liveness only.
      return next
    }
    default: {
      // Unknown event types still advance seq to avoid reprocessing gaps incorrectly.
      if (event.status) next = withStatus(next, event.status)
      if (event.delta) {
        // Do not guess which buffer; leave text unchanged.
      }
      return next
    }
  }
}

export function applyAgentRunEvents(
  state: HrAgentRunState,
  events: readonly AgentRunEvent[],
): HrAgentRunState {
  let current = state
  for (const event of events) {
    current = reduceAgentRunEvent(current, event)
  }
  return current
}
