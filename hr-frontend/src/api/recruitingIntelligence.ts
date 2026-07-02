import request from './request'
import type {
  CandidateComparisonItem,
  CandidateMatchEvaluationSnapshotInfo,
  ResumeProfileSnapshotInfo,
} from '@/types/recruitingIntelligence'

export const getApplicationResumeProfile = (
  applicationId: number,
): Promise<{ profile: ResumeProfileSnapshotInfo | null }> =>
  request.get(`/api/v1/hr/applications/${applicationId}/resume-profile`)

export const getResumeProfile = (params: {
  application_id?: number
  resume_id?: number
  profile_id?: number
}): Promise<{ profile: ResumeProfileSnapshotInfo | null }> =>
  request.get('/api/v1/hr/resume-profiles', { params })

export const parseResumeProfile = (data: {
  application_id?: number
  resume_id?: number
}): Promise<{ profile: ResumeProfileSnapshotInfo | null }> =>
  request.post('/api/v1/hr/resume-profiles/parse', data)

export const evaluateCandidateMatch = (
  applicationId: number,
  data: { agent_run_id?: number } = {},
): Promise<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }> =>
  request.post(`/api/v1/hr/applications/${applicationId}/match-evaluations`, data)

export const getCandidateMatchEvaluation = (
  applicationId: number,
  params?: { evaluation_id?: number; evaluation_version?: number },
): Promise<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }> =>
  request.get(`/api/v1/hr/applications/${applicationId}/match-evaluation`, { params })

export const compareCandidatesForJob = (
  jobId: number,
): Promise<{ job_id: number; candidates: CandidateComparisonItem[]; missing_application_ids: number[] }> =>
  request.get(`/api/v1/hr/jobs/${jobId}/candidate-comparison`)
