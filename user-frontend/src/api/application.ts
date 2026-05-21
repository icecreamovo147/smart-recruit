import request from './request'
import type { Application, JobQuery, PaginatedList } from '@/types/domain'

export const applyJob = (data: { job_id: number }): Promise<void> =>
  request.post('/api/v1/candidate/applications', data)

export const listMyApplications = (params: JobQuery): Promise<PaginatedList<Application>> =>
  request.get('/api/v1/candidate/applications', { params })
