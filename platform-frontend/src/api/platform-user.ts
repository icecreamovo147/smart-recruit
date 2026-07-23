import http from './http'
import type { PageResult, PlatformAccount } from '@/types'

export const listPlatformUsers = (params: Record<string, string | number | undefined>) =>
  http.get<never, PageResult<PlatformAccount>>('/api/v1/platform/users', { params })

export const createPlatformUser = (payload: { username: string; email?: string; password: string; role_key: string }) =>
  http.post<never, { user_id: number }>('/api/v1/platform/users', payload)

export const updatePlatformUser = (userId: number, payload: { role_key: string; status: string; reason: string }) =>
  http.patch<never, { user: PlatformAccount }>(`/api/v1/platform/users/${userId}`, payload)
