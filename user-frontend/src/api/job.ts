import request from './request'
import type { Job, JobOptionsResponse, JobQuery, PaginatedList } from '@/types/domain'

export const listJobs = (params: JobQuery): Promise<PaginatedList<Job>> => {
  const query: Record<string, string | number | undefined> = {
    page: params.page,
    page_size: params.page_size,
    keyword: params.keyword || undefined,
  }
  // Axios serializes arrays as department_ids[]= which Gin may not parse as QueryArray;
  // send comma-separated lists instead.
  if (params.department_ids?.length) {
    query.department_ids = params.department_ids.join(',')
  }
  if (params.location_ids?.length) {
    query.location_ids = params.location_ids.join(',')
  }
  return request.get('/api/v1/jobs', { params: query })
}

export const getJobDetail = (jobId: number): Promise<Job> =>
  request.get(`/api/v1/jobs/${jobId}`)

/** Public taxonomy for job-board filters (department tree + locations). */
export const getJobOptions = (): Promise<JobOptionsResponse> =>
  request.get('/api/v1/job-options')
