import request from './request'
import type { DashboardSummary } from '@/types/dashboard'

export const getDashboardSummary = (): Promise<DashboardSummary> =>
  request.get('/api/v1/hr/dashboard/summary')
