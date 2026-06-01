import { ElMessage } from 'element-plus'
import router from '@/router'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import { BusinessError } from '@/types/api'
import type { StreamHandlers, StreamPayload, ChatSessionListItem } from '@/types/ai'
import request from './request'
import { silentRefresh } from './authRefresh'

export const sendMessage = (data: { message: string; application_id?: number; session_id?: number }): Promise<{
  reply: string
  created_at: string
  action?: string
  application_id?: number
  action_status?: number
  candidate_name?: string
  job_title?: string
  status?: number
  session_id?: number
}> => request.post('/api/v1/hr/ai/chat', data)

export const getHistory = (params: { page: number; page_size: number }): Promise<{
  list: { role: string; content: string; created_at: string }[]
}> => request.get('/api/v1/hr/ai/history', { params })

export const analyzeApplication = (data: { application_id: number }): Promise<{
  reply: string
  candidate_name: string
  job_title: string
  status: number
  round_no: number
}> => request.post('/api/v1/hr/ai/analyze-application', data)

export const listSessions = (params: { page: number; page_size: number }): Promise<{
  total: number
  list: ChatSessionListItem[]
}> => request.get('/api/v1/hr/ai/sessions', { params })

export const createSession = (data: { title?: string }): Promise<{
  session: ChatSessionListItem
}> => request.post('/api/v1/hr/ai/sessions', data)

export const getSessionMessages = (sessionId: number, params: { page: number; page_size: number }): Promise<{
  list: { role: string; content: string; created_at: string }[]
}> => request.get(`/api/v1/hr/ai/sessions/${sessionId}/messages`, { params })

export const createApplicationAnalysisSession = (data: { application_id: number }): Promise<{
  session: ChatSessionListItem
  messages: { role: string; content: string; created_at: string }[]
}> => request.post('/api/v1/hr/ai/application-analysis-sessions', data)

export const updateSession = (sessionId: number, data: { title: string }): Promise<void> =>
  request.put(`/api/v1/hr/ai/sessions/${sessionId}`, data)

export const deleteSession = (sessionId: number): Promise<void> =>
  request.delete(`/api/v1/hr/ai/sessions/${sessionId}`)

const friendlyStreamMsg = (code: number, msg: string): string => {
  if (code === 42901) return msg || '今日 AI 使用次数已达上限，请明天再试'
  if (code === 42902) return msg || 'AI 请求太频繁，请稍后再试'
  if (code === 429) return msg || '请求过于频繁，请稍后再试'
  return msg || 'AI 服务响应错误'
}

const streamError = (code: number, msg: string, _requestId?: string): void => {
  if (code === 401) {
    clearLocalAuthCache()
    useAuthStore().$reset()
    router.push('/login')
  }
  ElMessage.error(friendlyStreamMsg(code, msg))
}

const parseSSEBlock = (block: string): string => block
  .split('\n')
  .map((line) => line.trimEnd())
  .filter((line) => line.startsWith('data:'))
  .map((line) => line.slice(5).trimStart())
  .join('\n')

const handleStreamPayload = (text: string, handlers: StreamHandlers): boolean => {
  if (!text) return false
  try {
    const payload: StreamPayload = JSON.parse(text)
    if (payload.code && payload.code !== 0) {
      handlers.onError?.(String(payload.code), payload.msg || 'AI 服务响应错误', payload)
      streamError(payload.code, payload.msg || 'AI 服务响应错误', payload.request_id)
      return true
    }
    // Phase 4: status/error events
    if (payload.event_type && payload.event_type === 'error') {
      handlers.onError?.(payload.error_type || '', payload.event_message || payload.msg || '', payload)
    }
    if (payload.event_type && payload.event_message && !payload.delta) {
      handlers.onStatus?.(payload.event_type, payload.event_message, payload)
    }
    if (payload.delta) {
      handlers.onDelta?.(payload.delta, payload)
    }
    if (payload.done) {
      handlers.onDone?.(payload)
      return true
    }
  } catch {
    // ignore incomplete or malformed SSE blocks
  }
  return false
}

export const sendMessageStream = async (
  data: { message: string; application_id?: number; session_id?: number },
  handlers: StreamHandlers = {},
  options: { signal?: AbortSignal; silentAbort?: boolean } = {},
): Promise<void> => {
  const timeoutController = new AbortController()
  const timeoutId = setTimeout(() => timeoutController.abort(), 180_000)

  // Track whether the abort came from the user's signal vs the timeout controller.
  let wasUserAbort = false

  // Merge external signal with timeout controller: when external fires, timeout cancels too.
  let signal: AbortSignal = timeoutController.signal
  if (options.signal) {
    if (options.signal.aborted) {
      wasUserAbort = true
      clearTimeout(timeoutId)
      return
    }
    const merged = new AbortController()
    const onExternalAbort = () => { wasUserAbort = true; merged.abort() }
    const onTimeoutAbort = () => merged.abort()
    options.signal.addEventListener('abort', onExternalAbort, { once: true })
    timeoutController.signal.addEventListener('abort', onTimeoutAbort, { once: true })
    merged.signal.addEventListener('abort', () => {
      options.signal!.removeEventListener('abort', onExternalAbort)
      timeoutController.signal.removeEventListener('abort', onTimeoutAbort)
      clearTimeout(timeoutId)
    }, { once: true })
    signal = merged.signal
  }

  try {
    const fetchStream = () => fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/hr/ai/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Client-App': 'hr' },
      body: JSON.stringify(data),
      signal,
      credentials: 'include',
    })
    let response = await fetchStream()
    if (response.status === 401) {
      try {
        await silentRefresh('hr')
        response = await fetchStream()
      } catch {
        handlers.onError?.('401', '登录状态已失效，请重新登录', { code: 401, msg: '登录状态已失效，请重新登录' })
        streamError(401, '登录状态已失效，请重新登录')
        return
      }
    }

    if (!response.ok) {
      try {
        const errorText = await response.text()
        const errorJson: StreamPayload = JSON.parse(errorText)
        if (errorJson.code) {
          handlers.onError?.(String(errorJson.code), errorJson.msg || '', errorJson)
          streamError(errorJson.code, errorJson.msg || 'AI 服务请求失败，请稍后重试')
        } else {
          streamError(response.status, 'AI 服务请求失败，请稍后重试')
        }
      } catch {
        streamError(response.status, 'AI 服务请求失败，请稍后重试')
      }
      return
    }

    if (!response.headers.get('content-type')?.includes('text/event-stream')) {
      const text = await response.text()
      try {
        const json: StreamPayload = JSON.parse(text)
        streamError(json.code || 500, json.msg || '响应数据格式异常', json.request_id)
      } catch {
        streamError(500, '响应数据格式异常')
      }
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      streamError(500, '流式响应不可用')
      return
    }

    const decoder = new TextDecoder()
    let buffer = ''
    let shouldStop = false

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const blocks = buffer.split(/\r?\n\r?\n/)
      buffer = blocks.pop() || ''

      for (const block of blocks) {
        if (handleStreamPayload(parseSSEBlock(block), handlers)) {
          shouldStop = true
          break
        }
      }
      if (shouldStop) break
    }

    if (!shouldStop && buffer.trim()) {
      handleStreamPayload(parseSSEBlock(buffer), handlers)
    }
  } catch (error: unknown) {
    if (error instanceof Error && error.name === 'AbortError') {
      if (wasUserAbort) return
      streamError(504, 'AI 服务响应超时，请稍后重试')
      throw new BusinessError(504, 'AI 服务响应超时，请稍后重试')
    } else if (error instanceof Error) {
      streamError(500, error.message || '流式请求失败')
      throw error
    } else {
      streamError(500, '流式请求失败')
      throw new Error('流式请求失败')
    }
  } finally {
    clearTimeout(timeoutId)
  }
}
