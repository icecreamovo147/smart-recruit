import request from './request'
import type {
  CandidateComparisonItem,
  CandidateMatchEvaluationSnapshotInfo,
  ResumeProfileSnapshotInfo,
} from '@/types/recruitingIntelligence'
import { debugLog } from '@/utils/debugLog'

export const getApplicationResumeProfile = async (
  applicationId: number,
  options: { silentError?: boolean } = {},
): Promise<{ profile: ResumeProfileSnapshotInfo | null }> => {
  debugLog.ri.info('getApplicationResumeProfile_started', { application_id: applicationId, silent_error: options.silentError })
  try {
    const res = await request.get<{ profile: ResumeProfileSnapshotInfo | null }>(`/api/v1/hr/applications/${applicationId}/resume-profile`, options)
    debugLog.ri.info('getApplicationResumeProfile_succeeded', { application_id: applicationId, has_profile: !!res?.profile, profile_id: (res?.profile as any)?.profile?.id ?? null })
    return res
  } catch (err: any) {
    debugLog.ri.warn('getApplicationResumeProfile_failed', { application_id: applicationId, error: err?.message })
    throw err
  }
}

export const getResumeProfile = async (params: {
  application_id?: number
  resume_id?: number
  profile_id?: number
}): Promise<{ profile: ResumeProfileSnapshotInfo | null }> => {
  debugLog.ri.info('getResumeProfile_started', params)
  const res = await request.get<{ profile: ResumeProfileSnapshotInfo | null }>('/api/v1/hr/resume-profiles', { params })
  debugLog.ri.info('getResumeProfile_succeeded', { ...params, has_profile: !!res?.profile })
  return res
}

export const parseResumeProfile = async (data: {
  application_id?: number
  resume_id?: number
}): Promise<{ profile: ResumeProfileSnapshotInfo | null }> => {
  debugLog.ri.info('parseResumeProfile_started', data)
  const res = await request.post<{ profile: ResumeProfileSnapshotInfo | null }>('/api/v1/hr/resume-profiles/parse', data)
  const profileObj = (res?.profile as any)?.profile
  debugLog.ri.info('parseResumeProfile_succeeded', { ...data, profile_id: profileObj?.id ?? null, version: profileObj?.version ?? null })
  return res
}

export const evaluateCandidateMatch = async (
  applicationId: number,
  data: { agent_run_id?: number } = {},
): Promise<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }> => {
  debugLog.ri.info('evaluateCandidateMatch_started', { application_id: applicationId, agent_run_id: data.agent_run_id })
  const res = await request.post<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }>(`/api/v1/hr/applications/${applicationId}/match-evaluations`, data)
  const evalInfo = (res?.evaluation as any)?.evaluation
  debugLog.ri.info('evaluateCandidateMatch_succeeded', {
    application_id: applicationId,
    evaluation_id: evalInfo?.id ?? null,
    overall_score: evalInfo?.overall_score ?? null,
    recommendation: evalInfo?.recommendation ?? null,
  })
  return res
}

export const getCandidateMatchEvaluation = async (
  applicationId: number,
  params?: { evaluation_id?: number; evaluation_version?: number },
  options: { silentError?: boolean } = {},
): Promise<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }> => {
  debugLog.ri.info('getCandidateMatchEvaluation_started', { application_id: applicationId, ...params })
  try {
    const res = await request.get<{ evaluation: CandidateMatchEvaluationSnapshotInfo | null }>(`/api/v1/hr/applications/${applicationId}/match-evaluation`, { params, ...options })
    debugLog.ri.info('getCandidateMatchEvaluation_succeeded', { application_id: applicationId, has_evaluation: !!res?.evaluation })
    return res
  } catch (err: any) {
    debugLog.ri.warn('getCandidateMatchEvaluation_failed', { application_id: applicationId, error: err?.message })
    throw err
  }
}

export const compareCandidatesForJob = async (
  jobId: number,
): Promise<{ job_id: number; candidates: CandidateComparisonItem[]; missing_application_ids: number[] }> => {
  debugLog.ri.info('compareCandidatesForJob_started', { job_id: jobId })
  const res = await request.get<{ job_id: number; candidates: CandidateComparisonItem[]; missing_application_ids: number[] }>(`/api/v1/hr/jobs/${jobId}/candidate-comparison`)
  debugLog.ri.info('compareCandidatesForJob_succeeded', { job_id: jobId, candidate_count: res?.candidates?.length ?? 0, missing_count: res?.missing_application_ids?.length ?? 0 })
  return res
}
