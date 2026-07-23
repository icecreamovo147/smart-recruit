import http, { type PlatformRequestConfig } from './http'
import type { PlatformUser } from '@/types'

export interface LoginResponse extends PlatformUser { role: number }

export const login = (username: string, password: string): Promise<LoginResponse> =>
  http.post('/api/v1/auth/login', { username, password })

const silentRestoreConfig: PlatformRequestConfig = { silentError: true }

export const me = (): Promise<LoginResponse> => http.get('/api/v1/auth/me', silentRestoreConfig)
export const logout = (): Promise<void> => http.post('/api/v1/auth/logout')
