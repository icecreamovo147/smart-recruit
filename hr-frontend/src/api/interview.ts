import request from './request'
import type { InterviewSchedule, InterviewFeedback, StaffUserInfo } from '@/types/domain'

export interface ScheduleInterviewPayload {
  application_id: number
  interviewer_id: number
  round_no?: number
  title?: string
  mode?: string
  meeting_url?: string
  location?: string
  duration_minutes?: number
  candidate_note?: string
  internal_note?: string
  scheduled_at?: string
}

export interface UpdateInterviewPayload {
  title?: string
  mode?: string
  meeting_url?: string
  location?: string
  duration_minutes?: number
  candidate_note?: string
  internal_note?: string
  scheduled_at?: string
}

export interface SubmitFeedbackPayload {
  application_id: number
  recommendation: string
  score?: number
  dimension_scores_json?: string
  comments?: string
}

export interface InterviewerQuery {
  page: number
  page_size: number
  keyword?: string
}

export const scheduleInterview = (data: ScheduleInterviewPayload): Promise<{ interview_id: number }> =>
  request.post('/api/v1/hr/interviews', data)

export const updateInterview = (interviewId: number, data: UpdateInterviewPayload): Promise<void> =>
  request.put(`/api/v1/hr/interviews/${interviewId}`, data)

export const cancelInterview = (interviewId: number, cancelReason?: string): Promise<void> =>
  request.patch(`/api/v1/hr/interviews/${interviewId}/cancel`, { cancel_reason: cancelReason || '' })

export const getInterview = (interviewId: number): Promise<{ interview: InterviewSchedule }> =>
  request.get(`/api/v1/hr/interviews/${interviewId}`)

export const listInterviewers = (params: InterviewerQuery): Promise<{ total: number; list: StaffUserInfo[] }> =>
  request.get('/api/v1/hr/interviewers', { params })

export const listApplicationInterviews = (applicationId: number): Promise<{ list: InterviewSchedule[] }> =>
  request.get(`/api/v1/hr/applications/${applicationId}/interviews`)

export const listMyInterviews = (status?: string): Promise<{ list: InterviewSchedule[] }> =>
  request.get('/api/v1/hr/my-interviews', { params: { status } })

export const submitFeedback = (interviewId: number, data: SubmitFeedbackPayload): Promise<void> =>
  request.post(`/api/v1/hr/interviews/${interviewId}/feedback`, data)

export const getFeedback = (interviewId: number): Promise<{ feedback: InterviewFeedback }> =>
  request.get(`/api/v1/hr/interviews/${interviewId}/feedback`)
