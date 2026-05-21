import request from './request'
import type { LoginPayload, LoginResponse, RegisterPayload } from '@/types/domain'

export const login = (data: LoginPayload): Promise<LoginResponse> =>
  request.post('/api/v1/auth/login', data)

export const register = (data: RegisterPayload): Promise<{ user_id: number; username: string; role: number }> =>
  request.post('/api/v1/auth/register', data)
