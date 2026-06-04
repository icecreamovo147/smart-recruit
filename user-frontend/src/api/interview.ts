import request from './request'
import type { InterviewSchedule } from '@/types/domain'

export const listMyInterviews = (): Promise<{ list: InterviewSchedule[] }> =>
  request.get('/api/v1/candidate/interviews')
