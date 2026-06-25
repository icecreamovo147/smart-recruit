import request from './request'
import type { LoginPayload, LoginResponse } from '@shared/types/domain'

export const login = (data: LoginPayload): Promise<LoginResponse> =>
  request.post('/api/v1/auth/login', data)

export const updateEmail = (email: string): Promise<{ code: number; msg: string }> =>
  request.put('/api/v1/auth/email', { email })
