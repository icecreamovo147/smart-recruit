// Re-export shared types
export type { User, LoginPayload, LoginResponse } from '@shared/types/domain'

// ── RBAC Constants ──────────────────────────────────────────────────

export const ROLE_KEY_INTERVIEWER = 'interviewer'

export const PERM = {
  AUTH_SESSION_READ: 'auth.session.read',
  INTERVIEW_READ: 'interview.read',
  INTERVIEW_FEEDBACK_SUBMIT: 'interview.feedback.submit',
  NOTIFICATION_READ: 'notification.read',
} as const

// ── Interview ───────────────────────────────────────────────────────

export interface InterviewSchedule {
  interview_id: number
  application_id: number
  interviewer_id: number
  round_no: number
  title: string
  mode: string
  meeting_url: string
  location: string
  duration_minutes: number
  candidate_note: string
  internal_note: string
  cancel_reason: string
  scheduled_at: string
  status: string
  created_by: number
  created_at: string
  updated_at: string
  interviewer_name: string
  application_status_key: string
  job_title: string
  candidate_name: string
  candidate_phone: string
  resume_url: string
  has_feedback: boolean
}

export interface InterviewFeedback {
  feedback_id: number
  interview_id: number
  application_id: number
  interviewer_id: number
  recommendation: string
  score: number
  dimension_scores_json: string
  comments: string
  submitted_at: string
  updated_at: string
  interviewer_name: string
}

export interface FeedbackPayload {
  application_id: number
  recommendation: string
  score: number
  dimension_scores_json?: string
  comments?: string
}

// ── Interview status display ────────────────────────────────────────

export const INTERVIEW_STATUS_LABEL: Record<string, string> = {
  pending: '待安排',
  scheduled: '待面试',
  completed: '已完成',
  cancelled: '已取消',
}

export const INTERVIEW_STATUS_TYPE: Record<string, string> = {
  pending: 'info',
  scheduled: 'primary',
  completed: 'success',
  cancelled: 'danger',
}

export const INTERVIEW_MODE_LABEL: Record<string, string> = {
  video: '视频面试',
  phone: '电话面试',
  onsite: '现场面试',
}

export const RECOMMENDATION_LABEL: Record<string, string> = {
  strong_recommend: '强烈推荐',
  recommend: '推荐通过',
  neutral: '待定',
  not_recommend: '不推荐',
  strong_not_recommend: '强烈不推荐',
}

export const RECOMMENDATION_OPTIONS = [
  { value: 'strong_recommend', label: '强烈推荐' },
  { value: 'recommend', label: '推荐通过' },
  { value: 'neutral', label: '待定' },
  { value: 'not_recommend', label: '不推荐' },
  { value: 'strong_not_recommend', label: '强烈不推荐' },
]

export const DIMENSION_LABELS: Record<string, string> = {
  professional: '专业能力',
  communication: '沟通表达',
  problem_solving: '问题分析',
  job_fit: '岗位匹配',
  potential: '发展潜力',
}

// ── Notification ────────────────────────────────────────────────────

export interface NotificationItem {
  notification_id: number
  title: string
  body: string
  link: string
  is_read: number
  created_at: string
}

export interface NotificationListResponse {
  list: NotificationItem[]
  total: number
}
