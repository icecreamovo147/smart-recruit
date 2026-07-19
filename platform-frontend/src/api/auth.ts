import http from './http'
import type { PlatformUser } from '@/types'

export interface LoginResponse extends PlatformUser { role: number }

export const login = (username: string, password: string): Promise<LoginResponse> =>
  http.post('/api/v1/auth/login', { username, password })

export const me = (): Promise<LoginResponse> => http.get('/api/v1/auth/me')
export const logout = (): Promise<void> => http.post('/api/v1/auth/logout')
