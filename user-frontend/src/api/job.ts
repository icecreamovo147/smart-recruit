import request from './request'
import type { Job, JobQuery, PaginatedList } from '@/types/domain'

export const listJobs = (params: JobQuery): Promise<PaginatedList<Job>> =>
  request.get('/api/v1/jobs', { params })

export const getJobDetail = (jobId: number): Promise<Job> =>
  request.get(`/api/v1/jobs/${jobId}`)
