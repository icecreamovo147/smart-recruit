import request from './request'
import type {
  DashboardReport,
  FunnelReport,
  TimeInStageReport,
  InterviewOfferMetrics,
  AuthAuditApiResponse,
  AnalyticsQuery,
} from '@/types/analytics'

export const getDashboardReport = (): Promise<DashboardReport> =>
  request.get('/api/v1/hr/analytics/dashboard')

export const getFunnelReport = (query?: AnalyticsQuery): Promise<FunnelReport> =>
  request.get('/api/v1/hr/analytics/funnel', { params: query })

export const getTimeInStageReport = (query?: AnalyticsQuery): Promise<TimeInStageReport> =>
  request.get('/api/v1/hr/analytics/time-in-stage', { params: query })

export const getInterviewOfferMetrics = (query?: AnalyticsQuery): Promise<InterviewOfferMetrics> =>
  request.get('/api/v1/hr/analytics/metrics', { params: query })

export const getAuthAuditLogs = (params: {
  page?: number
  page_size?: number
  actor_user_id?: number
  permission_key?: string
  decision?: string
}): Promise<AuthAuditApiResponse> =>
  request.get('/api/v1/hr/admin/auth-audit-logs', { params })
