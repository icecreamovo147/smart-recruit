import axios from 'axios'
import { getToken } from '@/utils/token'
import type { NotificationListResponse, NotificationSummaryResponse, UnreadCountResponse } from '@/types/notification'

// Raw axios instance for notification polling — no ElMessage.error on failure,
// because polling failures must be silent and not disrupt the user.
const silent = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 10000,
})

silent.interceptors.request.use((config) => {
  const token = getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Unwrap the Gin envelope {code, msg, data} → return inner data, or reject silently on non-zero.
silent.interceptors.response.use(
  (response) => {
    const { code, data } = response.data || {}
    if (code !== undefined && code !== 0) {
      return Promise.reject(new Error(`notification api: code=${code}`))
    }
    return data
  },
  (error) => Promise.reject(error),
)

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const get = <T>(url: string, params?: Record<string, unknown>): Promise<T> =>
  silent.get(url, { params }) as Promise<T>

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const patch = <T>(url: string): Promise<T> =>
  silent.patch(url) as Promise<T>

export const listNotifications = (params: { page: number; page_size: number }) =>
  get<NotificationListResponse>('/api/v1/hr/notifications', params)

export const getUnreadNotificationCount = () =>
  get<UnreadCountResponse>('/api/v1/hr/notifications/unread-count')

export const getNotificationSummary = () =>
  get<NotificationSummaryResponse>('/api/v1/hr/notifications/summary')

export const openNotificationStream = (signal: AbortSignal) => {
  const token = getToken()
  return fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/hr/notifications/stream`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    signal,
  })
}

export const markNotificationRead = (notificationId: number) =>
  patch(`/api/v1/hr/notifications/${notificationId}/read`)

export const markAllNotificationsRead = () =>
  patch('/api/v1/hr/notifications/read-all')
