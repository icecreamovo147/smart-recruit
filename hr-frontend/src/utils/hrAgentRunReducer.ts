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
} from '@shared/types/agentRun'
import { isTerminalAgentRunStatus } from '@shared/types/agentRun'

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

function modelPatchFromMetadata(
  metadata: AgentRunResultMetadata | null | undefined,
): Partial<Pick<HrAgentRunState, 'modelId' | 'modelName'>> {
  const usage = metadata?.context_usage
  if (!usage) return {}
  const patch: Partial<Pick<HrAgentRunState, 'modelId' | 'modelName'>> = {}
  if (typeof usage.model_id === 'number' && Number.isFinite(usage.model_id)) {
    patch.modelId = usage.model_id
  }
  if (typeof usage.model_name === 'string' && usage.model_name.trim()) {
    patch.modelName = usage.model_name.trim()
  }
  return patch
}

function withStatus(state: HrAgentRunState, status: string): HrAgentRunState {
  if (!status) return state
  return {
    ...state,
    status,
    isTerminal: isTerminalAgentRunStatus(status),
  }
}

function appendProcessLine(text: string, line: string): string {
  if (!line) return text
  const separator = text && !text.endsWith('\n') ? '\n' : ''
  return `${text}${separator}${line}`
}

function processMessageFromEvent(event: AgentRunEvent, phase: 'process' | 'tool.started' | 'tool.finished'): string {
  if (event.display_message) {
    return event.display_message
  }
  const message = event.event_message || ''
  if (phase === 'tool.started') {
    return '我正在查询实时招聘数据。'
  }
  if (phase === 'tool.finished') {
    if (event.error_type || event.error_message || /\bnot found\b/i.test(message)) {
      return '这一步实时数据暂时没有完成，我会尝试使用其他可用数据继续推进。'
    }
    return '已获取一项实时招聘数据。'
  }
  if (message === 'planning HR recruiting context') {
    return '我正在判断问题意图，并规划需要读取哪些招聘数据。'
  }
  if (message === 'context usage estimated') {
    return '我已确认上下文容量，准备整理工具结果。'
  }
  if (message === '正在分析问题...') {
    return '我正在理解问题，并准备整理已查到的数据。'
  }
  if (message === 'AI 响应较慢，请稍候...') {
    return '模型响应稍慢，我还在等待它基于工具结果继续推理。'
  }
  if (message === '正在生成回答...') {
    return '我正在整理已获取的数据并生成回复。'
  }
  if (message === '回答完成') {
    return '分析完成，已生成基于真实数据的回复。'
  }
  return message
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
      if (event.delta) {
        next = { ...next, processText: next.processText + event.delta }
      } else if (event.event_message) {
        next = { ...next, processText: appendProcessLine(next.processText, processMessageFromEvent(event, 'process')) }
      }
      if (event.result_metadata) {
        next = {
          ...next,
          resultMetadata: event.result_metadata,
          ...modelPatchFromMetadata(event.result_metadata),
        }
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
      const processText = processMessageFromEvent(event, 'tool.started')
      next = {
        ...next,
        lastToolName: event.tool_name || next.lastToolName,
        ...(processText ? { processText: appendProcessLine(next.processText, processText) } : {}),
      }
      if (event.status) next = withStatus(next, event.status)
      return next
    }
    case 'tool.finished': {
      const processText = processMessageFromEvent(event, 'tool.finished')
      next = {
        ...next,
        lastToolName: event.tool_name || next.lastToolName,
        ...(processText ? { processText: appendProcessLine(next.processText, processText) } : {}),
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
      const metadata = parseResultMetadata(event) ?? next.resultMetadata
      next = {
        ...next,
        resultMetadata: metadata,
        ...modelPatchFromMetadata(metadata),
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
        next = {
          ...next,
          resultMetadata: event.result_metadata,
          ...modelPatchFromMetadata(event.result_metadata),
        }
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
