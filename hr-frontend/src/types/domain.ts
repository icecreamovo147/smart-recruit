// ── Role Constants ─────────────────────────────────────────────────────
// Deprecated numeric roles — kept for backward compatibility during migration.
export const ROLE_CANDIDATE = 1
export const ROLE_HR = 2
export const ROLE_HR_ADMIN = 3

// RBAC role keys (string-based — use these for all new code).
export const ROLE_KEY_CANDIDATE = 'candidate'
export const ROLE_KEY_RECRUITER = 'recruiter'
export const ROLE_KEY_RECRUITING_ADMIN = 'recruiting_admin'
export const ROLE_KEY_SYSTEM_ADMIN = 'system_admin'
export const ROLE_KEY_INTERVIEWER = 'interviewer'

// RBAC permission keys.
export const PERM = {
  AUTH_SESSION_READ: 'auth.session.read',
  CANDIDATE_PROFILE_MANAGE: 'candidate.profile.manage',
  CANDIDATE_RESUME_MANAGE: 'candidate.resume.manage',
  CANDIDATE_APPLICATION_MANAGE: 'candidate.application.manage',
  JOB_READ: 'job.read',
  JOB_CREATE: 'job.create',
  JOB_UPDATE: 'job.update',
  JOB_PUBLISH: 'job.publish',
  APPLICATION_READ: 'application.read',
  APPLICATION_STATUS_UPDATE: 'application.status.update',
  INTERVIEW_READ: 'interview.read',
  INTERVIEW_SCHEDULE: 'interview.schedule',
  INTERVIEW_FEEDBACK_SUBMIT: 'interview.feedback.submit',
  NOTIFICATION_READ: 'notification.read',
  AI_HR_USE: 'ai.hr.use',
  AI_CANDIDATE_USE: 'ai.candidate.use',
  ADMIN_INVITE_MANAGE: 'admin.invite.manage',
  ADMIN_DEPARTMENT_MANAGE: 'admin.department.manage',
  ADMIN_LOCATION_MANAGE: 'admin.location.manage',
  ADMIN_USER_MANAGE: 'admin.user.manage',
  ADMIN_ROLE_MANAGE: 'admin.role.manage',
  AUDIT_USAGE_READ: 'audit.usage.read',
  AUDIT_SECURITY_READ: 'audit.security.read',
  SYSTEM_CONFIG_MANAGE: 'system.config.manage',

  // Offer management
  OFFER_READ: 'offer.read',
  OFFER_MANAGE: 'offer.manage',
  OFFER_SEND: 'offer.send',
  OFFER_DECISION_MANAGE: 'offer.decision.manage',

  // Collaboration
  COLLABORATION_NOTE_READ: 'collaboration.note.read',
  COLLABORATION_NOTE_CREATE: 'collaboration.note.create',
  COLLABORATION_TAG_MANAGE: 'collaboration.tag.manage',
  COLLABORATION_TASK_MANAGE: 'collaboration.task.manage',
} as const

export type PermissionKey = (typeof PERM)[keyof typeof PERM]

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

// Legacy numeric status mapping for backward compatibility.
export enum ApplicationStatus {
  Pending = 0,
  Viewed = 1,
  Passed = 2,
  Rejected = 3,
}

// Legacy status to status key mapping.
export const LEGACY_STATUS_TO_KEY: Record<number, AppStatusKey> = {
  [ApplicationStatus.Pending]: APP_STATUS_KEY.APPLIED,
  [ApplicationStatus.Viewed]: APP_STATUS_KEY.VIEWED,
  [ApplicationStatus.Passed]: APP_STATUS_KEY.SCREEN_PASSED,
  [ApplicationStatus.Rejected]: APP_STATUS_KEY.REJECTED,
}

// HR-facing status labels (internal).
export const HR_STATUS_LABELS: Record<string, string> = {
  [APP_STATUS_KEY.APPLIED]: '待查看',
  [APP_STATUS_KEY.VIEWED]: '已查看',
  [APP_STATUS_KEY.SCREENING]: '筛选中',
  [APP_STATUS_KEY.SCREEN_PASSED]: '筛选通过',
  [APP_STATUS_KEY.INTERVIEW_PENDING]: '待面试',
  [APP_STATUS_KEY.INTERVIEWING]: '面试中',
  [APP_STATUS_KEY.INTERVIEW_PASSED]: '面试通过',
  [APP_STATUS_KEY.OFFER_PENDING]: '待发Offer',
  [APP_STATUS_KEY.OFFER_SENT]: 'Offer已发',
  [APP_STATUS_KEY.OFFER_ACCEPTED]: 'Offer已接受',
  [APP_STATUS_KEY.OFFER_REJECTED]: 'Offer被拒',
  [APP_STATUS_KEY.HIRED]: '已入职',
  [APP_STATUS_KEY.REJECTED]: '淘汰',
  [APP_STATUS_KEY.WITHDRAWN]: '候选人撤回',
}

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

// Elo-type tag mapping for status display (for element-plus el-tag).
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

// Terminal status keys (no outgoing transitions, except REJECTED allows HR re-pass).
export const TERMINAL_STATUS_KEYS: Set<string> = new Set([
  APP_STATUS_KEY.WITHDRAWN,
  APP_STATUS_KEY.OFFER_REJECTED,
  APP_STATUS_KEY.HIRED,
])

// HR action buttons and the statuses they are allowed FROM.
// Mirrors the server-side transition validator matrix.
export const ALLOWED_HR_ACTIONS: Record<string, Set<string>> = {
  [APP_STATUS_KEY.SCREEN_PASSED]: new Set([
    APP_STATUS_KEY.APPLIED,
    APP_STATUS_KEY.VIEWED,
    APP_STATUS_KEY.SCREENING,
    APP_STATUS_KEY.REJECTED, // HR re-pass: re-open as a new round
  ]),
  [APP_STATUS_KEY.REJECTED]: new Set([
    APP_STATUS_KEY.APPLIED,
    APP_STATUS_KEY.VIEWED,
    APP_STATUS_KEY.SCREENING,
    APP_STATUS_KEY.SCREEN_PASSED,
    APP_STATUS_KEY.INTERVIEW_PENDING,
    APP_STATUS_KEY.INTERVIEWING,
    APP_STATUS_KEY.INTERVIEW_PASSED,
    APP_STATUS_KEY.OFFER_PENDING,
    APP_STATUS_KEY.OFFER_SENT,
  ]),
  // Schedule interview: allowed from screen_passed, interview stages, and multi-round loop.
  [APP_STATUS_KEY.INTERVIEW_PENDING]: new Set([
    APP_STATUS_KEY.VIEWED,            // skip screening, go directly to interview
    APP_STATUS_KEY.SCREEN_PASSED,
    APP_STATUS_KEY.INTERVIEW_PENDING,  // reschedule after cancellation
    APP_STATUS_KEY.INTERVIEWING,       // reschedule after cancellation
    APP_STATUS_KEY.INTERVIEW_PASSED,   // multi-round interview (2nd, 3rd, ...)
  ]),
  // Mark interview as passed: allowed from interviewing or interview_pending.
  [APP_STATUS_KEY.INTERVIEW_PASSED]: new Set([
    APP_STATUS_KEY.VIEWED,             // skip intermediate stages
    APP_STATUS_KEY.INTERVIEWING,
    APP_STATUS_KEY.INTERVIEW_PENDING,  // all rounds complete, skip directly
  ]),
  // Advance to offer stage: allowed from interview_passed.
  [APP_STATUS_KEY.OFFER_PENDING]: new Set([
    APP_STATUS_KEY.VIEWED,             // skip intermediate stages
    APP_STATUS_KEY.INTERVIEW_PASSED,
  ]),
}

// ── Enums (legacy numeric) ─────────────────────────────────────────────

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

// ── Helpers ────────────────────────────────────────────────────────────

export function getHRStatusLabel(statusKey: string, fallback?: string): string {
  return HR_STATUS_LABELS[statusKey] || fallback || '未知'
}

export function getCandidateStatusLabel(statusKey: string, fallback?: string): string {
  return CANDIDATE_STATUS_LABELS[statusKey] || fallback || '未知'
}

export function getStatusType(statusKey: string, fallback?: string): string {
  return STATUS_TYPE_MAP[statusKey] || fallback || 'info'
}

// ── User ───────────────────────────────────────────────────────────────

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

// ── Job ────────────────────────────────────────────────────────────────

export interface Job {
  job_id: number
  title: string
  department: string
  department_id?: number
  location: string
  location_id?: number
  salary_range: string
  description: string
  requirements: string
  status?: number
  application_count?: number
  created_at?: string
  // camelCase fallbacks for API inconsistency
  jobId?: number
  departmentId?: number
  locationId?: number
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
  department_id?: number
  location?: string
  location_id?: number
  salary_range?: string
  description?: string
  requirements?: string
}

export type JobUpdatePayload = JobCreatePayload

export interface PaginatedList<T> {
  list: T[]
  total: number
}

// ── Application ────────────────────────────────────────────────────────

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
  file_name?: string
  file_type?: string
}

// ── Profile ────────────────────────────────────────────────────────────

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

// ── Resume ─────────────────────────────────────────────────────────────

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
}

export interface ConfirmPayload {
  oss_key: string
  file_name: string
  file_type: string
  file_size: number
}

// ── Admin / Invite Codes ───────────────────────────────────────────────

export interface InviteCodeInfo {
  id: number
  code: string
  created_by: number
  expires_at?: string
  is_active: number
  created_at: string
}

// ── Taxonomy ───────────────────────────────────────────────────────────

export interface DepartmentNode {
  id: number
  parent_id: number
  name: string
  full_name: string
  is_active: number
  sort_order: number
  depth: number
  children: DepartmentNode[]
}

export interface LocationOption {
  id: number
  name: string
  code: string
  is_active: number
  sort_order: number
}

export interface DepartmentLocationMapItem {
  department_id: number
  location_ids: number[]
}

// ── Staff User Management ───────────────────────────────────────────────

export interface StaffUserInfo {
  user_id: number
  username: string
  email: string
  status: string
  account_type: string
  roles: string[]
  token_version: number
  created_at: string
}

export interface RoleInfo {
  id: number
  role_key: string
  name: string
  description: string
  is_system: boolean
  created_at: string
  updated_at: string
}

export interface PermissionInfo {
  id: number
  permission_key: string
  resource: string
  action: string
  description: string
}

export interface DataScopeInfo {
  id: number
  scope_key: string
  resource_type: string
  resource_id: number
  assigned_at: string
}

export interface CreateStaffUserPayload {
  username: string
  password: string
  email?: string
  role_keys?: string[]
}

export interface StaffUserQuery {
  page: number
  page_size: number
  status?: string
}

export interface JobOptionsResponse {
  department_tree: DepartmentNode[]
  locations: LocationOption[]
  department_location_map?: DepartmentLocationMapItem[]
}

export interface DepartmentLocationConfig {
  department_id: number
  inherit_locations: number
  direct_location_ids: number[]
  effective_location_ids: number[]
  locations: LocationOption[]
  available_location_ids: number[]
}

// ── Interview ──────────────────────────────────────────────────────────────

export interface InterviewSchedule {
  interview_id: number
  application_id: number
  interviewer_id: number
  round_no: number
  title: string
  mode: string          // video / phone / onsite
  meeting_url: string
  location: string
  duration_minutes: number
  candidate_note: string
  internal_note: string
  cancel_reason: string
  scheduled_at: string
  status: string        // pending / scheduled / completed / cancelled
  created_by: number
  created_at: string
  updated_at: string
  interviewer_name: string
  application_status_key: string
  job_title: string
  candidate_name: string
  candidate_phone: string
}

// ── Offer ──────────────────────────────────────────────────────────────────

export interface Offer {
  id: number
  application_id: number
  candidate_user_id: number
  job_id: number
  status: string              // draft / sent / accepted / rejected / withdrawn
  title: string
  salary_range: string
  level: string
  work_location: string
  start_date: string
  expires_at: string
  terms_json: string
  sent_snapshot_json: string
  created_by: number
  sent_by: number
  decided_at: string
  created_at: string
  updated_at: string
  job_title: string
  candidate_name: string
  application_status_key: string
  created_by_name: string
  sent_by_name: string
}

export interface OfferEvent {
  id: number
  offer_id: number
  event_type: string
  actor_user_id: number
  actor_account_type: string
  reason: string
  metadata_json: string
  created_at: string
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

// ── Interview feedback display constants ─────────────────────────────

export const RECOMMENDATION_LABEL: Record<string, string> = {
  strong_recommend: '强烈推荐',
  recommend: '推荐通过',
  neutral: '待定',
  not_recommend: '不推荐',
  strong_not_recommend: '强烈不推荐',
}

export const RECOMMENDATION_TYPE: Record<string, string> = {
  strong_recommend: 'success',
  recommend: 'success',
  neutral: 'warning',
  not_recommend: 'danger',
  strong_not_recommend: 'danger',
}

export const DIMENSION_LABELS: Record<string, string> = {
  professional: '专业能力',
  communication: '沟通表达',
  problem_solving: '问题分析',
  job_fit: '岗位匹配',
  potential: '发展潜力',
}

export interface TimelineEventInfo {
  id: string
  event_type: string  // status_transition | interview | offer | note
  title: string
  description: string
  timestamp: string
  actor_name: string
  application_id: number
}

export interface CandidateWorkspaceInterview {
  interview_id: number
  application_id: number
  title: string
  mode: string
  status: string
  scheduled_at: string
  interviewer_name: string
  job_title: string
  round_no: number
  // Feedback fields
  has_feedback: boolean
  feedback_recommendation: string
  feedback_score: number
  feedback_dimension_scores_json: string
  feedback_comments: string
}

export interface CandidateWorkspaceOffer {
  offer_id: number
  application_id: number
  title: string
  status: string
  salary_range: string
  level: string
  work_location: string
  start_date: string
  job_title: string
}

// ── Phase 4: Collaboration ────────────────────────────────────────────

export interface CandidateWorkspace {
  real_name: string
  phone: string
  education: string
  school: string
  work_experience: string
  skills: string[]
  applications: CandidateWorkspaceApplication[]
  tags: CandidateTagInfo[]
  total_applications: number
  total_interviews: number
  total_offers: number
  latest_activity_at: string
  resume_url: string
  interviews: CandidateWorkspaceInterview[]
  offers: CandidateWorkspaceOffer[]
}

export interface CandidateWorkspaceApplication {
  application_id: number
  job_id: number
  job_title: string
  department: string
  location: string
  status_key: string
  round_no: number
  is_current: number
  applied_at: string
}

export interface CandidateNoteInfo {
  id: number
  candidate_user_id: number
  application_id: number
  author_user_id: number
  content: string
  visibility: string
  created_at: string
  updated_at: string
  author_name: string
}

export interface CandidateTagInfo {
  id: number
  name: string
  color: string
}

export interface FollowUpTaskInfo {
  id: number
  candidate_user_id: number
  application_id: number
  assignee_user_id: number
  created_by: number
  title: string
  description: string
  due_at: string
  status: string
  completed_at: string
  created_at: string
  updated_at: string
  assignee_name: string
  candidate_name: string
}
