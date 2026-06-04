// ── Application Status Keys (Phase 1 string-based state machine) ───────

export const APP_STATUS_KEY = {
  APPLIED: 'applied',
  VIEWED: 'viewed',
  SCREENING: 'screening',
  SCREEN_PASSED: 'screen_passed',
  INTERVIEW_PENDING: 'interview_pending',
  INTERVIEWING: 'interviewing',
  INTERVIEW_PASSED: 'interview_passed',
  OFFER_PENDING: 'offer_pending',
  OFFER_SENT: 'offer_sent',
  OFFER_ACCEPTED: 'offer_accepted',
  OFFER_REJECTED: 'offer_rejected',
  HIRED: 'hired',
  REJECTED: 'rejected',
  WITHDRAWN: 'withdrawn',
} as const

export type AppStatusKey = (typeof APP_STATUS_KEY)[keyof typeof APP_STATUS_KEY]

// Candidate-facing status labels (safe, no internal reasons).
export const CANDIDATE_STATUS_LABELS: Record<string, string> = {
  [APP_STATUS_KEY.APPLIED]: '已投递',
  [APP_STATUS_KEY.VIEWED]: '简历被查看',
  [APP_STATUS_KEY.SCREENING]: '筛选中',
  [APP_STATUS_KEY.SCREEN_PASSED]: '筛选通过',
  [APP_STATUS_KEY.INTERVIEW_PENDING]: '待面试',
  [APP_STATUS_KEY.INTERVIEWING]: '面试中',
  [APP_STATUS_KEY.INTERVIEW_PASSED]: '面试通过',
  [APP_STATUS_KEY.OFFER_PENDING]: '待发offer',
  [APP_STATUS_KEY.OFFER_SENT]: 'Offer已发',
  [APP_STATUS_KEY.OFFER_ACCEPTED]: 'Offer已接受',
  [APP_STATUS_KEY.OFFER_REJECTED]: 'Offer已拒绝',
  [APP_STATUS_KEY.HIRED]: '已入职',
  [APP_STATUS_KEY.REJECTED]: '未通过',
  [APP_STATUS_KEY.WITHDRAWN]: '已撤回',
}

export const STATUS_TYPE_MAP: Record<string, string> = {
  [APP_STATUS_KEY.APPLIED]: 'info',
  [APP_STATUS_KEY.VIEWED]: 'primary',
  [APP_STATUS_KEY.SCREENING]: 'warning',
  [APP_STATUS_KEY.SCREEN_PASSED]: 'success',
  [APP_STATUS_KEY.INTERVIEW_PENDING]: 'warning',
  [APP_STATUS_KEY.INTERVIEWING]: 'warning',
  [APP_STATUS_KEY.INTERVIEW_PASSED]: 'success',
  [APP_STATUS_KEY.OFFER_PENDING]: 'warning',
  [APP_STATUS_KEY.OFFER_SENT]: 'primary',
  [APP_STATUS_KEY.OFFER_ACCEPTED]: 'success',
  [APP_STATUS_KEY.OFFER_REJECTED]: 'danger',
  [APP_STATUS_KEY.HIRED]: 'success',
  [APP_STATUS_KEY.REJECTED]: 'danger',
  [APP_STATUS_KEY.WITHDRAWN]: 'info',
}

// Legacy numeric status mapping for backward compatibility.
export enum ApplicationStatus {
  Pending = 0,
  Viewed = 1,
  Passed = 2,
  Rejected = 3,
}

export const APPLICATION_STATUS_TEXT: Record<ApplicationStatus, string> = {
  [ApplicationStatus.Pending]: '待查看',
  [ApplicationStatus.Viewed]: '已查看',
  [ApplicationStatus.Passed]: '通过',
  [ApplicationStatus.Rejected]: '淘汰',
}

export const APPLICATION_STATUS_TYPE: Record<ApplicationStatus, string> = {
  [ApplicationStatus.Pending]: 'info',
  [ApplicationStatus.Viewed]: 'primary',
  [ApplicationStatus.Passed]: 'success',
  [ApplicationStatus.Rejected]: 'danger',
}

export enum JobStatus {
  Offline = 0,
  Recruiting = 1,
}

// ---- Helpers ----

export function getCandidateStatusLabel(statusKey: string, fallback?: string): string {
  return CANDIDATE_STATUS_LABELS[statusKey] || fallback || '未知'
}

export function getStatusType(statusKey: string, fallback?: string): string {
  return STATUS_TYPE_MAP[statusKey] || fallback || 'info'
}

// ---- User ----

export interface User {
  user_id: number
  role: number              // Deprecated: kept for migration compatibility
  username: string
  account_type?: string     // 'candidate' | 'staff' | 'service'
  roles?: string[]          // RBAC role keys
  permissions?: string[]    // RBAC permission keys
}

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResponse {
  user_id: number
  role: number              // Deprecated: kept for migration compatibility
  username: string
  account_type?: string
  roles?: string[]
  permissions?: string[]
}

export interface RegisterPayload {
  username: string
  password: string
  email?: string
  role: number
  invite_code?: string  // Required for staff (HR) registration
}

// ---- Job ----

export interface Job {
  job_id: number
  title: string
  department: string
  location: string
  salary_range: string
  description: string
  requirements: string
  status?: number
  application_count?: number
  created_at?: string
  // camelCase fallbacks for API inconsistency
  jobId?: number
  salaryRange?: string
  applicationCount?: number
  createdAt?: string
}

export interface JobQuery {
  page: number
  page_size: number
  keyword?: string
}

export interface JobCreatePayload {
  title: string
  department?: string
  location?: string
  salary_range?: string
  description?: string
  requirements?: string
}

export type JobUpdatePayload = JobCreatePayload

export interface PaginatedList<T> {
  list: T[]
  total: number
}

// ---- Application ----

export interface Application {
  application_id: number
  job_id: number
  job_title: string
  status: number
  status_key?: string       // Phase 1: string status key
  round_no: number
  is_current: number
  applied_at: string
  applied_time_display?: string
  // Fields from candidate profile
  user_id?: number
  real_name?: string
  phone?: string
  education?: string
  school?: string
  skills?: string[]
  resume_url?: string
}

// ---- Profile ----

export interface Profile {
  real_name: string
  phone: string
  education: string
  school: string
  work_experience: string
  skills: string[] | string
  is_complete?: boolean
  // camelCase fallback
  realName?: string
  workExperience?: string
  isComplete?: boolean
}

// ---- Resume ----

export interface ResumeInfo {
  resume_id?: number
  file_name: string
  file_type: string
  file_size: number
  uploaded_at: string
  resume_url?: string
  // camelCase fallback
  resumeId?: number
  fileName?: string
  fileType?: string
  fileSize?: number
  uploadedAt?: string
  resumeUrl?: string
}

export interface PresignPayload {
  file_name: string
  file_type: string
}

export interface PresignResponse {
  upload_url: string
  oss_key: string
  expire_at?: string
  upload_id?: string
}

export interface ConfirmPayload {
  oss_key: string
  file_name: string
  file_type: string
  file_size: number
  upload_id?: string
}

// ---- Interview ----

export interface InterviewSchedule {
  interview_id: number
  application_id: number
  round_no: number
  title: string
  mode: string          // video / phone / onsite
  meeting_url: string
  location: string
  duration_minutes: number
  candidate_note: string
  scheduled_at: string
  status: string        // pending / scheduled / completed / cancelled
  created_at: string
  updated_at: string
  interviewer_name: string
  job_title: string
  candidate_name: string
}
