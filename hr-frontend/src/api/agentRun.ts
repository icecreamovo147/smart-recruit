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
} from '@/types/agentRun'
import { BusinessError } from '@/types/api'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import { ElMessage } from 'element-plus'
import request from './request'
import { silentRefresh } from './authRefresh'

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

const friendlyStreamMsg = (code: number, msg: string): string => {
  if (code === 42901) return msg || '今日 AI 使用次数已达上限，请明天再试'
  if (code === 42902) return msg || 'AI 请求太频繁，请稍后再试'
  if (code === 429) return msg || '请求过于频繁，请稍后再试'
  return msg || 'AI 服务响应错误'
}

export const normalizeAgentRunEvent = (event: AgentRunEvent): AgentRunEvent => {
  if (!event.payload_json) {
    if (event.event_type === 'process.delta' && event.result_metadata) {
      return { ...event, event_type: 'run.result' }
    }
    return event
  }
  try {
    const payload = JSON.parse(event.payload_json) as {
      event_message?: string
      result_metadata?: AgentRunEvent['result_metadata']
    }
    const normalized: AgentRunEvent = {
      ...event,
      ...(payload.event_message && !event.event_message ? { event_message: payload.event_message } : {}),
      ...(payload.result_metadata && !event.result_metadata ? { result_metadata: payload.result_metadata } : {}),
    }
    if (
      event.event_type === 'process.delta' &&
      normalized.result_metadata
    ) {
      return {
        ...normalized,
        event_type: 'run.result',
      }
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
    throw new Error('runId must be a positive number')
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
        handlers.onError?.({ code: 401, message: '登录状态已失效，请重新登录' })
        clearLocalAuthCache()
        useAuthStore().$reset()
        router.push('/login')
        ElMessage.error('登录状态已失效，请重新登录')
        handlers.onDone?.()
        return
      }
    }

    if (!response.ok) {
      let message = '订阅 Agent 运行事件失败，请稍后重试'
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
      ElMessage.error(friendlyStreamMsg(code, message))
      handlers.onDone?.()
      return
    }

    if (!response.headers.get('content-type')?.includes('text/event-stream')) {
      const text = await response.text()
      try {
        const json = JSON.parse(text) as { code?: number; msg?: string }
        handlers.onError?.({
          code: json.code || 500,
          message: json.msg || '响应数据格式异常',
        })
      } catch {
        handlers.onError?.({ code: 500, message: '响应数据格式异常' })
      }
      handlers.onDone?.()
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      handlers.onError?.({ code: 500, message: '流式响应不可用' })
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
            message: payload.msg || 'AI 服务响应错误',
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
      handlers.onError?.({ message: error.message || '订阅失败' })
      throw error
    }
    handlers.onError?.({ message: '订阅失败' })
    throw new BusinessError(500, '订阅失败')
  }
}
