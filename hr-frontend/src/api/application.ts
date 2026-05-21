import request from './request'
import type { Application, JobQuery, PaginatedList } from '@/types/domain'

export const listJobApplications = (jobId: number, params: JobQuery): Promise<PaginatedList<Application>> =>
  request.get(`/api/v1/hr/jobs/${jobId}/applications`, { params })

export const updateApplicationStatus = (applicationId: number, status: number): Promise<void> =>
  request.patch(`/api/v1/hr/applications/${applicationId}/status`, { status })
