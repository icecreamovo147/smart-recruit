import { ElMessage } from 'element-plus'
import router from '@/router'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import { BusinessError } from '@/types/api'
import type { StreamHandlers, StreamPayload, CandidateSession } from '@/types/ai'
import request from './request'
import { silentRefresh } from './authRefresh'

export interface CandidateAISessionSource {
  session_type?: string
  source_type?: string
  source_id?: number
  source_title?: string
}

export const sendMessage = (data: { message: string; session_id?: number; model_id?: number } & CandidateAISessionSource): Promise<{
  reply: string
  created_at: string
  session_id?: number
  action?: string
  action_payload?: string
  suggested_questions?: string[] | string
  suggestedQuestions?: string[] | string
}> => request.post('/api/v1/candidate/ai/chat', data)

export const listSessions = (params: {
  page: number
  page_size: number
  keyword?: string
  session_type?: string
  source_type?: string
  source_id?: number
}): Promise<{
  total: number
  list: CandidateSession[]
}> => request.get('/api/v1/candidate/ai/sessions', { params })

export const createSession = (data: { title?: string; initial_message?: string } & CandidateAISessionSource): Promise<{
  session: CandidateSession
}> => request.post('/api/v1/candidate/ai/sessions', data)

export const getSessionMessages = (sessionId: number, params: { page: number; page_size: number }): Promise<{
  list: {
    role: string
    content: string
    created_at: string
    model_name?: string
    process_content?: string
    suggested_questions?: string[] | string
    suggestedQuestions?: string[] | string
  }[]
}> => request.get(`/api/v1/candidate/ai/sessions/${sessionId}/messages`, { params })

export const updateSession = (sessionId: number, data: { title: string }): Promise<void> =>
  request.put(`/api/v1/candidate/ai/sessions/${sessionId}`, data)

export const deleteSession = (sessionId: number): Promise<void> =>
  request.delete(`/api/v1/candidate/ai/sessions/${sessionId}`)

const friendlyStreamMsg = (code: number, msg: string): string => {
  if (code === 40201) return msg || 'AI 套餐额度不足，请购买套餐或加量包后重试'
  if (code === 42901) return msg || '今日 AI 使用次数已达上限，请明天再试'
  if (code === 42902) return msg || 'AI 请求太频繁，请稍后再试'
  if (code === 429) return msg || '请求过于频繁，请稍后再试'
  return msg || 'AI 服务响应错误'
}

const streamError = (code: number, msg: string): void => {
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
      streamError(payload.code, payload.msg || '')
      return true
    }
    // Phase 4: status/error events
    if (payload.event_type && payload.event_type === 'error') {
      handlers.onError?.(payload.error_type || '', payload.event_message || payload.msg || '', payload)
    }
    if (
      payload.event_type
      && !payload.delta
      && (payload.event_message || payload.event_type === 'model_info' || payload.context_usage)
    ) {
      handlers.onStatus?.(payload.event_type, payload.event_message || '', payload)
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
  data: { message: string; session_id?: number; model_id?: number } & CandidateAISessionSource,
  handlers: StreamHandlers = {},
  options: { signal?: AbortSignal; silentAbort?: boolean } = {},
): Promise<void> => {
  const timeoutController = new AbortController()
  const timeoutId = setTimeout(() => timeoutController.abort(), 180_000)

  // Track whether the abort came from the user's signal vs the timeout controller.
  let wasUserAbort = false
  let streamCompleted = false
  let streamErrored = false
  let failureReported = false
  const trackedHandlers: StreamHandlers = {
    ...handlers,
    onDone: (payload) => {
      streamCompleted = true
      handlers.onDone?.(payload)
    },
    onError: (errorType, errorMessage, payload) => {
      streamErrored = true
      handlers.onError?.(errorType, errorMessage, payload)
    },
  }
  const reportFailure = (code: number, message: string, payload?: StreamPayload): void => {
    failureReported = true
    trackedHandlers.onError?.(String(code), message, payload || { code, msg: message })
    streamError(code, message)
  }

  // Merge external signal with timeout controller: when external fires, timeout cancels too.
  let signal: AbortSignal = timeoutController.signal
  if (options.signal) {
    if (options.signal.aborted) {
      wasUserAbort = true
      clearTimeout(timeoutId)
      return
    }
    // Create a merged controller that fires on either signal.
    const merged = new AbortController()
    const onExternalAbort = () => { wasUserAbort = true; merged.abort() }
    const onTimeoutAbort = () => merged.abort()
    options.signal.addEventListener('abort', onExternalAbort, { once: true })
    timeoutController.signal.addEventListener('abort', onTimeoutAbort, { once: true })
    // Cleanup
    merged.signal.addEventListener('abort', () => {
      options.signal!.removeEventListener('abort', onExternalAbort)
      timeoutController.signal.removeEventListener('abort', onTimeoutAbort)
      clearTimeout(timeoutId)
    }, { once: true })
    signal = merged.signal
  }

  try {
    const fetchStream = () => fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/candidate/ai/chat/stream`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Client-App': 'candidate' },
      body: JSON.stringify(data),
      signal,
      credentials: 'include',
    })
    let response = await fetchStream()
    if (response.status === 401) {
      try {
        await silentRefresh('candidate')
        response = await fetchStream()
      } catch {
        reportFailure(401, '登录状态已失效，请重新登录')
        return
      }
    }

    if (!response.ok) {
      try {
        const errorText = await response.text()
        const errorJson: StreamPayload = JSON.parse(errorText)
        if (errorJson.code) {
          reportFailure(errorJson.code, errorJson.msg || 'AI 服务请求失败，请稍后重试', errorJson)
        } else {
          reportFailure(response.status, 'AI 服务请求失败，请稍后重试')
        }
      } catch {
        reportFailure(response.status, 'AI 服务请求失败，请稍后重试')
      }
      return
    }

    if (!response.headers.get('content-type')?.includes('text/event-stream')) {
      const text = await response.text()
      try {
        const json: StreamPayload = JSON.parse(text)
        reportFailure(json.code || 500, json.msg || '响应数据格式异常', json)
      } catch {
        reportFailure(500, '响应数据格式异常')
      }
      return
    }

    const reader = response.body?.getReader()
    if (!reader) {
      reportFailure(500, '流式响应不可用')
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
        if (handleStreamPayload(parseSSEBlock(block), trackedHandlers)) {
          shouldStop = true
          break
        }
      }
      if (shouldStop) break
    }

    if (!shouldStop && buffer.trim()) {
      handleStreamPayload(parseSSEBlock(buffer), trackedHandlers)
    }

    if (!streamCompleted && !streamErrored && !wasUserAbort) {
      const message = 'AI 服务连接已中断，请稍后重试'
      reportFailure(502, message)
      throw new BusinessError(502, message)
    }
  } catch (error: unknown) {
    if (failureReported) throw error
    if (error instanceof Error && error.name === 'AbortError') {
      if (wasUserAbort) return // user-initiated abort, silent
      reportFailure(504, 'AI 服务响应超时，请稍后重试')
      throw new BusinessError(504, 'AI 服务响应超时，请稍后重试')
    } else if (error instanceof Error) {
      reportFailure(500, error.message || '流式请求失败')
      throw error
    } else {
      reportFailure(500, '流式请求失败')
      throw new Error('流式请求失败')
    }
  } finally {
    clearTimeout(timeoutId)
  }
}
