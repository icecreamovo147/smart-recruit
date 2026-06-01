import axios from 'axios'
import { clearLocalAuthCache } from '@/utils/token'
import { useAuthStore } from '@/stores/auth'
import type { NotificationListResponse, NotificationSummaryResponse, UnreadCountResponse } from '@/types/notification'
import { silentRefresh } from './authRefresh'

const silent = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 10000,
  withCredentials: true,
})

// Auth handled via httpOnly Cookie.
silent.interceptors.request.use((config) => {
  config.headers.set('X-Client-App', 'candidate')
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
        await silentRefresh('candidate')
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const get = <T>(url: string, params?: Record<string, unknown>): Promise<T> =>
  silent.get(url, { params }) as Promise<T>

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const patch = <T>(url: string): Promise<T> =>
  silent.patch(url) as Promise<T>

export const listNotifications = (params: { page: number; page_size: number }) =>
  get<NotificationListResponse>('/api/v1/candidate/notifications', params)

export const getUnreadNotificationCount = () =>
  get<UnreadCountResponse>('/api/v1/candidate/notifications/unread-count')

export const getNotificationSummary = () =>
  get<NotificationSummaryResponse>('/api/v1/candidate/notifications/summary')

export const openNotificationStream = async (signal: AbortSignal) => {
  const fetchStream = () => fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/candidate/notifications/stream`, {
    signal,
    credentials: 'include',
    headers: { 'X-Client-App': 'candidate' },
  })
  let response = await fetchStream()
  if (response.status === 401) {
    await silentRefresh('candidate')
    response = await fetchStream()
  }
  return response
}

export const markNotificationRead = (notificationId: number) =>
  patch(`/api/v1/candidate/notifications/${notificationId}/read`)

export const markAllNotificationsRead = () =>
  patch('/api/v1/candidate/notifications/read-all')
