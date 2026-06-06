import axios from 'axios'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { useAuthStore } from '@/stores/auth'
import { clearLocalAuthCache } from '@/utils/token'
import { BusinessError } from '@/types/api'
import { silentRefresh } from './authRefresh'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 60000,
  withCredentials: true, // Send httpOnly auth cookies for CORS requests
})

// Auth handled via httpOnly Cookie — no Authorization header needed.
http.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'hr')
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
    const { code, msg, data } = response.data
    if (code !== 0) {
      const requestId = response.data.request_id as string || ''
      const error = new BusinessError(code, friendlyBusinessMessage(code, msg), requestId)
      const resetAt = response.data?.data?.reset_at as string || ''
      const displayMsg = resetAt ? `${error.message}（${formatResetTime(resetAt)}恢复）` : error.message
      ElMessage.error(requestId ? `${displayMsg} [${requestId.slice(0, 8)}]` : displayMsg)
      return Promise.reject(error)
    }
    return data
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }

    // Attempt silent refresh on 401 for non-refresh requests.
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      if (!isRefreshing) {
        isRefreshing = true
        return silentRefresh('hr')
          .then(() => {
            // Retry all queued requests.
            failedRequests.forEach((p) => p.resolve(undefined))
            failedRequests.length = 0
            isRefreshing = false
            originalRequest._retry = true
            return http(originalRequest)
          })
          .catch(() => {
            failedRequests.forEach((p) => p.reject(new Error('refresh failed')))
            failedRequests.length = 0
            isRefreshing = false
            clearLocalAuthCache()
            useAuthStore().$reset()
            router.push('/login')
            return Promise.reject(new Error('refresh failed'))
          })
      }
      // Queue this request while refresh is in progress.
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
    ElMessage.error(requestId ? `${displayMsg} [${requestId.slice(0, 8)}]` : displayMsg)
    if (data?.code && typeof data.code === 'number') {
      return Promise.reject(new BusinessError(data.code as number, friendlyMessage, requestId))
    }
    return Promise.reject(new Error(friendlyMessage))
  },
)

const friendlyBusinessMessage = (code: number, msg: string): string => {
  if (code === 401) return '登录状态已失效，请重新登录'
  if (code === 403 || code === 4030) return msg || '当前账号没有权限执行这个操作'
  if (code === 404) return msg || '请求的资源不存在或已失效'
  if (code === 429) return msg || '请求过于频繁，请稍后再试'
  if (code === 42901) return msg || '今日 AI 使用次数已达上限，请明天再试'
  if (code === 42902) return msg || 'AI 请求太频繁，请稍后再试'
  if (code === 42921) return msg || '当前操作过于频繁，请稍后再试'
  if (code === 499) return '请求已取消，请重新操作'
  if (code === 502) return 'AI 或第三方服务暂时不可用，请稍后重试'
  if (code === 503) return '后端服务暂不可用，请稍后重试'
  if (code === 504) return '请求处理超时，请稍后重试'
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
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
}

const formatResetTime = (resetAt: string): string => {
  try {
    const d = new Date(resetAt)
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch { return '' }
}

export default http as unknown as RequestInstance
