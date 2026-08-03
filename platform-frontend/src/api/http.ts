import axios, { type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { clearPlatformUser } from '@/stores/auth'
import { localizedBackendMessage } from '@shared/i18n'

const http = axios.create({ baseURL: import.meta.env.VITE_API_BASE_URL || '', timeout: 30000, withCredentials: true })

export interface PlatformRequestConfig extends AxiosRequestConfig {
  silentError?: boolean
}

interface PlatformRequestInstance {
  get<T = unknown, R = T>(url: string, config?: PlatformRequestConfig): Promise<R>
  post<T = unknown, R = T>(url: string, data?: unknown, config?: PlatformRequestConfig): Promise<R>
  put<T = unknown, R = T>(url: string, data?: unknown, config?: PlatformRequestConfig): Promise<R>
  patch<T = unknown, R = T>(url: string, data?: unknown, config?: PlatformRequestConfig): Promise<R>
  delete<T = unknown, R = T>(url: string, config?: PlatformRequestConfig): Promise<R>
}

http.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'platform')
  return config
})

http.interceptors.response.use(
  (response) => {
    const { code, msg, data, message_key: messageKey } = response.data || {}
    if (code !== 0) {
      const requestConfig = response.config as typeof response.config & PlatformRequestConfig
      const displayMessage = localizedBackendMessage({ message_key: messageKey, msg })
      if (!requestConfig.silentError) ElMessage.error(displayMessage)
      return Promise.reject(new Error(displayMessage))
    }
    return data
  },
  async (error) => {
    const request = error.config as typeof error.config & PlatformRequestConfig & { _retry?: boolean }
    if (error.response?.status === 401 && request && !request._retry) {
      request._retry = true
      const refresh = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/auth/refresh`, {
        method: 'POST', credentials: 'include', headers: { 'X-Client-App': 'platform' },
      })
      const body = await refresh.json().catch(() => ({}))
      if (body.code === 0) return http(request)
      clearPlatformUser()
      await router.push('/login')
    }
    const msg = localizedBackendMessage(error.response?.data, 'common.operation_failed')
    if (!request?.silentError) ElMessage.error(msg)
    return Promise.reject(error)
  },
)

export default http as unknown as PlatformRequestInstance
