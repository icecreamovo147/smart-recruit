// Types for the workbench dashboard API response.

export interface DashboardKPI {
  online_jobs: number
  offline_jobs: number
  total_applications: number
  today_applications: number
  unread_notifications: number
  pending_actions: number
}

export interface TrendData {
  dates: string[]
  applications: number[]
}

export interface DistributionData {
  labels: string[]
  values: number[]
}

export interface DashboardSummary {
  kpi: DashboardKPI
  trend: TrendData
  stage_distribution: DistributionData
  job_distribution: DistributionData
}
