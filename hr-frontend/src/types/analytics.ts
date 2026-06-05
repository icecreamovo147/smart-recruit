// Types for Phase 6 analytics and reporting APIs.

export interface DashboardTrendPoint {
  date: string
  applications: number
}

export interface DashboardStageItem {
  stage_key: string
  stage_label: string
  count: number
}

// Backend returns flat fields (not nested under "kpi") — matches proto GetDashboardReportResponse.
export interface DashboardReport {
  online_jobs: number
  offline_jobs: number
  total_applications: number
  today_applications: number
  unread_notifications: number
  pending_actions: number
  trend: DashboardTrendPoint[]
  stage_distribution: DashboardStageItem[]
}

export interface FunnelStage {
  stage_key: string
  stage_label: string
  count: number
  conversion_rate: number
}

export interface FunnelReport {
  stages: FunnelStage[]
}

export interface StageDuration {
  stage_key: string
  stage_label: string
  avg_hours: number
  transition_count: number
}

export interface TimeInStageReport {
  durations: StageDuration[]
}

export interface InterviewOfferMetrics {
  total_interviews: number
  completed_interviews: number
  positive_feedbacks: number
  pass_rate: number
  total_offers: number
  accepted_offers: number
  rejected_offers: number
  acceptance_rate: number
}

export interface AuthAuditLogItem {
  id: number
  actor_user_id: number
  actor_roles: string
  permission_key: string
  resource_type: string
  resource_id: number
  decision: string
  reason: string
  request_id: string
  client_ip: string
  created_at: string
}

export interface AuthAuditApiResponse {
  code?: number
  msg?: string
  total: number
  list: AuthAuditLogItem[]
}

export interface AnalyticsQuery {
  job_id?: number
  start_date?: string
  end_date?: string
}
