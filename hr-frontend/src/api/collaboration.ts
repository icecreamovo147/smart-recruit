import request from './request'
import type { CandidateWorkspace, CandidateNoteInfo, CandidateTagInfo, FollowUpTaskInfo, TimelineEventInfo } from '@/types/domain'

// ── Workspace ─────────────────────────────────────────────────────────

export const getCandidateWorkspace = (candidateUserId: number): Promise<{ workspace: CandidateWorkspace }> =>
  request.get(`/api/v1/hr/candidates/${candidateUserId}/workspace`)

// ── Notes ─────────────────────────────────────────────────────────────

export const createNote = (data: {
  candidate_user_id: number
  application_id?: number
  content: string
}): Promise<{ note: CandidateNoteInfo }> =>
  request.post('/api/v1/hr/notes', data)

export const listNotes = (candidateUserId: number, applicationId?: number): Promise<{ list: CandidateNoteInfo[] }> => {
  const params: Record<string, string | number> = { candidate_user_id: candidateUserId }
  if (applicationId) { params.application_id = applicationId }
  return request.get('/api/v1/hr/notes', { params })
}

// ── Tags ──────────────────────────────────────────────────────────────

export const createTag = (data: { name: string; color?: string }): Promise<{ tag: CandidateTagInfo }> =>
  request.post('/api/v1/hr/tags', data)

export const listTags = (): Promise<{ list: CandidateTagInfo[] }> =>
  request.get('/api/v1/hr/tags')

export const assignTag = (data: { tag_id: number; candidate_user_id: number }): Promise<void> =>
  request.post('/api/v1/hr/tags/assign', data)

export const unassignTag = (data: { tag_id: number; candidate_user_id: number }): Promise<void> =>
  request.post('/api/v1/hr/tags/unassign', data)

export const listCandidateTags = (candidateUserId: number): Promise<{ list: CandidateTagInfo[] }> =>
  request.get(`/api/v1/hr/candidates/${candidateUserId}/tags`)

// ── Follow-up Tasks ───────────────────────────────────────────────────

export const createFollowUpTask = (data: {
  candidate_user_id: number
  application_id?: number
  assignee_user_id: number
  title: string
  description?: string
  due_at?: string
}): Promise<{ task: FollowUpTaskInfo }> =>
  request.post('/api/v1/hr/follow-up-tasks', data)

export const listFollowUpTasks = (params?: {
  candidate_user_id?: number
  assignee_user_id?: number
  status?: string
}): Promise<{ list: FollowUpTaskInfo[] }> =>
  request.get('/api/v1/hr/follow-up-tasks', { params })

export const completeFollowUpTask = (taskId: number): Promise<void> =>
  request.patch(`/api/v1/hr/follow-up-tasks/${taskId}/complete`)

// ── Timeline Events ────────────────────────────────────────────────────

export const listTimelineEvents = (candidateUserId: number): Promise<{ events: TimelineEventInfo[] }> =>
  request.get(`/api/v1/hr/candidates/${candidateUserId}/timeline`)
