import type {
  AgentRunEvent,
  AgentRunEventHandlers,
  CancelAgentRunRequest,
  CancelAgentRunResponse,
  ConfirmAgentRunRequest,
  ConfirmAgentRunResponse,
  CreateAgentRunRequest,
  CreateAgentRunResponse,
  GetActiveAgentRunResponse,
  GetAgentRunResponse,
} from '@shared/types/agentRun'
import { BusinessError } from '@/types/api'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import { ElMessage } from 'element-plus'
import request from './request'
import { silentRefresh } from './authRefresh'
import { contextGuardCodeFrom, contextGuardMessage } from '@/utils/contextUsage'
import { t } from '@shared/i18n'

const RUNS_BASE = '/api/v1/hr/ai/runs'

export const createAgentRun = (data: CreateAgentRunRequest): Promise<CreateAgentRunResponse> =>
  request.post(RUNS_BASE, data)

export const getAgentRun = (runId: number): Promise<GetAgentRunResponse> =>
  request.get(`${RUNS_BASE}/${runId}`)

export const getActiveAgentRun = (
  sessionId: number,
  options?: { silentError?: boolean },
): Promise<GetActiveAgentRunResponse> =>
  request.get(`/api/v1/hr/ai/sessions/${sessionId}/active-run`, {
    silentError: options?.silentError,
  })

export const cancelAgentRun = (
  runId: number,
  data: CancelAgentRunRequest = {},
): Promise<CancelAgentRunResponse> =>
  request.post(`${RUNS_BASE}/${runId}/cancel`, data)

export const confirmAgentRun = (
  runId: number,
  data: ConfirmAgentRunRequest = {},
): Promise<ConfirmAgentRunResponse> =>
  request.post(`${RUNS_BASE}/${runId}/confirm`, data)

const parseSSEBlock = (block: string): { id?: string; data: string } => {
  let id: string | undefined
  const dataLines: string[] = []
  for (const rawLine of block.split('\n')) {
    const line = rawLine.trimEnd()
    if (line.startsWith('id:')) {
      id = line.slice(3).trimStart()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  return { id, data: dataLines.join('\n') }
}

export const friendlyAgentRunStreamMsg = (code: number, msg: string): string => {
  const guardMessage = contextGuardMessage(contextGuardCodeFrom(msg))
  if (guardMessage) return guardMessage
  if (code === 40201) return msg || t('ai.insufficient_credits')
  if (code === 42901) return msg || t('ai.daily_quota_exceeded')
  if (code === 42902) return msg || t('ai.too_many_requests')
  if (code === 429) return msg || t('common.too_many_requests')
  return msg || t('ai.unavailable')
}

export const normalizeAgentRunEvent = (event: AgentRunEvent): AgentRunEvent => {
  if (!event.payload_json) {
    return event
  }
  try {
    const payload = JSON.parse(event.payload_json) as {
      event_message?: string
      display_message?: string
      display_source?: string
      step_key?: string
      step_purpose?: string
      tool_group?: string
      status?: string
      delta?: string
      snapshot_text?: string
      tool_name?: string
      error_type?: string
      error_message?: string
      result_metadata?: AgentRunEvent['result_metadata']
    }
    const normalized: AgentRunEvent = {
      ...event,
      ...(payload.event_message && !event.event_message ? { event_message: payload.event_message } : {}),
      ...(payload.display_message && !event.display_message ? { display_message: payload.display_message } : {}),
      ...(payload.display_source && !event.display_source ? { display_source: payload.display_source } : {}),
      ...(payload.step_key && !event.step_key ? { step_key: payload.step_key } : {}),
      ...(payload.step_purpose && !event.step_purpose ? { step_purpose: payload.step_purpose } : {}),
      ...(payload.tool_group && !event.tool_group ? { tool_group: payload.tool_group } : {}),
      ...(payload.status && !event.status ? { status: payload.status } : {}),
      ...(payload.delta && !event.delta ? { delta: payload.delta } : {}),
      ...(payload.snapshot_text && !event.snapshot_text ? { snapshot_text: payload.snapshot_text } : {}),
      ...(payload.tool_name && !event.tool_name ? { tool_name: payload.tool_name } : {}),
      ...(payload.error_type && !event.error_type ? { error_type: payload.error_type } : {}),
      ...(payload.error_message && !event.error_message ? { error_message: payload.error_message } : {}),
      ...(payload.result_metadata && !event.result_metadata ? { result_metadata: payload.result_metadata } : {}),
    }
    return normalized
  } catch {
    return event
  }
}

/**
 * Subscribe to durable run events via SSE.
 * AbortSignal cancels only this subscription fetch — never the backend run.
 */
export const subscribeAgentRunEvents = async (
  runId: number,
  afterSeq: number,
  handlers: AgentRunEventHandlers = {},
  options: { signal?: AbortSignal; lastEventId?: string | number } = {},
): Promise<void> => {
  if (!Number.isFinite(runId) || runId <= 0) {
    throw new Error(t('validation.invalid_id'))
  }
  const seq = Number.isFinite(afterSeq) && afterSeq > 0 ? Math.floor(afterSeq) : 0
  const query = seq > 0 ? `?after_seq=${encodeURIComponent(String(seq))}` : ''
  const url = `${import.meta.env.VITE_API_BASE_URL || ''}${RUNS_BASE}/${runId}/events${query}`

  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'X-Client-App': 'hr',
  }
  if (options.lastEventId !== undefined && options.lastEventId !== '') {
    headers['Last-Event-ID'] = String(options.lastEventId)
  } else if (seq > 0) {
    // Prefer after_seq query; also set Last-Event-ID for proxies that strip query on reconnect.
    headers['Last-Event-ID'] = String(seq)
  }

  const fetchStream = () =>
    fetch(url, {
      method: 'GET',
      headers,
      signal: options.signal,
      credentials: 'include',
    })

  try {
    let response = await fetchStream()
    if (response.status === 401) {
      try {
        await silentRefresh('hr')
        response = await fetchStream()
      } catch {
        handlers.onError?.({ code: 401, message: t('common.unauthenticated') })
        clearLocalAuthCache()
        useAuthStore().$reset()
        router.push('/login')
        ElMessage.error(t('common.unauthenticated'))
        handlers.onDone?.()
        return
      }
    }

    if (!response.ok) {
      let message = t('frontend.subscribe_failed')
      let code = response.status
      try {
        const errorText = await response.text()
        const errorJson = JSON.parse(errorText) as { code?: number; msg?: string }
        if (errorJson.code) {
          code = errorJson.code
          message = errorJson.msg || message
        }
      } catch {
        // keep defaults
      }
      handlers.onError?.({ code, message })
      if (code === 401) {
        clearLocalAuthCache()
        useAuthStore().$reset()
        router.push('/login')
      }
      ElMessage.error(friendlyAgentRunStreamMsg(code, message))
      handlers.onDone?.()
      return
    }

    if (!response.headers.get('content-type')?.includes('text/event-stream')) {
      const text = await response.text()
      try {
        const json = JSON.parse(text) as { code?: number; msg?: string }
        handlers.onError?.({
          code: json.code || 500,
          message: json.msg || t('ai.invalid_response'),
        })
      } catch {
        handlers.onError?.({ code: 500, message: t('ai.invalid_response') })
      }
      handlers.onDone?.()
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      handlers.onError?.({ code: 500, message: t('ai.stream_unavailable') })
      handlers.onDone?.()
      return
    }

    const decoder = new TextDecoder()
    let buffer = ''
    let shouldStop = false

    const handleBlock = (block: string): boolean => {
      const { id, data: text } = parseSSEBlock(block)
      if (!text) return false
      try {
        const payload = normalizeAgentRunEvent(JSON.parse(text) as AgentRunEvent)
        if (payload.code && payload.code !== 0) {
          handlers.onError?.({
            code: payload.code,
            message: payload.msg || t('ai.unavailable'),
            event: payload,
          })
          if (payload.done) {
            return true
          }
          return false
        }
        if (id && (payload.seq === undefined || payload.seq === null)) {
          const parsedSeq = Number(id)
          if (Number.isFinite(parsedSeq)) {
            payload.seq = parsedSeq
          }
        }
        handlers.onEvent?.(payload)
        if (payload.done) {
          return true
        }
        // Terminal durable event types end the subscription naturally on the server;
        // keep reading until stream closes so late events are not dropped.
      } catch {
        // ignore incomplete or malformed SSE blocks
      }
      return false
    }

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const blocks = buffer.split(/\r?\n\r?\n/)
      buffer = blocks.pop() || ''

      for (const block of blocks) {
        if (handleBlock(block)) {
          shouldStop = true
          break
        }
      }
      if (shouldStop) break
    }

    if (!shouldStop && buffer.trim()) {
      handleBlock(buffer)
    }
    handlers.onDone?.()
  } catch (error: unknown) {
    if (error instanceof Error && error.name === 'AbortError') {
      // Subscription abort only — do not treat as run cancel.
      handlers.onDone?.()
      return
    }
    if (error instanceof Error) {
      handlers.onError?.({ message: error.message || t('frontend.subscribe_failed') })
      throw error
    }
    handlers.onError?.({ message: t('frontend.subscribe_failed') })
    throw new BusinessError(500, t('frontend.subscribe_failed'))
  }
}
