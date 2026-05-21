import request from './request'
import type { Profile } from '@/types/domain'

export const getProfile = (): Promise<Profile> =>
  request.get('/api/v1/candidate/profile')

export const updateProfile = (data: Profile): Promise<Profile> =>
  request.put('/api/v1/candidate/profile', data)
