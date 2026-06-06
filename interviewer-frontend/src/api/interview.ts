import request from './request'
import type { InterviewSchedule, InterviewFeedback, FeedbackPayload } from '@/types/domain'

export const listMyInterviews = (params?: { status?: string }): Promise<{ list: InterviewSchedule[] }> =>
  request.get('/api/v1/hr/my-interviews', { params })

export const getInterview = (interviewId: number): Promise<InterviewSchedule> =>
  request.get(`/api/v1/hr/interviews/${interviewId}`).then((res: any) => res.interview)

export const submitFeedback = (interviewId: number, data: FeedbackPayload): Promise<any> =>
  request.post(`/api/v1/hr/interviews/${interviewId}/feedback`, data)

export const getFeedback = (interviewId: number): Promise<InterviewFeedback | null> =>
  request.get(`/api/v1/hr/interviews/${interviewId}/feedback`).then((res: any) => res?.feedback ?? null)
