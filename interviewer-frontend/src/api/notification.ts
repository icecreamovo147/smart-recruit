import axios from 'axios'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import type { NotificationListResponse } from '@/types/domain'
import { silentRefresh } from './authRefresh'

// Silent axios instance — no ElMessage.error on failure for polling
const silent = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 10000,
  withCredentials: true,
})

silent.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'interviewer')
  return config
})

silent.interceptors.response.use(
  (response) => {
    const { code, data } = response.data || {}
    if (code === 401) {
      clearLocalAuthCache()
      useAuthStore().$reset()
      return Promise.reject(new Error('notification api: unauthorized'))
    }
    if (code !== undefined && code !== 0) {
      return Promise.reject(new Error(`notification api: code=${code}`))
    }
    return data
  },
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      try {
        await silentRefresh()
        originalRequest._retry = true
        return silent(originalRequest)
      } catch {
        clearLocalAuthCache()
        useAuthStore().$reset()
      }
    }
    return Promise.reject(error)
  },
)

const get = <T>(url: string, params?: Record<string, unknown>): Promise<T> =>
  silent.get(url, { params }) as Promise<T>

const patch = <T>(url: string): Promise<T> =>
  silent.patch(url) as Promise<T>

export const listNotifications = (params: { page: number; page_size: number }) =>
  get<NotificationListResponse>('/api/v1/hr/notifications', params)

export const getUnreadNotificationCount = () =>
  get<{ count: number }>('/api/v1/hr/notifications/unread-count')

export const markNotificationRead = (notificationId: number) =>
  patch(`/api/v1/hr/notifications/${notificationId}/read`)

export const markAllNotificationsRead = () =>
  patch('/api/v1/hr/notifications/read-all')
