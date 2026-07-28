import request from './http'
import type { PaginatedList } from '@shared/types/pagination'
import type {
  AgentSkillDetailResponse,
  AgentSkillInfo,
  AgentSkillListParams,
  AgentSkillPreviewPayload,
  AgentSkillPreviewResult,
  AgentSkillResponse,
  AgentSkillVersionListResponse,
  AgentSkillVersionResponse,
  AvailableAgentSkill,
  CreateAgentSkillPayload,
  CreateAgentSkillVersionPayload,
  SemanticRetrievalDebugParams,
  SemanticRetrievalDebugResult,
  UpdateAgentSkillPayload,
  UpdateAgentSkillStatusPayload,
} from '@shared/types/agentSkill'
import { debugLog } from '@shared/utils/debugLog'

export const listAgentSkills = async (
  params: AgentSkillListParams = {},
): Promise<PaginatedList<AgentSkillInfo>> => {
  debugLog.skill.info('listAgentSkills_started', { page: params.page, page_size: params.page_size, keyword: params.keyword })
  const res = await request.get<PaginatedList<AgentSkillInfo>>('/api/v1/platform/ai/agent-skills', {
    params: {
      page: params.page ?? 1,
      page_size: params.page_size ?? 20,
      ...(params.keyword ? { keyword: params.keyword } : {}),
      ...(params.enabled_only !== undefined ? { enabled_only: params.enabled_only ? 'true' : 'false' } : {}),
    },
  })
  debugLog.skill.info('listAgentSkills_succeeded', { total: res.total || 0, returned: (res.list || []).length })
  return res
}

export const previewAgentSkill = async (
  data: AgentSkillPreviewPayload,
): Promise<AgentSkillPreviewResult> => {
  debugLog.skill.info('previewAgentSkill_started', { skill_name: data.package.manifest.skill_name })
  const res = await request.post<AgentSkillPreviewResult>('/api/v1/platform/ai/agent-skills/preview', data)
  debugLog.skill.info('previewAgentSkill_finished', {
    compiled_hash: res.package.compiled_hash,
    package_estimated_tokens: res.package.package_estimated_tokens,
  })
  return res
}

export const getAgentSkill = async (
  id: number,
): Promise<AgentSkillDetailResponse> => {
  debugLog.skill.info('getAgentSkill_started', { skill_id: id })
  const res = await request.get<AgentSkillDetailResponse>(`/api/v1/platform/ai/agent-skills/${id}`)
  debugLog.skill.info('getAgentSkill_succeeded', { skill_id: id, has_detail: !!res })
  return res
}

export const createAgentSkill = async (
  data: CreateAgentSkillPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('createAgentSkill_started', {
    name: data.package.manifest.skill_name,
    display_name: data.package.manifest.display_name,
    version: data.version,
  })
  const res = await request.post<AgentSkillResponse>('/api/v1/platform/ai/agent-skills', data)
  debugLog.skill.info('createAgentSkill_succeeded', {
    name: data.package.manifest.skill_name,
    skill_id: res.skill.id,
  })
  return res
}

export const updateAgentSkill = async (
  id: number,
  data: UpdateAgentSkillPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('updateAgentSkill_started', { skill_id: id })
  const res = await request.put<AgentSkillResponse>(`/api/v1/platform/ai/agent-skills/${id}`, data)
  debugLog.skill.info('updateAgentSkill_succeeded', { skill_id: id })
  return res
}

export const updateAgentSkillStatus = async (
  id: number,
  data: UpdateAgentSkillStatusPayload,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('updateAgentSkillStatus_started', { skill_id: id, is_enabled: data.is_enabled })
  const res = await request.patch<AgentSkillResponse>(`/api/v1/platform/ai/agent-skills/${id}/status`, data)
  debugLog.skill.info('updateAgentSkillStatus_succeeded', { skill_id: id, is_enabled: data.is_enabled })
  return res
}

export const regenerateAgentSkillVersionEmbedding = async (
  versionId: number,
): Promise<{ success_count: number; failed_count: number; skipped_count: number }> => {
  debugLog.skill.info('regenerateAgentSkillVersionEmbedding_started', { version_id: versionId })
  const res = await request.post<{ success_count: number; failed_count: number; skipped_count: number }>(
    `/api/v1/platform/ai/agent-skills/${versionId}/embedding/regenerate`,
  )
  debugLog.skill.info('regenerateAgentSkillVersionEmbedding_succeeded', {
    version_id: versionId,
    success_count: res.success_count,
    failed_count: res.failed_count,
    skipped_count: res.skipped_count,
  })
  return res
}

export const listAgentSkillVersions = async (
  id: number,
): Promise<AgentSkillVersionListResponse> => {
  debugLog.skill.info('listAgentSkillVersions_started', { skill_id: id })
  const res = await request.get<AgentSkillVersionListResponse>(`/api/v1/platform/ai/agent-skills/${id}/versions`)
  debugLog.skill.info('listAgentSkillVersions_succeeded', { skill_id: id, version_count: (res.list || []).length })
  return res
}

export const createAgentSkillVersion = async (
  id: number,
  data: CreateAgentSkillVersionPayload,
): Promise<AgentSkillVersionResponse> => {
  debugLog.skill.info('createAgentSkillVersion_started', { skill_id: id, version: data.version })
  const res = await request.post<AgentSkillVersionResponse>(`/api/v1/platform/ai/agent-skills/${id}/versions`, data)
  debugLog.skill.info('createAgentSkillVersion_succeeded', { skill_id: id, version_id: res.version.id })
  return res
}

export const activateAgentSkillVersion = async (
  id: number,
  versionId: number,
): Promise<AgentSkillResponse> => {
  debugLog.skill.info('activateAgentSkillVersion_started', { skill_id: id, version_id: versionId })
  const res = await request.post<AgentSkillResponse>(`/api/v1/platform/ai/agent-skills/${id}/versions/${versionId}/activate`)
  debugLog.skill.info('activateAgentSkillVersion_succeeded', { skill_id: id, version_id: versionId })
  return res
}

export const listAvailableAgentSkills = async (): Promise<{ list: AvailableAgentSkill[] }> => {
  debugLog.skill.info('listAvailableAgentSkills_started', {})
  const res = await request.get<{ list: AvailableAgentSkill[] }>('/api/v1/hr/agent-skills/available')
  debugLog.skill.info('listAvailableAgentSkills_succeeded', { count: (res.list || []).length })
  return res
}

export const debugSemanticRetrieval = async (
  params: SemanticRetrievalDebugParams,
): Promise<SemanticRetrievalDebugResult> => {
  debugLog.skill.info('debugSemanticRetrieval_started', { query_length: params.query.length, agent_type: params.agent_type })
  const res = await request.get<SemanticRetrievalDebugResult>('/api/v1/platform/ai/agent-skills/semantic-debug', {
    params: {
      query: params.query,
      ...(params.agent_type ? { agent_type: params.agent_type } : {}),
      ...(params.job_id ? { job_id: params.job_id } : {}),
      ...(params.application_id ? { application_id: params.application_id } : {}),
      ...(params.limit ? { limit: params.limit } : {}),
      ...(params.owner_role ? { owner_role: params.owner_role } : {}),
      ...(params.owner_id ? { owner_id: params.owner_id } : {}),
    },
  })
  debugLog.skill.info('debugSemanticRetrieval_succeeded', { skill_count: (res.skills || []).length, memory_count: (res.memories || []).length })
  return res
}
