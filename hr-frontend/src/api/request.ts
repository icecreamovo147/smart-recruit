import axios from 'axios'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import { clearLocalAuthCache } from '@/utils/token'
import { BusinessError } from '@/types/api'
import { silentRefresh } from './authRefresh'
import { debugLog } from '@shared/utils/debugLog'
import { contextGuardCodeFrom, contextGuardMessage } from '@/utils/contextUsage'

interface RequestConfig extends AxiosRequestConfig {
  silentError?: boolean
  _startTime?: number
  clientTraceId?: string
}

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 60000,
  withCredentials: true, // Send httpOnly auth cookies for CORS requests
})

// Auth handled via httpOnly Cookie — no Authorization header needed.
http.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'hr')
  const requestConfig = config as RequestConfig
  requestConfig._startTime = Date.now()
  requestConfig.clientTraceId = crypto.randomUUID().slice(0, 8)
  const params = config.params ? { ...config.params } : undefined
  const bodySize = config.data ? JSON.stringify(config.data).length : 0
  debugLog.http.info('request_started', {
    method: config.method?.toUpperCase(),
    url: config.url,
    params: params,
    body_size: bodySize,
    client_trace_id: requestConfig.clientTraceId,
  })
  return config
})

// Silent refresh state — shared across all interceptors.
let isRefreshing = false
let refreshStartTime = 0
const failedRequests: Array<{
  resolve: (value: unknown) => void
  reject: (reason: unknown) => void
  config: RequestConfig
}> = []

http.interceptors.response.use(
  (response) => {
    const { code, msg, data } = response.data
    const requestId = response.data.request_id as string || ''
    const requestConfig = response.config as RequestConfig
    const durationMs = requestConfig._startTime ? Date.now() - requestConfig._startTime : 0
    const logData: Record<string, unknown> = {
      method: response.config?.method?.toUpperCase(),
      url: response.config?.url,
      client_trace_id: requestConfig.clientTraceId || '',
      request_id: requestId?.slice(0, 8) || '',
      business_code: code,
      duration_ms: durationMs,
    }
    if (code !== 0) {
      const error = new BusinessError(code, friendlyBusinessMessage(code, msg), requestId)
      const contextGuardCode = contextGuardCodeFrom(msg)
      if (contextGuardCode) {
        Object.assign(error, { contextGuardCode })
      }
      const resetAt = response.data?.data?.reset_at as string || ''
      const displayMsg = resetAt ? `${error.message}（${formatResetTime(resetAt)}恢复）` : error.message
      if (!(response.config as RequestConfig).silentError) {
        ElMessage.error(requestId ? `${displayMsg} [${requestId.slice(0, 8)}]` : displayMsg)
      }
      debugLog.http.warn('business_error', { ...logData, business_msg: msg })
      return Promise.reject(error)
    }
    debugLog.http.info('response_succeeded', logData)
    return data
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as RequestConfig & { _retry?: boolean }
    const durationMs = originalRequest?._startTime ? Date.now() - originalRequest._startTime : 0
    const logData: Record<string, unknown> = {
      method: originalRequest?.method?.toUpperCase(),
      url: originalRequest?.url,
      client_trace_id: originalRequest?.clientTraceId || '',
      duration_ms: durationMs,
      network_error: error.code,
      silent_error: originalRequest?.silentError,
    }

    // Attempt silent refresh on 401 for non-refresh requests.
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      if (!isRefreshing) {
        isRefreshing = true
        refreshStartTime = Date.now()
        debugLog.http.info('refresh_started', { method: originalRequest.method?.toUpperCase(), url: originalRequest.url })
        return silentRefresh('hr')
          .then(() => {
            const refreshMs = Date.now() - refreshStartTime
            const retryCount = failedRequests.length
            // Retry all queued requests.
            failedRequests.forEach((p) => p.resolve(undefined))
            failedRequests.length = 0
            isRefreshing = false
            originalRequest._retry = true
            debugLog.http.info('refresh_succeeded', { retry_count: retryCount, refresh_duration_ms: refreshMs })
            return http(originalRequest)
          })
          .catch(() => {
            const refreshMs = Date.now() - refreshStartTime
            failedRequests.forEach((p) => p.reject(new Error('refresh failed')))
            failedRequests.length = 0
            isRefreshing = false
            clearLocalAuthCache()
            useAuthStore().$reset()
            router.push('/login')
            debugLog.http.warn('refresh_failed', { refresh_duration_ms: refreshMs })
            return Promise.reject(new Error('refresh failed'))
          })
      }
      // Queue this request while refresh is in progress.
      debugLog.http.info('refresh_queued', { method: originalRequest.method?.toUpperCase(), url: originalRequest.url })
      return new Promise((resolve, reject) => {
        failedRequests.push({ resolve: resolve as (value: unknown) => void, reject, config: originalRequest })
      }).then(() => {
        originalRequest._retry = true
        return http(originalRequest)
      })
    }

    const friendlyMessage = friendlyNetworkMessage(error)
    const data = error.response?.data as Record<string, unknown> | undefined
    const requestId = (data?.request_id as string) || ''
    const resetAt = (data?.data as Record<string, unknown>)?.reset_at as string || ''
    const displayMsg = resetAt ? `${friendlyMessage}（${formatResetTime(resetAt)}恢复）` : friendlyMessage
    if (!originalRequest?.silentError) {
      ElMessage.error(requestId ? `${displayMsg} [${requestId.slice(0, 8)}]` : displayMsg)
    }
    debugLog.http.warn('response_failed', { ...logData, request_id: requestId?.slice(0, 8) || '', business_msg: friendlyMessage })
    if (data?.code && typeof data.code === 'number') {
      const businessError = new BusinessError(data.code as number, friendlyMessage, requestId)
      const contextGuardCode = contextGuardCodeFrom(data.msg)
      if (contextGuardCode) {
        Object.assign(businessError, { contextGuardCode })
      }
      return Promise.reject(businessError)
    }
    return Promise.reject(new Error(friendlyMessage))
  },
)

export const friendlyBusinessMessage = (code: number, msg: string): string => {
  const guardMessage = contextGuardMessage(contextGuardCodeFrom(msg))
  if (guardMessage) return guardMessage
  if (msg === 'extract resume profile: resume profile extractor is not configured') {
    return '简历画像解析器未配置，请先完成后端解析器配置后再发起解析'
  }
  if (code === 401) return msg || '登录状态已失效，请重新登录'
  if (code === 403 || code === 4030) return msg || '当前账号没有权限执行这个操作'
  if (code === 404) return msg || '请求的资源不存在或已失效'
  if (code === 429) return msg || '请求过于频繁，请稍后再试'
  if (code === 42901) return msg || '今日 AI 使用次数已达上限，请明天再试'
  if (code === 42902) return msg || 'AI 请求太频繁，请稍后再试'
  if (code === 42921) return msg || '当前操作过于频繁，请稍后再试'
  if (code === 499) return '请求已取消，请重新操作'
  if (code === 502) return msg || 'AI 或第三方服务暂时不可用，请稍后重试'
  if (code === 503) return msg || '后端服务暂不可用，请稍后重试'
  if (code === 504) return msg || '请求处理超时，请稍后重试'
  if (code === 500) return msg || '服务暂时开小差了，请稍后再试'
  return msg || '操作没有成功，请稍后再试'
}

const friendlyNetworkMessage = (error: AxiosError): string => {
  if (error.code === 'ECONNABORTED') return '请求处理超时，请稍后重试'
  if (error.code === 'ERR_NETWORK') return '网络连接失败，请确认后端服务已启动'
  if (error.response?.data) {
    const data = error.response.data as Record<string, unknown>
    if (data.code && data.msg) {
      return friendlyBusinessMessage(data.code as number, data.msg as string)
    }
  }
  if (error.response && error.response.status === 429) return '请求过于频繁，请稍后再试'
  if (error.response && error.response.status >= 500) return '服务器暂时不可用，请稍后再试'
  return '请求失败，请稍后再试'
}

interface RequestInstance {
  get<T = unknown>(url: string, config?: RequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: RequestConfig): Promise<T>
}

const formatResetTime = (resetAt: string): string => {
  try {
    const d = new Date(resetAt)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch { return '' }
}

export default http as unknown as RequestInstance
