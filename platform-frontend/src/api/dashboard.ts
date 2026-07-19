import http from './http'
import type { PlatformDashboard } from '@/types'

export const getPlatformDashboard = (): Promise<PlatformDashboard> =>
  http.get('/api/v1/platform/dashboard')
