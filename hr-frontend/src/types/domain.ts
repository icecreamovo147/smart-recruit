// ---- Enums / Constants ----

export const ROLE_CANDIDATE = 1
export const ROLE_HR = 2
export const ROLE_HR_ADMIN = 3

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

// ---- User ----

export interface User {
  user_id: number
  role: number
  username: string
}

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user_id: number
  role: number
  username: string
}

export interface RegisterPayload {
  username: string
  password: string
  email?: string
  role: number
  invite_code?: string
}

// ---- Job ----

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

// ---- Application ----

export interface Application {
  application_id: number
  job_id: number
  job_title: string
  status: number
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
}

export interface ConfirmPayload {
  oss_key: string
  file_name: string
  file_type: string
  file_size: number
}

// ---- Admin / Invite Codes ----

export interface InviteCodeInfo {
  id: number
  code: string
  created_by: number
  expires_at?: string
  is_active: number
  created_at: string
}

// ---- Taxonomy ----

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
