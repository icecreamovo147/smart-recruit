import request from './request'
import type { Job, JobCreatePayload, JobOptionsResponse, JobQuery, JobUpdatePayload, PaginatedList } from '@/types/domain'

export const getJobOptions = (): Promise<JobOptionsResponse> =>
  request.get('/api/v1/hr/job-options')

export const createJob = (data: JobCreatePayload): Promise<{ job_id: number }> =>
  request.post('/api/v1/hr/jobs', data)

export const updateJob = (jobId: number, data: JobUpdatePayload): Promise<void> =>
  request.put(`/api/v1/hr/jobs/${jobId}`, data)

export const offlineJob = (jobId: number): Promise<void> =>
  request.patch(`/api/v1/hr/jobs/${jobId}/offline`)

export const onlineJob = (jobId: number): Promise<void> =>
  request.patch(`/api/v1/hr/jobs/${jobId}/online`)

export const listHRJobs = (params: JobQuery): Promise<PaginatedList<Job>> =>
  request.get('/api/v1/hr/jobs', { params })
