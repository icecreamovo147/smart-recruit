import axios from 'axios'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import { clearLocalAuthCache } from '@/utils/token'
import { BusinessError } from '@/types/api'
import { silentRefresh } from './authRefresh'
import { formatShanghaiTime } from '@shared/utils/format'
import { t } from '@shared/i18n'

interface RequestConfig extends AxiosRequestConfig {
  silentError?: boolean
}

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 60000,
  withCredentials: true, // Send httpOnly auth cookies for CORS requests
})

// Auth handled via httpOnly Cookie — no Authorization header needed.
http.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'candidate')
  return config
})

// Silent refresh state — shared across all interceptors.
let isRefreshing = false
const failedRequests: Array<{
  resolve: (value: unknown) => void
  reject: (reason: unknown) => void
}> = []

http.interceptors.response.use(
  (response) => {
    const { code, msg, data, message_key: messageKey } = response.data
    if (code !== 0) {
      const requestId = response.data.request_id as string || ''
      const error = new BusinessError(code, friendlyBusinessMessage(code, messageKey ? msg : ''), requestId)
      const resetAt = response.data?.data?.reset_at as string || ''
      const displayMsg = resetAt ? `${error.message}（${formatResetTime(resetAt)}恢复）` : error.message
      if (!(response.config as RequestConfig).silentError) {
        ElMessage.error(requestId ? `${displayMsg} [${requestId.slice(0, 8)}]` : displayMsg)
      }
      return Promise.reject(error)
    }
    return data
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as RequestConfig & { _retry?: boolean }

    // Attempt silent refresh on 401 for non-refresh requests.
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      if (!isRefreshing) {
        isRefreshing = true
        return silentRefresh('candidate')
          .then(() => {
            failedRequests.forEach((p) => p.resolve(undefined))
            failedRequests.length = 0
            isRefreshing = false
            originalRequest._retry = true
            return http(originalRequest)
          })
          .catch(() => {
            failedRequests.forEach((p) => p.reject(new Error(t('common.refresh_failed'))))
            failedRequests.length = 0
            isRefreshing = false
            clearLocalAuthCache()
            useAuthStore().$reset()
            router.push('/login')
            return Promise.reject(new Error(t('common.refresh_failed')))
          })
      }
      return new Promise((resolve, reject) => {
        failedRequests.push({ resolve: resolve as (value: unknown) => void, reject })
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
    if (data?.code && typeof data.code === 'number') {
      return Promise.reject(new BusinessError(data.code as number, friendlyMessage, requestId))
    }
    return Promise.reject(new Error(friendlyMessage))
  },
)

const friendlyBusinessMessage = (code: number, msg: string): string => {
  if (code === 401) return msg || t('common.unauthenticated')
  if (code === 403 || code === 4030) return msg || t('common.forbidden')
  if (code === 4001) return msg || '请先完善个人资料后再投递'
  if (code === 4002) return msg || '请先上传简历后再投递'
  if (code === 4003) return msg || '你已经投递过这个岗位'
  if (code === 4004) return msg || '该岗位已下架，无法投递'
  if (code === 404) return msg || t('common.not_found')
  if (code === 40201) return msg || t('ai.insufficient_credits')
  if (code === 429) return msg || t('common.too_many_requests')
  if (code === 42901) return msg || t('ai.daily_quota_exceeded')
  if (code === 42902) return msg || t('ai.too_many_requests')
  if (code === 42911) return msg || '简历上传过于频繁，请稍后再试'
  if (code === 42912) return msg || '简历上传确认过于频繁，请稍后再试'
  if (code === 42921) return msg || t('common.too_many_requests')
  if (code === 499) return t('common.canceled')
  if (code === 502) return msg || t('ai.unavailable')
  if (code === 503) return msg || t('common.backend_unavailable')
  if (code === 504) return msg || t('common.timeout')
  if (code === 500) return msg || t('common.unknown_error')
  return msg || t('common.operation_failed')
}

const friendlyNetworkMessage = (error: AxiosError): string => {
  if (error.code === 'ECONNABORTED') return t('common.timeout')
  if (error.code === 'ERR_NETWORK') return t('common.network_error')
  if (error.response?.data) {
    const data = error.response.data as Record<string, unknown>
    if (data.code && data.msg) {
      return friendlyBusinessMessage(data.code as number, data.msg as string)
    }
  }
  if (error.response && error.response.status === 429) return t('common.too_many_requests')
  if (error.response && error.response.status >= 500) return t('common.unknown_error')
  return t('common.operation_failed')
}

// Typed request interface — the interceptor unwraps response.data,
// so callers get T directly instead of AxiosResponse<T>.
interface RequestInstance {
  get<T = unknown>(url: string, config?: RequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: RequestConfig): Promise<T>
}

const formatResetTime = (resetAt: string): string => {
  try {
    return formatShanghaiTime(resetAt)
  } catch { return '' }
}

export default http as unknown as RequestInstance
